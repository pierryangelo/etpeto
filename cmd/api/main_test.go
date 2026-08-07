package main

import (
	"strings"
	"testing"
)

func TestLoadConfigRequiresDatabaseURL(t *testing.T) {
	_, err := loadConfig(func(string) string { return "" })

	if err == nil || !strings.Contains(err.Error(), "ETPETO_DATABASE_URL") {
		t.Fatalf("loadConfig() error = %v, want missing database URL", err)
	}
}

func TestLoadConfigUsesDefaultAddress(t *testing.T) {
	config, err := loadConfig(func(key string) string {
		switch key {
		case "ETPETO_DATABASE_URL":
			return "postgres://configured"
		case "ETPETO_RENDERER_URL":
			return "http://renderer:8081"
		case "ETPETO_RENDERER_TOKEN":
			return "renderer-secret"
		}
		return ""
	})

	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if config.Address != defaultAddress {
		t.Fatalf("address = %q, want %q", config.Address, defaultAddress)
	}
}

func TestLoadConfigRequiresRendererConnection(t *testing.T) {
	_, err := loadConfig(func(key string) string {
		if key == "ETPETO_DATABASE_URL" {
			return "postgres://configured"
		}
		return ""
	})

	if err == nil || !strings.Contains(err.Error(), "ETPETO_RENDERER_URL") {
		t.Fatalf("loadConfig() error = %v, want missing renderer URL", err)
	}
}
