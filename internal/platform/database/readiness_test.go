package database

import (
	"context"
	"errors"
	"testing"
)

type pingerStub struct {
	err error
}

func (s pingerStub) Ping(context.Context) error {
	return s.err
}

func TestCheckerReturnsDependencyFailure(t *testing.T) {
	want := errors.New("postgres unavailable")
	checker := Checker{Pinger: pingerStub{err: want}}

	got := checker.Ready(context.Background())

	if !errors.Is(got, want) {
		t.Fatalf("Ready() error = %v, want %v", got, want)
	}
}

func TestCheckerReportsReadyAfterSuccessfulPing(t *testing.T) {
	checker := Checker{Pinger: pingerStub{}}

	if err := checker.Ready(context.Background()); err != nil {
		t.Fatalf("Ready() error = %v, want nil", err)
	}
}

func TestOpenRejectsEmptyDatabaseURL(t *testing.T) {
	pool, err := Open(context.Background(), "")

	if err == nil {
		t.Fatal("Open() error = nil, want configuration error")
	}
	if pool != nil {
		t.Fatal("Open() returned a pool for an empty URL")
	}
}
