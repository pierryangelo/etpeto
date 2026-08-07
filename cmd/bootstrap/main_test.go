package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunRefusesToStartWithoutADatabaseURL(t *testing.T) {
	code := run(environment{
		getenv: staticEnvironment(map[string]string{}),
		stdin:  strings.NewReader("senha institucional\n"),
		stdout: &bytes.Buffer{},
	})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
}

func TestRunRefusesToStartWithoutAPassword(t *testing.T) {
	code := run(environment{
		getenv: staticEnvironment(map[string]string{
			"ETPETO_DATABASE_URL": "postgres://unreachable.invalid/etpeto",
		}),
		stdin:  strings.NewReader(""),
		stdout: &bytes.Buffer{},
	})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
}

func TestReadPasswordPrefersStandardInputOverTheEnvironment(t *testing.T) {
	password, err := readPassword(environment{
		getenv: staticEnvironment(map[string]string{"ETPETO_ADMIN_PASSWORD": "do ambiente"}),
		stdin:  strings.NewReader("da entrada padrão\n"),
	})
	if err != nil || password != "da entrada padrão" {
		t.Fatalf("readPassword() = %q, %v", password, err)
	}
}

func TestReadPasswordFallsBackToTheEnvironmentForAutomatedInstalls(t *testing.T) {
	password, err := readPassword(environment{
		getenv: staticEnvironment(map[string]string{"ETPETO_ADMIN_PASSWORD": "do ambiente"}),
		stdin:  strings.NewReader(""),
	})
	if err != nil || password != "do ambiente" {
		t.Fatalf("readPassword() = %q, %v", password, err)
	}
}

func TestReadPasswordKeepsInternalWhitespaceAndStripsOnlyTheLineBreak(t *testing.T) {
	password, err := readPassword(environment{
		getenv: staticEnvironment(map[string]string{}),
		stdin:  strings.NewReader("senha com espaços\r\n"),
	})
	if err != nil || password != "senha com espaços" {
		t.Fatalf("readPassword() = %q, %v", password, err)
	}
}

func staticEnvironment(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}
