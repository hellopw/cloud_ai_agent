package git

import (
	"context"
	"fmt"
	"os/exec"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Clone(ctx context.Context, repoURL, branch, targetDir string) error {
	args := []string{"clone", "--depth", "1"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, repoURL, targetDir)

	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %s: %w", string(output), err)
	}
	return nil
}
