package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

var (
	outputFile = flag.String("output", "urlscan.recent", "Output file path")
	loop       = flag.Bool("loop", false, "Enable continuous loop mode")
	interval   = flag.Int("interval", 5, "Loop interval in seconds")
	timeout    = flag.Int("timeout", 20, "Page load timeout in seconds")
)

func main() {
	flag.Parse()

	if *loop {
		runLoop()
	} else {
		if err := scrapeAndSave(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func runLoop() {
	fmt.Printf("[*] Starting URLScan auto-scraper (every %d seconds)...\n", *interval)
	fmt.Printf("[*] Output file: %s\n\n", *outputFile)

	// Create Chrome allocator once - browser process stays open for entire loop
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Run scraper in a goroutine
	go func() {
		ticker := time.NewTicker(time.Duration(*interval) * time.Second)
		defer ticker.Stop()

		// Run first scrape immediately
		if err := scrapeAndSaveWithContext(allocCtx); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			fmt.Printf("[+] Updated: %s\n", time.Now().Format("Mon Jan  2 15:04:05 UTC 2006"))
		}

		// Then run every interval
		for range ticker.C {
			if err := scrapeAndSaveWithContext(allocCtx); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			} else {
				fmt.Printf("[+] Updated: %s\n", time.Now().Format("Mon Jan  2 15:04:05 UTC 2006"))
			}
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	fmt.Println("\n[*] Shutting down...")
	// Context cancellation will close Chrome browser
}

func scrapeAndSave() error {
	domains, err := scrapeURLScan()
	if err != nil {
		return fmt.Errorf("failed to scrape: %w", err)
	}

	if err := saveDomains(domains); err != nil {
		return fmt.Errorf("failed to save domains: %w", err)
	}

	return nil
}

func scrapeAndSaveWithContext(ctx context.Context) error {
	domains, err := scrapeURLScanWithContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to scrape: %w", err)
	}

	if err := saveDomains(domains); err != nil {
		return fmt.Errorf("failed to save domains: %w", err)
	}

	return nil
}

func scrapeURLScan() ([]string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	// Create chromedp context
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(allocCtx, chromedp.WithLogf(func(format string, v ...interface{}) {
		// Suppress chromedp logs
	}))
	defer cancel()

	var htmlContent string

	// Navigate and wait for selector
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://urlscan.io"),
		chromedp.WaitVisible("td.break-all.url a", chromedp.ByQuery),
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to load page: %w", err)
	}

	// Parse HTML and extract domains
	domains := extractDomains(htmlContent)
	return domains, nil
}

func scrapeURLScanWithContext(allocCtx context.Context) ([]string, error) {
	// Create a new Chrome context (browser tab) for this scrape
	chromeCtx, cancelChrome := chromedp.NewContext(allocCtx, chromedp.WithLogf(func(format string, v ...interface{}) {
		// Suppress chromedp logs
	}))
	defer cancelChrome()

	// Create timeout context for this specific scrape operation
	scrapeCtx, cancel := context.WithTimeout(chromeCtx, time.Duration(*timeout)*time.Second)
	defer cancel()

	var htmlContent string

	// Navigate and wait for selector (using new tab, browser process stays open)
	err := chromedp.Run(scrapeCtx,
		chromedp.Navigate("https://urlscan.io"),
		chromedp.WaitVisible("td.break-all.url a", chromedp.ByQuery),
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to load page: %w", err)
	}

	// Parse HTML and extract domains
	domains := extractDomains(htmlContent)
	return domains, nil
}

func extractDomains(html string) []string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		// Fallback to regex if goquery fails
		return extractDomainsRegex(html)
	}

	var domains []string
	seen := make(map[string]bool)

	// Find all <td class="break-all url"><a> elements
	doc.Find("td.break-all.url a").Each(func(i int, s *goquery.Selection) {
		title, exists := s.Attr("title")
		if exists && title != "" {
			// Clean up the domain (remove trailing slash if present)
			domain := strings.TrimSuffix(title, "/")
			domain = strings.TrimSpace(domain)
			if domain != "" && !seen[domain] {
				domains = append(domains, domain)
				seen[domain] = true
			}
		}
	})

	return domains
}

func extractDomainsRegex(html string) []string {
	// Fallback regex pattern to extract title attributes from <td class="break-all url"><a title="...">
	pattern := regexp.MustCompile(`<td class="break-all url"><a[^>]*title="([^"]+)"`)
	matches := pattern.FindAllStringSubmatch(html, -1)

	seen := make(map[string]bool)
	var domains []string

	for _, match := range matches {
		if len(match) > 1 {
			domain := strings.TrimSuffix(match[1], "/")
			domain = strings.TrimSpace(domain)
			if domain != "" && !seen[domain] {
				domains = append(domains, domain)
				seen[domain] = true
			}
		}
	}

	return domains
}

func saveDomains(newDomains []string) error {
	if len(newDomains) == 0 {
		return nil
	}

	// Read existing domains
	existingDomains := make(map[string]bool)
	file, err := os.Open(*outputFile)
	if err == nil {
		data, err := io.ReadAll(file)
		file.Close()
		if err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					existingDomains[line] = true
				}
			}
		}
	}

	// Append new unique domains
	outputFile, err := os.OpenFile(*outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer outputFile.Close()

	for _, domain := range newDomains {
		if !existingDomains[domain] {
			if _, err := outputFile.WriteString(domain + "\n"); err != nil {
				return fmt.Errorf("failed to write domain: %w", err)
			}
			existingDomains[domain] = true
		}
	}

	return nil
}
