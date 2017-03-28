package codebuild

import (
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/remind101/conveyor/builder"
	"golang.org/x/net/context"
)

// Builder is a builder.Builder implementation that runs the build in a docker
type Builder struct {
	// Client for interacting with CodeBuild API
	codebuild codebuildClient

	// Client for interacting with CloudWatch Logs API.
	cloudwatchlogs cloudwatchlogsClient
}

// NewBuilder returns a new Builder backed by the docker client.
func NewBuilder() *Builder {
	return &Builder{}
}

// Build executes the docker image.
func (b *Builder) Build(ctx context.Context, w io.Writer, opts builder.BuildOptions) (string, error) {
	projectName := fmt.Sprintf("conveyor-%s", opts.Repository)

	// Start build
	resp, err := b.codebuild.StartBuild(codebuild.StartBuildInput{
		ProjectName:   aws.String(projectName),
		SourceVersion: aws.String(opts.Sha),
	})
	if err != nil {
		// TODO: if the error was because the project doesn't exist,
		// create it and retry.
		return "", fmt.Errorf("unable to start build: %v", err)
	}
	build := resp.Build

	logstream, err := cloudwatch.NewGroup(*build.Logs.GroupName, b.cloudwatchlogs).Open(*build.Logs.StreamName)
	if err != nil {
		return "", fmt.Errorf("unable to open log stream: %v", err)
	}

	// Copy the log stream to w.
	if _, err := io.Copy(w, logstream); err != nil {
		return "", fmt.Errorf("unable to stream logs: %v", err)
	}

	return fmt.Sprintf("%s:%s", opts.Repository, opts.Sha), nil
}
