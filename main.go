package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/nbd-wtf/go-nostr/nip44"
)

var version = "0.3.0"

type options struct {
	key        string
	recipient  string
	message    string
	relays     string
	wait       time.Duration
	verbose    bool
	jsonOutput bool
	count      int
	read       bool
}

func printHelp() {
	fmt.Fprintf(os.Stderr, `ndm (%s) - Send and Read Nostr Direct Messages (NIP-17)

USAGE:
  ndm send -k <key> -r <recipient> -m <message>
  ndm read -k <key> [-n <count>]

COMMANDS:
  send    Send a direct message (default)
  read    Read received messages
  inbox   Same as read

OPTIONS:
  -k, --key <nsec>         Your private key (nsec or hex) [required for send]
  -r, --recipient <pubkey> Recipient's public key (npub, hex, or nsec) [required for send]
  -m, --message <text>    The message to send [required for send]
  -n, --count <num>       Number of messages to read (default: 10)
  -relay, --relays <urls> Comma-separated relay URLs (default: uses well-known relays)
  -t, --timeout <sec>    How long to wait for publish confirmation (default: 30)
  -v, --verbose           Print verbose output
  -j, --json              Output result as JSON
  -h, --help              Show help
  --version               Show version number

EXAMPLES:
  ndm send -k <nsec> -r <npub> -m "Hello!"
  ndm read -k <nsec>
  ndm read -k <nsec> -n 5

NOTES:
  - Recipient can be an npub, nsec (will derive pubkey), or hex pubkey
  - If you use your own nsec as recipient, it sends to yourself

`, version)
}

func parseArgs(args []string) (*options, error) {
	opts := &options{
		wait:  30 * time.Second,
		count: 10,
	}

	// Check for command
	command := "send"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		command = args[0]
		args = args[1:]
	}

	if command == "read" || command == "inbox" {
		opts.read = true
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "-h", "--help":
			printHelp()
			os.Exit(0)
		case "--version":
			fmt.Printf("ndm version %s\n", version)
			os.Exit(0)
		case "-k", "--key":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for -k")
			}
			opts.key = args[i+1]
			i++
		case "-r", "--recipient":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for -r")
			}
			opts.recipient = args[i+1]
			i++
		case "-m", "--message":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for -m")
			}
			opts.message = args[i+1]
			i++
		case "-n", "--count":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for -n")
			}
			if _, err := fmt.Sscanf(args[i+1], "%d", &opts.count); err != nil {
				return nil, fmt.Errorf("invalid count: %w", err)
			}
			i++
		case "-relay", "--relays":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --relays")
			}
			opts.relays = args[i+1]
			i++
		case "-t", "--timeout":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for -t")
			}
			var t int
			if _, err := fmt.Sscanf(args[i+1], "%d", &t); err != nil {
				return nil, fmt.Errorf("invalid timeout: %w", err)
			}
			opts.wait = time.Duration(t) * time.Second
			i++
		case "-v", "--verbose":
			opts.verbose = true
		case "-j", "--json":
			opts.jsonOutput = true
		}
	}

	if opts.read {
		if opts.key == "" {
			return nil, fmt.Errorf("missing required flag: -k/--key (your private key)")
		}
	} else {
		if opts.key == "" {
			return nil, fmt.Errorf("missing required flag: -k/--key (your private key)")
		}
		if opts.recipient == "" {
			return nil, fmt.Errorf("missing required flag: -r/--recipient (recipient's public key)")
		}
		if opts.message == "" {
			return nil, fmt.Errorf("missing required flag: -m/--message (the message to send)")
		}
	}

	return opts, nil
}

