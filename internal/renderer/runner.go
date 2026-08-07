package renderer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const maxPDFBytes = 32 << 20

type Runner interface {
	Render(context.Context, string) ([]byte, error)
}

type CLIRunner struct {
	tempRoot  string
	timeout   time.Duration
	semaphore chan struct{}
}

func NewCLIRunner(tempRoot string, maxConcurrent int, timeout time.Duration) (*CLIRunner, error) {
	if maxConcurrent < 1 {
		return nil, errors.New("maximum concurrency must be positive")
	}
	if timeout <= 0 {
		return nil, errors.New("render timeout must be positive")
	}
	return &CLIRunner{
		tempRoot:  tempRoot,
		timeout:   timeout,
		semaphore: make(chan struct{}, maxConcurrent),
	}, nil
}

func (r *CLIRunner) Render(parent context.Context, source string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, r.timeout)
	defer cancel()

	select {
	case r.semaphore <- struct{}{}:
		defer func() { <-r.semaphore }()
	case <-ctx.Done():
		return nil, fmt.Errorf("wait for render capacity: %w", ctx.Err())
	}

	temporaryDirectory, err := os.MkdirTemp(r.tempRoot, "etpeto-render-*")
	if err != nil {
		return nil, fmt.Errorf("create render directory: %w", err)
	}
	defer os.RemoveAll(temporaryDirectory)

	inputPath := filepath.Join(temporaryDirectory, "document.typ")
	outputPath := filepath.Join(temporaryDirectory, "document.pdf")
	if err := os.WriteFile(inputPath, []byte(source), 0o600); err != nil {
		return nil, fmt.Errorf("write render source: %w", err)
	}

	command := exec.CommandContext(ctx, "typst", "compile", "--format", "pdf", inputPath, outputPath)
	command.Dir = temporaryDirectory
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("Typst render deadline: %w", ctx.Err())
		}
		return nil, errors.New("Typst compilation failed")
	}

	file, err := os.Open(outputPath)
	if err != nil {
		return nil, fmt.Errorf("open rendered PDF: %w", err)
	}
	defer file.Close()

	pdf, err := io.ReadAll(io.LimitReader(file, maxPDFBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read rendered PDF: %w", err)
	}
	if len(pdf) > maxPDFBytes {
		return nil, errors.New("rendered PDF exceeds size limit")
	}
	return pdf, nil
}
