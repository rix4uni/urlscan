## URLScan Scraper

A Go-based tool to scrape recent scans from [urlscan.io](https://urlscan.io) and extract unique domains. This tool replaces the previous Node.js/Playwright implementation with a native Go solution that can be easily installed and distributed.

## Features

- 🚀 Fast and lightweight Go implementation
- 🔄 Continuous loop mode for automatic updates
- 📝 Extracts unique domains from recent scans
- ⚙️ Configurable output file and intervals
- 🛑 Graceful shutdown handling (Ctrl+C)

## Installation

### From Source

If you have Go installed, you can install directly from this repository:

```bash
go install github.com/rix4uni/urlscan@latest
```

Or clone and build:

```bash
git clone <repository-url>
cd urlscan
go build -o urlscan
```

### Prerequisites

- Go 1.21 or later
- Chrome/Chromium browser (chromedp will download it automatically on first run)

## Usage

### Single Run

Run the scraper once and save domains to the output file:

```bash
urlscan --output urlscan.recent
```

### Continuous Loop Mode

Run the scraper continuously, updating every 5 seconds:

```bash
urlscan --loop --output urlscan.recent --interval 5
```

### Command Line Options

- `--output` or `-o`: Output file path (default: `urlscan.recent`)
- `--loop` or `-l`: Enable continuous loop mode
- `--interval` or `-i`: Loop interval in seconds (default: 5)
- `--timeout`: Page load timeout in seconds (default: 20)

### Examples

```bash
urlscan --loop

# Single run with custom output file
urlscan --output my-domains.txt

# Loop mode with 10 second intervals
urlscan --loop --interval 10

# Loop mode with custom timeout
urlscan --loop --timeout 30 --output domains.txt
```

## How It Works

1. Launches a headless Chrome browser using `chromedp`
2. Navigates to `https://urlscan.io`
3. Waits for the recent scans table to load
4. Extracts domains from `<td class="break-all url"><a>` elements
5. Saves unique domains to the output file (appends new ones)

## Output Format

The tool writes one domain per line to the output file:

```
example.com
another-domain.org
test-site.net
```

Duplicate domains are automatically filtered out.

## Dependencies

- [chromedp](https://github.com/chromedp/chromedp) - Headless browser automation
- [goquery](https://github.com/PuerkitoBio/goquery) - HTML parsing

## Migration from Node.js Version

This Go version replaces the previous Node.js implementation:

- `dump.js` → Replaced by `main.go`
- `urlscan_loop.sh` → Replaced by `--loop` flag
- No need for `npm install` or Node.js dependencies

## License

[Add your license here]

## Contributing

[Add contribution guidelines here]