func resolveKey(input string) (string, error) {
	input = strings.TrimSpace(input)

	if len(input) == 64 && isHex(input) {
		_, err := derivePublicKeyFromPrivate(input)
		if err == nil {
			return derivePublicKeyFromPrivate(input)
		}
		return input, nil
	}

	if strings.HasPrefix(input, "nsec") {
		prefix, value, _ := nip19.Decode(input)
		if prefix == "nsec" {
			secret, ok := value.(string)
			if ok {
				return derivePublicKeyFromPrivate(secret)
			}
		}
	}

	if strings.HasPrefix(input, "npub") {
		prefix, value, _ := nip19.Decode(input)
		if prefix == "npub" {
			pubkey, ok := value.(string)
			if ok {
				return pubkey, nil
			}
		}
	}

	return "", fmt.Errorf("could not resolve key: %s", input)
}

func resolvePrivateKey(input string) (string, error) {
	input = strings.TrimSpace(input)

	if len(input) == 64 && isHex(input) {
		return input, nil
	}

	if strings.HasPrefix(input, "nsec") {
		prefix, value, _ := nip19.Decode(input)
		if prefix == "nsec" {
			secret, ok := value.(string)
			if !ok {
				return "", fmt.Errorf("invalid secret value")
			}
			return secret, nil
		}
	}

	return "", fmt.Errorf("invalid private key format")
}

func derivePublicKeyFromPrivate(privkeyHex string) (string, error) {
	evt := &nostr.Event{
		Kind:      1,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
		Content:   "temp",
	}
	err := evt.Sign(privkeyHex)
	if err != nil {
		return "", err
	}
	return evt.PubKey, nil
}

func decryptMessage(privkey, pubkey, content string) (string, error) {
	if content == "" {
		return "", fmt.Errorf("empty content")
	}
	key, err := nip44.GenerateConversationKey(pubkey, privkey)
	if err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}
	return nip44.Decrypt(content, key)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func run(args []string) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}

	opts, err := parseArgs(args)
	if err != nil {
		return err
	}

	if opts.read {
		return readMessages(opts)
	}
	return sendMessage(opts)
}

func sendMessage(opts *options) error {
	ctx, cancel := context.WithTimeout(context.Background(), opts.wait)
	defer cancel()

	privkey, err := resolvePrivateKey(opts.key)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	recipientPubkey, err := resolveKey(opts.recipient)
	if err != nil {
		return fmt.Errorf("invalid recipient: %w", err)
	}

	relays := []string{
		"wss://relay.damus.io",
		"wss://relay.nostr.band",
		"wss://nos.lol",
	}

	if opts.relays != "" {
		relays = strings.Split(opts.relays, ",")
		for i := range relays {
			relays[i] = strings.TrimSpace(relays[i])
		}
	}

	if opts.verbose {
		fmt.Fprintf(os.Stderr, "[ndm] Using key: %s...\n", privkey[:20])
		fmt.Fprintf(os.Stderr, "[ndm] Sending to: %s\n", recipientPubkey)
	}

	// Encrypt the message
	conversationKey, err := nip44.GenerateConversationKey(recipientPubkey, privkey)
	if err != nil {
		return fmt.Errorf("failed to generate conversation key: %w", err)
	}
	encryptedContent, err := nip44.Encrypt(opts.message, conversationKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt: %w", err)
	}

	event := nostr.Event{
		Kind:      nostr.KindEncryptedDirectMessage,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
		Tags:      nostr.Tags{{"p", recipientPubkey}},
		Content:   encryptedContent,
	}

	err = event.Sign(privkey)
	if err != nil {
		return fmt.Errorf("failed to sign event: %w", err)
	}

	published := 0
	for _, relay := range relays {
		rc, err := nostr.RelayConnect(ctx, relay)
		if err != nil {
			continue
		}

		err = rc.Publish(ctx, event)
		rc.Close()
		if err == nil {
			published++
		}
	}

	if published == 0 {
		return fmt.Errorf("failed to publish to any relay")
	}

	recipientNpub, _ := nip19.EncodePublicKey(recipientPubkey)

	if opts.jsonOutput {
		fmt.Printf(`{"success":true,"message_id":"%s","encrypted_to":"%s","relays":%d}`, event.ID, recipientNpub, published)
	} else {
		fmt.Printf("✓ DM sent successfully\n")
		fmt.Printf("  Message ID: %s\n", event.ID)
		fmt.Printf("  To: %s\n", recipientNpub)
		fmt.Printf("  Relays: %d\n", published)
	}

	return nil
}

