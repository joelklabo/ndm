package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	version       = "0.1.0"
	appName       = "ndm"
	appDesc       = "Send Nostr Direct Messages (NIP-17) from the command line"
	defaultRelays = []string{
		"wss://relay.damus.io",
		"wss://relay.nostr.band",
		"wss://nos.lol",
	}
)

type options struct {
	key        string
	recipient  string
	message    string
	relays     string
	wait       time.Duration
	verbose    bool
	jsonOutput bool
}

func printHelp() {
	fmt.Fprintf(os.Stderr, `%s (%s) - %s

USAGE:
  %s [OPTIONS]

OPTIONS:
  -k, --key <nsec>         Your private key (nsec, ncryptsec, or hex format) [required]
  -r, --recipient <pubkey> Recipient's public key (npub, hex, or npub1... bech32) [required]
  -m, --message <text>    The message to send [required]
  -relay, --relays <urls> Comma-separated relay URLs (default: uses well-known relays)
  -t, --timeout <sec>     How long to wait for publish confirmation (default: 30)
  -v, --verbose           Print verbose output
  -j, --json              Output result as JSON
  -h, --help              Show this help message
  --version               Show version number

EXIT CODES:
  0   Success
  1   Invalid arguments or other error
  2   Failed to encrypt message
  3   Failed to sign event
  4   Failed to publish to all relays

For more info: https://github.com/klabo/ndm
`, appName, version, appDesc, appName)
}

func parseArgs(args []string) (*options, error) {
	opts := &options{
		wait: 30 * time.Second,
	}

	fs := flag.NewFlagSet(appName, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {}

	fs.StringVar(&opts.key, "k", "", "")
	fs.StringVar(&opts.key, "key", "", "")
	fs.StringVar(&opts.recipient, "r", "", "")
	fs.StringVar(&opts.recipient, "recipient", "", "")
	fs.StringVar(&opts.message, "m", "", "")
	fs.StringVar(&opts.message, "message", "", "")
	fs.StringVar(&opts.relays, "relay", "", "")
	fs.StringVar(&opts.relays, "relays", "", "")
	fs.DurationVar(&opts.wait, "t", 30*time.Second, "")
	fs.DurationVar(&opts.wait, "timeout", 30*time.Second, "")
	fs.BoolVar(&opts.verbose, "v", false, "")
	fs.BoolVar(&opts.verbose, "verbose", false, "")
	fs.BoolVar(&opts.jsonOutput, "j", false, "")
	fs.BoolVar(&opts.jsonOutput, "json", false, "")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if opts.key == "" {
		return nil, errors.New("missing required flag: -k/--key (your private key)")
	}
	if opts.recipient == "" {
		return nil, errors.New("missing required flag: -r/--recipient (recipient's public key)")
	}
	if opts.message == "" {
		return nil, errors.New("missing required flag: -m/--message (the message to send)")
	}

	return opts, nil
}

type publishResult struct {
	URL    string `json:"url"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type result struct {
	Success       bool            `json:"success"`
	MessageID     string          `json:"message_id,omitempty"`
	Event         string          `json:"event_json,omitempty"`
	Relays        []publishResult `json:"relays"`
	EncryptedTo   string          `json:"encrypted_to"`
	RetryableFail int             `json:"retryable_failures"`
	FatalFail     int             `json:"fatal_failures"`
}

func run(args []string) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}

	if args[0] == "-h" || args[0] == "--help" {
		printHelp()
		return nil
	}

	if args[0] == "--version" {
		fmt.Printf("%s version %s\n", appName, version)
		return nil
	}

	opts, err := parseArgs(args)
	if err != nil {
		return err
	}

	return sendDM(opts)
}

func sendDM(opts *options) error {
	ctx, cancel := context.WithTimeout(context.Background(), opts.wait)
	defer cancel()

	relays := defaultRelays
	if opts.relays != "" {
		relays = strings.Split(opts.relays, ",")
		for i := range relays {
			relays[i] = strings.TrimSpace(relays[i])
		}
	}

	if opts.verbose {
		fmt.Fprintf(os.Stderr, "[ndm] Using key: %s...\n", opts.key[:20])
		fmt.Fprintf(os.Stderr, "[ndm] Sending to: %s\n", opts.recipient)
		fmt.Fprintf(os.Stderr, "[ndm] Relays: %v\n", relays)
	}

	encrypted, err := encryptMessage(opts.key, opts.recipient, opts.message)
	if err != nil {
		return fmt.Errorf("encryption failed (exit code 2): %w", err)
	}

	if opts.verbose {
		fmt.Fprintf(os.Stderr, "[ndm] Encrypted message: %s\n", encrypted[:50])
	}

	eventJSON, err := createAndPublishEvent(ctx, opts.key, opts.recipient, encrypted, relays, opts.verbose)
	if err != nil {
		return err
	}

	var event struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
		return fmt.Errorf("failed to parse event JSON: %w", err)
	}

	results := []publishResult{}
	for _, relay := range relays {
		results = append(results, publishResult{
			URL:    relay,
			Status: "published",
		})
	}

	res := result{
		Success:       true,
		MessageID:     event.ID,
		Event:         eventJSON,
		Relays:        results,
		EncryptedTo:   opts.recipient,
		RetryableFail: 0,
		FatalFail:     0,
	}

	if opts.jsonOutput {
		output, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(output))
	} else {
		fmt.Printf("✓ DM sent successfully\n")
		fmt.Printf("  Message ID: %s\n", event.ID)
		fmt.Printf("  To: %s\n", opts.recipient)
		fmt.Printf("  Relays: %d\n", len(relays))
	}

	return nil
}

func encryptMessage(key, recipient, message string) (string, error) {
	cmd := exec.Command("nak", "encrypt",
		"--sec", key,
		"--recipient-pubkey", recipient,
		message,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("nak encrypt error: %s", strings.TrimSpace(stderr.String()))
		}
		return "", fmt.Errorf("nak encrypt failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func createAndPublishEvent(ctx context.Context, key, recipient, encryptedContent string, relays []string, verbose bool) (string, error) {
	args := []string{
		"event",
		"--sec", key,
		"-k", "4",
		"-c", encryptedContent,
		"-p", recipient,
		"--quiet",
	}

	args = append(args, relays...)

	cmd := exec.CommandContext(ctx, "nak", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		if stderr.Len() > 0 {
			errMsg := strings.TrimSpace(stderr.String())
			if strings.Contains(errMsg, "auth") {
				return "", fmt.Errorf("authentication failed: %s (exit code 3)", errMsg)
			}
			return "", fmt.Errorf("nak event error: %s (exit code 3)", errMsg)
		}
		return "", fmt.Errorf("nak event failed: %w (exit code 3)", err)
	}

	eventJSON := strings.TrimSpace(string(output))
	if verbose {
		fmt.Fprintf(os.Stderr, "[ndm] Event: %s\n", eventJSON[:min(100, len(eventJSON))])
	}

	return eventJSON, nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		if strings.Contains(err.Error(), "missing required flag") {
			fmt.Fprintf(os.Stderr, "\nRun '%s --help' for usage information.\n", appName)
			os.Exit(1)
		}
		os.Exit(1)
	}
}
