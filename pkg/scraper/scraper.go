// Package scraper provides functionality to fetch data from URLs and download files
package scraper

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// FetchURL downloads the HTML content from a URL and returns it as a string
func FetchURL(url string) (string, error) {
	log.Printf("Fetching URL: %s", url)

	// Create an HTTP client with a timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Send the HTTP request
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("error fetching URL: %w", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	log.Printf("HTTP Status: %d (%s)", resp.StatusCode, resp.Status)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-200 status code: %d %s", resp.StatusCode, resp.Status)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	// Print some information about the response
	contentType := resp.Header.Get("Content-Type")
	contentLength := resp.Header.Get("Content-Length")
	log.Printf("Content-Type: %s, Content-Length: %s bytes", contentType, contentLength)

	return string(body), nil
}

// DownloadPDF downloads a PDF file from a URL and saves it locally
func DownloadPDF(url string, localPath string) error {
	log.Printf("Downloading PDF from %s to %s", url, localPath)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Send the HTTP request
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("error fetching PDF: %w", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-200 status code: %d %s", resp.StatusCode, resp.Status)
	}

	// Create the file
	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer out.Close()

	// Write response body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("error saving PDF to file: %w", err)
	}

	log.Printf("Successfully downloaded PDF to %s", localPath)
	return nil
}

// SaveContentToFile saves content to a file
func SaveContentToFile(filename string, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}

// ExtractStandingsLinks extracts links to individual standings pages
func ExtractStandingsLinks(htmlContent string) []string {
	var links []string

	// Use goquery to parse the HTML content
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		log.Printf("Error parsing HTML content: %v", err)
		return links
	}

	// Find all <a> tags with href attributes
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		// Only collect links that look like standings pages
		if strings.Contains(href, "Fall2024") && strings.Contains(href, "Wk") {
			log.Printf("Found standings link: %s", href)
			links = append(links, href)
		}
	})

	log.Printf("Extracted %d standings links", len(links))
	return links
}

// ResolveRelativeURL resolves a relative URL to an absolute URL
func ResolveRelativeURL(baseURL, relativeURL string) string {
	// Check if the relative URL is already an absolute URL
	if strings.HasPrefix(relativeURL, "http://") || strings.HasPrefix(relativeURL, "https://") {
		return relativeURL
	}

	// Fix protocol in base URL if needed
	if !strings.HasPrefix(baseURL, "https://") && !strings.HasPrefix(baseURL, "http://") {
		// If no protocol, assume https
		baseURL = "https://" + baseURL
	} else if strings.HasPrefix(baseURL, "https:/") && !strings.HasPrefix(baseURL, "https://") {
		// Fix malformed https:/ protocol (missing slash)
		baseURL = "https://" + strings.TrimPrefix(baseURL, "https:/")
	} else if strings.HasPrefix(baseURL, "http:/") && !strings.HasPrefix(baseURL, "http://") {
		// Fix malformed http:/ protocol (missing slash)
		baseURL = "http://" + strings.TrimPrefix(baseURL, "http:/")
	}

	// Get base directory by removing the filename component
	baseDir := baseURL
	lastSlashIndex := strings.LastIndex(baseURL, "/")
	if lastSlashIndex > 0 && lastSlashIndex < len(baseURL)-1 {
		baseDir = baseURL[:lastSlashIndex+1]
	} else if !strings.HasSuffix(baseDir, "/") {
		baseDir += "/"
	}

	// Combine with relative URL
	return baseDir + relativeURL
}

// ExtractWeekNumber extracts the week number from a URL
func ExtractWeekNumber(url string) int {
	re := regexp.MustCompile(`Wk(\d+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		weekNum, err := strconv.Atoi(matches[1])
		if err == nil {
			return weekNum
		}
	}
	return 0
}