func readMessages(opts *options) error {
	ctx, cancel := context.WithTimeout(context.Background(), opts.wait)
	defer cancel()

	privkey, err := resolvePrivateKey(opts.key)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	pubkey, err := derivePublicKeyFromPrivate(privkey)
	if err != nil {
		return fmt.Errorf("invalid key: %w", err)
	}

	relays := []string{
		"wss://relay.damus.io",
		"wss://relay.nostr.band",
		"wss://nos.lol",
	}

	if opts.relays != "" {
		relays = strings.Split(opts.relays, ",")
		for i := range relays {
			relays[i] = strings.TrimSpace(relays[i])
		}
	}

	if opts.verbose {
		fmt.Fprintf(os.Stderr, "[ndm] Using key: %s...\n", privkey[:20])
		fmt.Fprintf(os.Stderr, "[ndm] Pubkey: %s\n", pubkey)
		fmt.Fprintf(os.Stderr, "[ndm] Fetching from: %v\n", relays)
	}

	filter := nostr.Filter{
		Kinds: []int{nostr.KindEncryptedDirectMessage},
		Tags:  nostr.TagMap{"p": []string{pubkey}},
		Limit: opts.count,
	}

	var events []*nostr.Event
	for _, relay := range relays {
		rc, err := nostr.RelayConnect(ctx, relay)
		if err != nil {
			if opts.verbose {
				fmt.Fprintf(os.Stderr, "[ndm] Failed to connect to %s: %v\n", relay, err)
			}
			continue
		}

		eventsCh, err := rc.QueryEvents(ctx, filter)
		if err != nil {
			rc.Close()
			continue
		}

		for evt := range eventsCh {
			events = append(events, evt)
			if len(events) >= opts.count {
				break
			}
		}
		rc.Close()
		if len(events) >= opts.count {
			break
		}
	}

	if len(events) == 0 {
		fmt.Println("No messages found")
		return nil
	}

	if opts.jsonOutput {
		type msg struct {
			ID        string `json:"id"`
			From      string `json:"from"`
			Content   string `json:"content"`
			CreatedAt int64  `json:"created_at"`
		}
		var msgs []msg
		for _, e := range events {
			decrypted, _ := decryptMessage(privkey, e.PubKey, e.Content)
			msgs = append(msgs, msg{
				ID:        e.ID,
				From:      e.PubKey,
				Content:   decrypted,
				CreatedAt: int64(e.CreatedAt),
			})
		}
		out, _ := json.MarshalIndent(msgs, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Printf("Found %d messages:\n\n", len(events))
		for i, e := range events {
			decrypted, err := decryptMessage(privkey, e.PubKey, e.Content)
			if err != nil {
				fmt.Printf("[%d] From: %s\n", i+1, e.PubKey[:16]+"...")
				fmt.Printf("    ID: %s\n", e.ID[:16]+"...")
				fmt.Printf("    Content: (decrypt failed: %v)\n", err)
				fmt.Printf("    Raw: %s\n\n", e.Content[:min(50, len(e.Content))]+"...")
			} else {
				fromNpub, _ := nip19.EncodePublicKey(e.PubKey)
				fmt.Printf("[%d] From: %s\n", i+1, fromNpub[:20]+"...")
				fmt.Printf("    ID: %s\n", e.ID[:16]+"...")
				fmt.Printf("    Time: %s\n", time.Unix(int64(e.CreatedAt), 0).Format("2006-01-02 15:04:05"))
				fmt.Printf("    Content: %s\n\n", decrypted)
			}
		}
	}

	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
