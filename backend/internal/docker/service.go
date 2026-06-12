package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type Service struct {
	workDir string
}

func NewService(workDir string) *Service {
	return &Service{workDir: workDir}
}

func (s *Service) BuildImage(ctx context.Context, buildDir, imageTag string, logWriter io.Writer) error {
	dockerfile := filepath.Join(buildDir, "Dockerfile")
	if _, err := os.Stat(dockerfile); os.IsNotExist(err) {
		return fmt.Errorf("Dockerfile not found in %s", buildDir)
	}

	cmd := exec.CommandContext(ctx,
		"docker", "build",
		"-t", imageTag,
		"-f", dockerfile,
		buildDir,
	)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	return cmd.Run()
}

func (s *Service) RunContainer(ctx context.Context, imageTag, containerName, repoPath, logDir string, port int, envVars map[string]string) (string, error) {
	args := []string{
		"run", "-d",
		"--name", containerName,
		"-p", fmt.Sprintf("%d:3000", port),
		"-v", fmt.Sprintf("%s:/workspace", repoPath),
		"-v", fmt.Sprintf("%s:/logs", logDir),
		"-e", fmt.Sprintf("WORKSPACE_DIR=/workspace"),
	}

	for k, v := range envVars {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, imageTag)

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker run failed: %s: %w", string(output), err)
	}

	return string(output), nil
}

func (s *Service) StopContainer(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, "docker", "stop", containerID)
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = exec.CommandContext(ctx, "docker", "rm", containerID)
	return cmd.Run()
}

func (s *Service) GetContainerPort(containerID string) string {
	cmd := exec.Command("docker", "port", containerID, "3000")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return string(output)
}
