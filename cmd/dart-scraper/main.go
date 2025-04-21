// Package main is the entry point for the dart-statistic-scraper application
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/myusername/dart-statistic-scraper/internal/utils"
	"github.com/myusername/dart-statistic-scraper/pkg/models"
	"github.com/myusername/dart-statistic-scraper/pkg/parser"
	"github.com/myusername/dart-statistic-scraper/pkg/scraper"
)

// Version is set during build using ldflags
var (
	version = "dev"
)

func main() {
	// Define command-line flags
	versionFlag := flag.Bool("version", false, "Print version information and exit")
	outputFlag := flag.String("output", "", "Output directory for CSV files (default: current directory)")
	flag.Parse()

	// Print version and exit if requested
	if *versionFlag {
		fmt.Printf("dart-statistic-scraper version %s\n", version)
		return
	}

	// Setup logging
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("Dart Standings Scraper starting...")
	log.Printf("Version: %s", version)

	// Create output directory if specified
	outputDir := "."
	if *outputFlag != "" {
		outputDir = *outputFlag
		err := os.MkdirAll(outputDir, 0755)
		if err != nil {
			log.Fatalf("Failed to create output directory: %v", err)
		}
		log.Printf("Using output directory: %s", outputDir)
	}

	// Create subdirectories for different file types
	htmlDir := filepath.Join(outputDir, "html")
	csvDir := filepath.Join(outputDir, "csv")
	pdfDir := filepath.Join(outputDir, "pdf")

	// Create the directories
	for _, dir := range []string{htmlDir, csvDir, pdfDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Initialize parser with fetch function
	parser.FetchURL = scraper.FetchURL

	// PDF schedule URL
	scheduleURL := "https://macdleagues.com/DartSchedules/FALL2024Schedules/FALL2024%2024SUN1.pdf"
	localPDFPath := filepath.Join(pdfDir, "fall2024_schedule.pdf")

	// Check if we already have the PDF
	var schedules []models.MatchSchedule
	if _, err := os.Stat(localPDFPath); os.IsNotExist(err) {
		// Download the PDF if it doesn't exist
		log.Printf("Attempting to download schedule PDF from %s", scheduleURL)
		err := scraper.DownloadPDF(scheduleURL, localPDFPath)
		if err != nil {
			log.Printf("Error downloading PDF schedule: %v. Using fallback manual schedule.", err)
			schedules = parser.ParseScheduleManually()
		}
	}

	// Process the schedule PDF
	if len(schedules) == 0 {
		pdfText, err := parser.ReadPDFText(localPDFPath)
		if err != nil {
			log.Printf("Error reading PDF text: %v. Using fallback manual schedule.", err)
			schedules = parser.ParseScheduleManually()
		} else {
			// Extract schedule information from the PDF text
			schedules = parser.ExtractScheduleFromText(pdfText)

			// If no schedules were extracted, fall back to manual parsing
			if len(schedules) == 0 {
				log.Printf("No schedules extracted from PDF. Using fallback manual schedule.")
				schedules = parser.ParseScheduleManually()
			} else {
				log.Printf("Successfully extracted %d match schedules from PDF", len(schedules))
			}
		}
	}

	// Base URL for the standings page
	urls := []string{
		"https://macdleagues.com/DartStandings/FALL2024standings/FALL2024%2024SUN1OZCounty.html",
	}
	log.Printf("Will scrape %d URLs", len(urls))

	// Process each URL
	var allWeeklyStats []*models.WeeklyStats

	for i, url := range urls {
		log.Printf("Processing URL %d of %d: %s", i+1, len(urls), url)

		// Download and extract standings links
		htmlContent, err := scraper.FetchURL(url)
		if err != nil {
			log.Printf("Error scraping URL: %v", err)
			continue
		}

		// Save the main index page HTML
		indexHTMLPath := filepath.Join(htmlDir, fmt.Sprintf("index_%d.html", i+1))
		if err := scraper.SaveContentToFile(indexHTMLPath, htmlContent); err != nil {
			log.Printf("Error saving index HTML: %v", err)
		} else {
			log.Printf("Saved index HTML to %s", indexHTMLPath)
		}

		log.Println("Extracting standings links...")
		standingsLinks := scraper.ExtractStandingsLinks(htmlContent)

		// Convert relative links to absolute URLs
		var standingsURLs []string
		for _, link := range standingsLinks {
			absURL := scraper.ResolveRelativeURL(url, link)
			standingsURLs = append(standingsURLs, absURL)
		}

		log.Printf("Found %d standings links to process", len(standingsURLs))

		// Process each standings page
		for j, standingsURL := range standingsURLs {
			// Extract the week number from the URL
			week := j + 1 // Default: sequential weeks
			extractedWeek := scraper.ExtractWeekNumber(standingsURL)
			if extractedWeek > 0 {
				week = extractedWeek
			}

			log.Printf("Processing standings for Week %d: %s", week, standingsURL)

			// Define the local HTML file path
			localFilename := filepath.Join(htmlDir, fmt.Sprintf("standings_week_%d.html", week))
			var weeklyStats *models.WeeklyStats
			var htmlContent string

			// Try to use existing HTML file if available
			if fileContent, err := os.ReadFile(localFilename); err == nil {
				log.Printf("Using existing HTML file for week %d: %s", week, localFilename)
				htmlContent = string(fileContent)
			} else {
				// Download the HTML content if we don't have it locally
				log.Printf("Downloading HTML for week %d from %s", week, standingsURL)
				content, err := scraper.FetchURL(standingsURL)
				if err != nil {
					log.Printf("Error downloading standings page: %v", err)
					continue
				}

				// Save the downloaded HTML content
				htmlContent = content
				if err := scraper.SaveContentToFile(localFilename, htmlContent); err != nil {
					log.Printf("Error saving standings HTML: %v", err)
				} else {
					log.Printf("Saved standings HTML for week %d to %s", week, localFilename)
				}
			}

			// Extract player and team stats from the HTML content
			playerStats, teamStats := parser.ExtractPlayerStats(htmlContent)

			// Add opponent information to each player
			for i := range playerStats {
				opponent := parser.FindOpponent(playerStats[i].Team, week, schedules)
				playerStats[i].Opponent = opponent
			}

			// Create the weekly stats object
			weeklyStats = &models.WeeklyStats{
				Week:        week,
				PlayerStats: playerStats,
				TeamStats:   teamStats,
			}

			// Add to weekly stats collection
			allWeeklyStats = append(allWeeklyStats, weeklyStats)

			// Display the stats for this week with opponent information
			utils.DisplayWeeklyStatsWithOpponents(weeklyStats)

			// Save to CSV
			csvFilename := filepath.Join(csvDir, fmt.Sprintf("player_stats_week_%d.csv", week))
			err = utils.SaveWeeklyStatsToCSV(weeklyStats, csvFilename)
			if err != nil {
				log.Printf("Error saving CSV file: %v", err)
			} else {
				log.Printf("Saved player stats for week %d to %s", week, csvFilename)
			}
		}
	}

	log.Println("Scraping complete")
}
