package git

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Clone(ctx context.Context, repoURL, branch, targetDir string, username, password string, logWriter io.Writer) error {
	cloneURL := repoURL

	// HTTPS to code.sohuno.com hangs; convert to SSH which is pre-configured
	cloneURL = tryConvertToSSH(cloneURL)

	if username != "" && password != "" && !strings.HasPrefix(cloneURL, "git@") {
		cloneURL = embedCredentials(cloneURL, username, password)
	}

	if branch != "" {
		// Try branch first; capture output manually to avoid CombinedOutput issues
		branchArgs := []string{"clone", "--branch", branch, cloneURL, targetDir}
		cmd := exec.CommandContext(ctx, "git", branchArgs...)
		var buf bytes.Buffer
		cmd.Stdout = io.MultiWriter(&buf, logWriter)
		cmd.Stderr = io.MultiWriter(&buf, logWriter)
		err := cmd.Run()
		output := buf.Bytes()
		if err == nil {
			return nil
		}
		if strings.Contains(string(output), "Remote branch") && strings.Contains(string(output), "not found") {
			os.RemoveAll(targetDir)
		} else {
			return fmt.Errorf("git clone failed: %s: %w", string(output), err)
		}
	}

	args := []string{"clone", cloneURL, targetDir}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}
	return nil
}

// tryConvertToSSH converts known HTTPS git hosts to SSH when SSH keys are configured.
func tryConvertToSSH(rawURL string) string {
	if strings.HasPrefix(rawURL, "https://code.sohuno.com/") {
		rest := strings.TrimPrefix(rawURL, "https://code.sohuno.com/")
		rest = strings.TrimSuffix(rest, ".git")
		return "git@code.sohuno.com:" + rest + ".git"
	}
	return rawURL
}

func embedCredentials(rawURL, username, password string) string {
	if strings.HasPrefix(rawURL, "git@") {
		return rawURL
	}
	if strings.HasPrefix(rawURL, "https://") {
		rest := strings.TrimPrefix(rawURL, "https://")
		return fmt.Sprintf("https://%s:%s@%s", url.PathEscape(username), url.PathEscape(password), rest)
	}
	if strings.HasPrefix(rawURL, "http://") {
		rest := strings.TrimPrefix(rawURL, "http://")
		return fmt.Sprintf("http://%s:%s@%s", url.PathEscape(username), url.PathEscape(password), rest)
	}
	return rawURL
}