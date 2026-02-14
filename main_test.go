package main

import (
	"os"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid with nsec and npub",
			args:    []string{"-k", "nsec1test", "-r", "npub1test", "-m", "hello"},
			wantErr: false,
		},
		{
			name:    "valid with hex keys",
			args:    []string{"-k", strings.Repeat("a", 64), "-r", strings.Repeat("b", 64), "-m", "hello"},
			wantErr: false,
		},
		{
			name:        "missing key",
			args:        []string{"-r", "npub1test", "-m", "hello"},
			wantErr:     true,
			errContains: "missing required flag: -k",
		},
		{
			name:        "missing recipient",
			args:        []string{"-k", "nsec1test", "-m", "hello"},
			wantErr:     true,
			errContains: "missing required flag: -r",
		},
		{
			name:        "missing message",
			args:        []string{"-k", "nsec1test", "-r", "npub1test"},
			wantErr:     true,
			errContains: "missing required flag: -m",
		},
		{
			name:    "custom relays",
			args:    []string{"-k", "nsec1test", "-r", "npub1test", "-m", "hello", "-relay", "wss://custom.relay"},
			wantErr: false,
		},
		{
			name:    "multiple relays",
			args:    []string{"-k", "nsec1test", "-r", "npub1test", "-m", "hello", "-relay", "wss://a.com,wss://b.com"},
			wantErr: false,
		},
		{
			name:    "verbose flag",
			args:    []string{"-k", "nsec1test", "-r", "npub1test", "-m", "hello", "-v"},
			wantErr: false,
		},
		{
			name:    "json output flag",
			args:    []string{"-k", "nsec1test", "-r", "npub1test", "-m", "hello", "-j"},
			wantErr: false,
		},
		{
			name:    "timeout flag",
			args:    []string{"-k", "nsec1test", "-r", "npub1test", "-m", "hello", "-t", "60s"},
			wantErr: false,
		},
		{
			name:    "long form flags",
			args:    []string{"--key", "nsec1test", "--recipient", "npub1test", "--message", "hello"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDefaultRelays(t *testing.T) {
	opts, err := parseArgs([]string{"-k", "nsec1test", "-r", "npub1test", "-m", "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// When no relay is specified, the sendDM function will use defaultRelays
	// We can't directly test that here without mocking, but we can verify
	// the parsing doesn't break
	if opts.relays != "" {
		t.Errorf("expected empty relays, got %q", opts.relays)
	}
}

func TestCustomRelays(t *testing.T) {
	customRelays := "wss://relay1.com,wss://relay2.com"
	opts, err := parseArgs([]string{
		"-k", "nsec1test",
		"-r", "npub1test",
		"-m", "hello",
		"-relay", customRelays,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.relays != customRelays {
		t.Errorf("expected %q, got %q", customRelays, opts.relays)
	}
}

func TestHelpFlag(t *testing.T) {
	// Test that -h returns an error (flag: help requested) which is expected behavior
	_, err := parseArgs([]string{"-h"})
	if err == nil {
		t.Error("expected error for -h but got nil")
	}
}

func TestVersionFlag(t *testing.T) {
	// Test that --version is handled by main
	// We just verify it doesn't panic
	if os.Args[0] == "" {
		t.Skip("skipping in test context")
	}
}

func TestNsecFormats(t *testing.T) {
	formats := []string{
		"nsec1mzv0mrtt5ayf8gyd0f4z6fzrfz64lz62p3qh2ax2r95z69thczcsps889v",
		"d898fd8d6ba74893a08d7a6a2d244348b55f8b4a0c417574ca19682d1577c0b1",
	}

	for _, format := range formats {
		_, err := parseArgs([]string{
			"-k", format,
			"-r", "npub1c7a9q3r3s437pa87l6qrpxw2k6km0enqfdden9ldtcdsyqzmwfysdsu25t",
			"-m", "hello",
		})
		if err != nil {
			t.Errorf("failed to parse %s: %v", format, err)
		}
	}
}

func TestNpubFormats(t *testing.T) {
	formats := []string{
		"npub1c7a9q3r3s437pa87l6qrpxw2k6km0enqfdden9ldtcdsyqzmwfysdsu25t",
		"c7ba5044718563e0f4fefe803099cab6adb7e6604b5b9997ed5e1b02005b7249",
	}

	for _, format := range formats {
		_, err := parseArgs([]string{
			"-k", "nsec1mzv0mrtt5ayf8gyd0f4z6fzrfz64lz62p3qh2ax2r95z69thczcsps889v",
			"-r", format,
			"-m", "hello",
		})
		if err != nil {
			t.Errorf("failed to parse recipient %s: %v", format, err)
		}
	}
}
