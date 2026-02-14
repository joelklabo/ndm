package main

import (
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
			args:    []string{"-k", "nsec1test", "-r", "npub1test", "-m", "hello", "-t", "60"},
			wantErr: false,
		},
		{
			name:    "long form flags",
			args:    []string{"--key", "nsec1test", "--recipient", "npub1test", "--message", "hello"},
			wantErr: false,
		},
		{
			name:    "read command",
			args:    []string{"read", "-k", "nsec1test"},
			wantErr: false,
		},
		{
			name:    "inbox command",
			args:    []string{"inbox", "-k", "nsec1test"},
			wantErr: false,
		},
		{
			name:    "count flag for read",
			args:    []string{"read", "-k", "nsec1test", "-n", "5"},
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

func TestIsHex(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"", true},
		{"abcdef1234567890", true},
		{"ABCDEF1234567890", true},
		{"abcdefgh", false},
		{"0123456789abcdef", true},
		{"0123456789ABCDEF", true},
		{"0", true},
		{"g", false},
		{"a", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isHex(tt.input)
			if got != tt.want {
				t.Errorf("isHex(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolvePrivateKey(t *testing.T) {
	// Test with a valid hex key
	validHex := "d898fd8d6ba74893a08d7a6a2d244348b55f8b4a0c417574ca19682d1577c0b1"
	got, err := resolvePrivateKey(validHex)
	if err != nil {
		t.Errorf("resolvePrivateKey(%q) error = %v", validHex, err)
	}
	if got != validHex {
		t.Errorf("resolvePrivateKey(%q) = %v, want %v", validHex, got, validHex)
	}
}

func TestResolveKey(t *testing.T) {
	// Test npub resolution
	npub := "npub1c7a9q3r3s437pa87l6qrpxw2k6km0enqfdden9ldtcdsyqzmwfysdsu25t"
	got, err := resolveKey(npub)
	if err != nil {
		t.Errorf("resolveKey(%q) error = %v", npub, err)
	}
	if len(got) != 64 {
		t.Errorf("resolveKey(%q) = %v, want 64-char hex", npub, got)
	}

	// Test hex privkey - should derive pubkey
	hexPriv := strings.Repeat("a", 64)
	got, err = resolveKey(hexPriv)
	if err != nil {
		t.Errorf("resolveKey(%q) error = %v", hexPriv, err)
	}
	// Should be a derived pubkey, not the same as input
	if got == hexPriv {
		t.Errorf("resolveKey(%q) should derive pubkey, got same", hexPriv)
	}
}

func TestDerivePublicKeyFromPrivate(t *testing.T) {
	// Test with a known private key
	privkey := "d898fd8d6ba74893a08d7a6a2d244348b55f8b4a0c417574ca19682d1577c0b1"
	got, err := derivePublicKeyFromPrivate(privkey)
	if err != nil {
		t.Errorf("derivePublicKeyFromPrivate(%q) error = %v", privkey, err)
	}
	// Should be 64 char hex
	if len(got) != 64 {
		t.Errorf("derivePublicKeyFromPrivate(%q) = %v (len %d), want 64-char hex", privkey, got, len(got))
	}

	// Test that deriving from same privkey gives same pubkey
	got2, _ := derivePublicKeyFromPrivate(privkey)
	if got != got2 {
		t.Errorf("derivePublicKeyFromPrivate(%q) not deterministic: got %v, want %v", privkey, got2, got)
	}
}

func TestDecryptMessage(t *testing.T) {
	// We'll test that encryption then decryption gives back original
	// This is tested via the full round-trip in integration tests
	// Just test error cases here

	_, err := decryptMessage("", "abc", "test")
	if err == nil {
		t.Error("expected error for empty privkey")
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

func TestCountFlag(t *testing.T) {
	opts, err := parseArgs([]string{"read", "-k", "nsec1test", "-n", "5"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.count != 5 {
		t.Errorf("expected count 5, got %d", opts.count)
	}
}
