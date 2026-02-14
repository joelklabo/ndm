# ndm - Nostr Direct Message CLI

A simple, reliable CLI tool for sending encrypted direct messages (DMs) on Nostr.

## Features

- Send NIP-17 encrypted DMs from the command line
- Supports nsec, ncryptsec, and hex secret key formats
- Supports npub and hex public key formats for recipients
- Configurable relay list
- JSON output for scripting
- Timeout control
- Verbose mode for debugging

## Installation

### From Source

```bash
go install github.com/klabo/ndm@latest
```

### Using Homebrew

```bash
brew install ndm
```

### Manual

1. Download the latest release for your platform
2. Make it executable: `chmod +x ndm`
3. Move it to your PATH: `mv ndm /usr/local/bin/`

## Requirements

- [NAK](https://github.com/fiatjaf/nak) must be installed and in your PATH

```bash
go install github.com/fiatjaf/nak@latest
```

## Usage

```bash
ndm -k <your-nsec> -r <recipient-npub> -m "Hello, world!"
```

### Options

| Flag | Description |
|------|-------------|
| `-k`, `--key` | Your private key (nsec, ncryptsec, or hex format) [required] |
| `-r`, `--recipient` | Recipient's public key (npub or hex) [required] |
| `-m`, `--message` | The message to send [required] |
| `-relay`, `--relays` | Comma-separated relay URLs (default: uses well-known relays) |
| `-t`, `--timeout` | Timeout duration (default: 30s) |
| `-v`, `--verbose` | Print verbose output |
| `-j`, `--json` | Output result as JSON |
| `-h`, `--help` | Show help message |
| `--version` | Show version number |

### Examples

Send a DM using nsec:
```bash
ndm -k nsec1... -r npub1... -m "Hello!"
```

Send a DM using hex keys:
```bash
ndm -k 0123456789abcdef... -r abcdef0123456789... -m "Hello!"
```

Use custom relays:
```bash
ndm -k nsec1... -r npub1... -m "Hello!" -relay wss://relay.damus.io,wss://nos.lol
```

Get JSON output for scripting:
```bash
ndm -k nsec1... -r npub1... -m "Hello!" -j
```

Verbose mode for debugging:
```bash
ndm -k nsec1... -r npub1... -m "Hello!" -v
```

## Exit Codes

- `0` - Success
- `1` - Invalid arguments or other error
- `2` - Failed to encrypt message
- `3` - Failed to sign event
- `4` - Failed to publish to all relays

## Development

### Building

```bash
go build -o ndm .
```

### Testing

```bash
go test -v ./...
```

### Linting

```bash
golangci-lint run
```

## License

MIT

## Related

- [NAK](https://github.com/fiatjaf/nak) - Nostr Army Knife
- [NIP-17](https://github.com/nostr-protocol/nips/blob/master/17.md) - Encrypted Direct Messages
- [NIP-44](https://github.com/nostr-protocol/nips/blob/master/44.md) - Encryption
