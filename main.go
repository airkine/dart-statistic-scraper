package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ledongthuc/pdf"
)

// PlayerStat holds statistics for a player
type PlayerStat struct {
	PlayerName   string
	Team         string
	Opponent     string // Added opponent team field
	SancPd       string
	GamesPlayed  int
	GamesWon     int
	PPD          float64
	MPR          float64
	HatTricks    int
	HighScore    int
	HighCheckout int
}

// MatchSchedule holds scheduling information for a match
type MatchSchedule struct {
	Week     int
	Date     string
	HomeTeam string
	AwayTeam string
}

// TeamStat holds statistics for a team
type TeamStat struct {
	TeamName    string
	GamesPlayed int
	GamesWon    int
	PPD         float64
	MPR         float64
}

// WeeklyStats holds the stats for a specific week
type WeeklyStats struct {
	Week        int
	PlayerStats []PlayerStat
	TeamStats   []TeamStat
}

// ScrapeURL downloads the HTML content from a URL and returns it as a string
func ScrapeURL(url string) (string, error) {
	// First normalize the URL to ensure proper formatting
	url = NormalizeURL(url)
	log.Printf("Scraping URL: %s", url)

	// Create an HTTP client with a timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Send the HTTP request
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("error fetching URL: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	log.Printf("HTTP Status: %d (%s)", resp.StatusCode, resp.Status)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-200 status code: %d %s", resp.StatusCode, resp.Status)
	}

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
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
		return fmt.Errorf("error fetching PDF: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-200 status code: %d %s", resp.StatusCode, resp.Status)
	}

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading PDF response body: %v", err)
	}

	// Write to file
	err = ioutil.WriteFile(localPath, body, 0644)
	if err != nil {
		return fmt.Errorf("error saving PDF to file: %v", err)
	}

	log.Printf("Successfully downloaded PDF to %s", localPath)
	return nil
}

// ReadPDFText reads a PDF file and returns its text content
func ReadPDFText(pdfPath string) (string, error) {
	// Open the PDF file
	f, r, err := pdf.Open(pdfPath)
	if err != nil {
		return "", fmt.Errorf("error opening PDF: %v", err)
	}
	defer f.Close()

	// Extract plain text from the PDF
	plainText, err := r.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("error extracting text from PDF: %v", err)
	}

	// Convert the bytes to string
	content, err := ioutil.ReadAll(plainText)
	if err != nil {
		return "", fmt.Errorf("error reading plain text from PDF: %v", err)
	}

	return string(content), nil
}

// ExtractScheduleFromText parses the raw text content from the PDF to extract schedule information
func ExtractScheduleFromText(text string) []MatchSchedule {
	var schedules []MatchSchedule

	// Split the text into lines
	lines := strings.Split(text, "\n")

	// Regular expression to match week numbers and dates
	weekDateRegex := regexp.MustCompile(`Week\s*(\d+)\s*-\s*(\w+\s*\d+\s*,\s*\d{4})`)

	// Regular expression to match team matchups
	// Looking for patterns like "TEAM A vs TEAM B" or "TEAM A @ TEAM B"
	matchupRegex := regexp.MustCompile(`([A-Z\s&']+)\s*(?:vs\.?|@|at)\s*([A-Z\s&']+)`)

	// Regular expression to match lines with a BYE
	byeRegex := regexp.MustCompile(`([A-Z\s&']+)\s*(?:BYE|bye|Bye)`)

	currentWeek := 0
	currentDate := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check if line contains week and date information
		weekDateMatch := weekDateRegex.FindStringSubmatch(line)
		if len(weekDateMatch) > 2 {
			weekNum, err := strconv.Atoi(weekDateMatch[1])
			if err == nil {
				currentWeek = weekNum
				currentDate = weekDateMatch[2]
				log.Printf("Found Week %d - %s", currentWeek, currentDate)
				continue
			}
		}

		// First, check for BYE entries
		byeMatch := byeRegex.FindStringSubmatch(line)
		if len(byeMatch) > 1 && currentWeek > 0 {
			team := strings.TrimSpace(byeMatch[1])
			// Create match schedule entry with BYE as the away team
			schedule := MatchSchedule{
				Week:     currentWeek,
				Date:     currentDate,
				HomeTeam: team,
				AwayTeam: "BYE",
			}
			schedules = append(schedules, schedule)
			log.Printf("Week %d: %s vs BYE", currentWeek, team)
			continue
		}

		// Check if line contains regular matchup information
		matchupMatches := matchupRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matchupMatches {
			if len(match) > 2 && currentWeek > 0 {
				homeTeam := strings.TrimSpace(match[1])
				awayTeam := strings.TrimSpace(match[2])

				// Special case: if one of the teams is "BYE", handle it specially
				if strings.ToUpper(homeTeam) == "BYE" || strings.ToUpper(awayTeam) == "BYE" {
					log.Printf("Found BYE match in Week %d: %s vs %s", currentWeek, homeTeam, awayTeam)
				}

				// Create match schedule entry
				schedule := MatchSchedule{
					Week:     currentWeek,
					Date:     currentDate,
					HomeTeam: homeTeam,
					AwayTeam: awayTeam,
				}

				schedules = append(schedules, schedule)
				log.Printf("Week %d: %s vs %s", currentWeek, homeTeam, awayTeam)
			}
		}
	}

	return schedules
}

// ParseScheduleManually creates a hardcoded schedule based on known patterns
// This is a fallback in case the PDF parsing doesn't work properly
func ParseScheduleManually() []MatchSchedule {
	var schedules []MatchSchedule

	// Team names in the league
	teams := []string{
		"THE HUTCH",
		"CAPITALIZE",
		"GRAND AVE",
		"HARBOR HILLS",
		"HARBOR HILLS TOO",
		"HILLS HAS EYES",
		"REDHEADS",
		"SIR JAMES PUB DOS",
		"SPEARS N BEERS",
	}

	// Create a simplified schedule for the first 26 weeks
	// This is an approximation - in reality we'd need the exact schedule from the PDF
	for week := 1; week <= 26; week++ {
		// Create some matchups for this week
		// In a real schedule, each team plays exactly one other team per week
		for i := 0; i < len(teams); i += 2 {
			// Skip if we don't have enough teams for a pair
			if i+1 >= len(teams) {
				continue
			}

			// Create the matchup (alternating home/away)
			var homeTeam, awayTeam string
			if week%2 == 0 {
				homeTeam = teams[i]
				awayTeam = teams[i+1]
			} else {
				homeTeam = teams[i+1]
				awayTeam = teams[i]
			}

			// Create date string (we don't have actual dates, so use placeholder)
			date := fmt.Sprintf("Week %d, 2024", week)

			// Create match schedule entry
			schedule := MatchSchedule{
				Week:     week,
				Date:     date,
				HomeTeam: homeTeam,
				AwayTeam: awayTeam,
			}

			schedules = append(schedules, schedule)
		}
	}

	return schedules
}

// FindOpponent returns the opponent team for a given team in a specific week
func FindOpponent(team string, week int, schedules []MatchSchedule) string {
	for _, schedule := range schedules {
		if schedule.Week == week {
			// Normalize team name for comparison
			normTeam := NormalizeTeamName(team)
			normHomeTeam := NormalizeTeamName(schedule.HomeTeam)
			normAwayTeam := NormalizeTeamName(schedule.AwayTeam)

			// Special case for BYE entries
			if strings.ToUpper(schedule.AwayTeam) == "BYE" && normTeam == normHomeTeam {
				return "BYE"
			}

			if strings.ToUpper(schedule.HomeTeam) == "BYE" && normTeam == normAwayTeam {
				return "BYE"
			}

			if normTeam == normHomeTeam {
				return schedule.AwayTeam
			} else if normTeam == normAwayTeam {
				return schedule.HomeTeam
			}
		}
	}
	return "Unknown"
}

// NormalizeTeamName standardizes team names for comparison
func NormalizeTeamName(name string) string {
	// First, preserve original name for specific case handling
	originalName := strings.ToUpper(name)

	// Special handling for Bridge Inn teams - must be checked first
	if strings.Contains(originalName, "BRIDGE INN 1") ||
		(strings.Contains(originalName, "BRIDGE INN") && strings.Contains(originalName, "1")) {
		return "BRIDGE INN 1" // Return with spaces preserved
	} else if strings.Contains(originalName, "BRIDGE INN 2") ||
		(strings.Contains(originalName, "BRIDGE INN") && strings.Contains(originalName, "2")) {
		return "BRIDGE INN 2" // Return with spaces preserved
	}

	// Special handling for Sir James Pub teams
	if strings.Contains(originalName, "SIR JAMES PUB 1") ||
		(strings.Contains(originalName, "SIR JAMES PUB") && strings.Contains(originalName, "1") && !strings.Contains(originalName, "DOS")) {
		return "SIR JAMES PUB 1"
	} else if strings.Contains(originalName, "SIR JAMES PUB 2") ||
		(strings.Contains(originalName, "SIR JAMES PUB") && (strings.Contains(originalName, "2") || strings.Contains(originalName, "DOS")) && !strings.Contains(originalName, "3")) {
		return "SIR JAMES PUB 2"
	} else if strings.Contains(originalName, "SIR JAMES PUB 3") ||
		(strings.Contains(originalName, "SIR JAMES PUB") && strings.Contains(originalName, "3")) {
		return "SIR JAMES PUB 3"
	}

	// Remove spaces, convert to uppercase, and remove non-alphanumeric chars
	name = strings.ToUpper(name)
	name = regexp.MustCompile(`[^A-Z0-9]`).ReplaceAllString(name, "")

	// Replace common abbreviations/alternatives
	replacements := map[string]string{
		"THEHUTCH":       "THE HUTCH",
		"HARBORHILLSTOO": "HARBOR HILLS TOO",
		"HARBORHILLS2":   "HARBOR HILLS TOO",
		"HARBORHILLSTWO": "HARBOR HILLS TOO",
		"HILLSHASEYES":   "HILLS HAS EYES",
		"EYESOFTHEHILL":  "HILLS HAS EYES",
		"SIRJAMESPUBDOS": "SIR JAMES PUB 2",
		"SIRJAMESPUB":    "SIR JAMES PUB",
		"SPEARSNBEERS":   "SPEARS N BEERS",
	}

	for k, v := range replacements {
		if strings.Contains(name, k) {
			return v
		}
	}

	return originalName
}

// NormalizeURL ensures a URL is properly formatted with correct protocol slashes
func NormalizeURL(url string) string {
	// Fix common protocol formatting issues
	if strings.HasPrefix(url, "https:/") && !strings.HasPrefix(url, "https://") {
		url = "https://" + strings.TrimPrefix(url, "https:/")
	} else if strings.HasPrefix(url, "http:/") && !strings.HasPrefix(url, "http://") {
		url = "http://" + strings.TrimPrefix(url, "http:/")
	}

	// Ensure spaces are properly encoded
	url = strings.ReplaceAll(url, " ", "%20")

	return url
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

// isTeamNameLine checks if a line contains just a team name (usually all caps with no stats)
func isTeamNameLine(line string) bool {
	// Team names are usually all caps, don't contain numbers (except for Bridge Inn 1/2), and are standalone
	line = strings.TrimSpace(line)

	// Team names are typically not very long
	if len(line) < 3 || len(line) > 40 {
		return false
	}

	// Special case for Bridge Inn team names which contain numbers
	if strings.Contains(strings.ToUpper(line), "BRIDGE INN") {
		return true
	}

	// Check if it's mostly uppercase letters and spaces
	hasLetter := false
	hasNumber := false
	for _, char := range line {
		if char >= '0' && char <= '9' {
			hasNumber = true
		}
		if char >= 'A' && char <= 'Z' {
			hasLetter = true
		}
	}

	// For most teams, they shouldn't have numbers in their names
	// But we'll allow up to 1 number (for teams with a number designation)
	if hasNumber && !strings.Contains(line, "1") && !strings.Contains(line, "2") {
		return false
	}

	// Must have at least one letter and not contain typical non-team text
	return hasLetter &&
		!strings.Contains(line, "Player") &&
		!strings.Contains(line, "Team Totals") &&
		!strings.Contains(line, "PPD") &&
		!strings.Contains(line, "MPR") &&
		!strings.Contains(line, "Wins") &&
		!strings.Contains(line, "Games")
}

// extractTeamName extracts a team name from a line
func extractTeamName(line string) string {
	// Clean up the line to get just the team name
	teamName := strings.TrimSpace(line)
	teamName = strings.Replace(teamName, "Team:", "", 1)

	// Special case for Bridge Inn teams
	upperLine := strings.ToUpper(teamName)
	if strings.Contains(upperLine, "BRIDGE INN") {
		if strings.Contains(upperLine, "1") || strings.Contains(upperLine, "I") && !strings.Contains(upperLine, "II") {
			return "BRIDGE INN 1"
		} else if strings.Contains(upperLine, "2") || strings.Contains(upperLine, "II") {
			return "BRIDGE INN 2"
		}
	}

	// Special case for Sir James Pub teams
	if strings.Contains(upperLine, "SIR JAMES PUB") {
		if strings.Contains(upperLine, "1") && !strings.Contains(upperLine, "DOS") {
			return "SIR JAMES PUB 1"
		} else if strings.Contains(upperLine, "2") || strings.Contains(upperLine, "DOS") && !strings.Contains(upperLine, "3") {
			return "SIR JAMES PUB 2"
		} else if strings.Contains(upperLine, "3") {
			return "SIR JAMES PUB 3"
		}
	}

	// Remove any extra garbage
	re := regexp.MustCompile(`[^\w\s]`)
	teamName = re.ReplaceAllString(teamName, "")

	return strings.TrimSpace(teamName)
}

// parsePlayerStatsLine parses a line of text into player stats
func parsePlayerStatsLine(line string) PlayerStat {
	var playerStat PlayerStat

	// Split the line into fields (accounting for variable whitespace)
	fields := regexp.MustCompile(`\s+`).Split(line, -1)

	// Need at least 7 fields for valid player data
	if len(fields) < 7 {
		return playerStat
	}

	// Determine which fields are which
	// This is somewhat heuristic as the data format can vary

	// First field usually contains the player name
	playerStat.PlayerName = fields[0]

	// Look for second field that might be a rating like "AA", "A", "B" etc.
	ratingIndex := -1
	for i := 1; i < len(fields); i++ {
		if isPlayerRating(fields[i]) {
			ratingIndex = i
			playerStat.SancPd = fields[i]
			break
		}
	}

	if ratingIndex == -1 {
		// If no rating field found, assume standard layout
		if len(fields) > 1 {
			playerStat.SancPd = fields[1]
		}

		// Try to parse numeric fields
		if len(fields) > 2 {
			playerStat.GamesPlayed, _ = strconv.Atoi(fields[2])
		}
		if len(fields) > 3 {
			playerStat.GamesWon, _ = strconv.Atoi(fields[3])
		}
		if len(fields) > 4 {
			playerStat.PPD, _ = strconv.ParseFloat(fields[4], 64)
		}
		if len(fields) > 5 {
			playerStat.MPR, _ = strconv.ParseFloat(fields[5], 64)
		}
		if len(fields) > 6 {
			playerStat.HatTricks, _ = strconv.Atoi(fields[6])
		}
		if len(fields) > 7 {
			playerStat.HighScore, _ = strconv.Atoi(fields[7])
		}
		if len(fields) > 8 {
			playerStat.HighCheckout, _ = strconv.Atoi(fields[8])
		}
	} else {
		// If we found the rating field, parse from there
		if ratingIndex+1 < len(fields) {
			playerStat.GamesPlayed, _ = strconv.Atoi(fields[ratingIndex+1])
		}
		if ratingIndex+2 < len(fields) {
			playerStat.GamesWon, _ = strconv.Atoi(fields[ratingIndex+2])
		}
		if ratingIndex+3 < len(fields) {
			playerStat.PPD, _ = strconv.ParseFloat(fields[ratingIndex+3], 64)
		}
		if ratingIndex+4 < len(fields) {
			playerStat.MPR, _ = strconv.ParseFloat(fields[ratingIndex+4], 64)
		}
		if ratingIndex+5 < len(fields) {
			playerStat.HatTricks, _ = strconv.Atoi(fields[ratingIndex+5])
		}
		if ratingIndex+6 < len(fields) {
			playerStat.HighScore, _ = strconv.Atoi(fields[ratingIndex+6])
		}
		if ratingIndex+7 < len(fields) {
			playerStat.HighCheckout, _ = strconv.Atoi(fields[ratingIndex+7])
		}
	}

	return playerStat
}

// isNumeric checks if a string contains only numeric data
func isNumeric(s string) bool {
	// Check if this looks like a number
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// isPlayerRating checks if a string looks like a player rating (AA, A, BB, B, etc.)
func isPlayerRating(s string) bool {
	// Player ratings are usually 1-2 characters from A-Z
	if len(s) > 3 {
		return false
	}

	// Must be all uppercase A-Z
	for _, c := range s {
		if c < 'A' || c > 'Z' {
			return false
		}
	}

	// Common patterns: A, AA, B, BB, etc.
	return true
}

// parseTeamTotalsLine parses a team totals line into team stats
func parseTeamTotalsLine(line string) TeamStat {
	var teamStat TeamStat

	// Check if this is actually a team totals line
	if !strings.Contains(line, "Team Totals:") {
		return teamStat
	}

	// Extract the numeric fields
	fields := regexp.MustCompile(`\s+`).Split(line, -1)

	// Need at least 5 fields for valid team data (Team Totals, Games, Wins, PPD, MPR)
	if len(fields) < 5 {
		return teamStat
	}

	// Find the actual data fields (after "Team Totals:")
	var dataFields []string
	foundTotals := false
	for _, field := range fields {
		if foundTotals {
			if field != "" {
				dataFields = append(dataFields, field)
			}
		} else if strings.Contains(field, "Totals:") {
			foundTotals = true
		}
	}

	// Check if we have enough data fields
	if len(dataFields) < 4 {
		return teamStat
	}

	// Parse the team data
	teamStat.TeamName = "TEAM" // Will be replaced with actual team name later
	teamStat.GamesPlayed, _ = strconv.Atoi(dataFields[0])
	teamStat.GamesWon, _ = strconv.Atoi(dataFields[1])
	teamStat.PPD, _ = strconv.ParseFloat(dataFields[2], 64)
	teamStat.MPR, _ = strconv.ParseFloat(dataFields[3], 64)

	return teamStat
}

// Helper function to sanitize numeric strings by removing non-numeric characters except decimal points
func sanitizeNumberString(s string) string {
	s = strings.TrimSpace(s)
	result := ""
	for _, c := range s {
		if (c >= '0' && c <= '9') || c == '.' {
			result += string(c)
		}
	}
	return result
}

// ExtractPlayerStats extracts player statistics from the HTML content
func ExtractPlayerStats(htmlContent string) ([]PlayerStat, []TeamStat) {
	var playerStats []PlayerStat
	var teamStats []TeamStat
	var teamName string

	log.Println("Extracting player stats from HTML...")

	// Look for the Combined X01/Cricket games section
	startMarker := "Combined X01/Cricket games, sorted by Team + PPD:"
	endMarker := "Most Improved Players for week"

	startIndex := strings.Index(htmlContent, startMarker)
	if startIndex == -1 {
		// Try alternative markers if the exact one is not found
		alternatePossibleMarkers := []string{
			"All X01 games, sorted by PPD:",
			"X01/Cricket games, sorted by Team",
			"Combined X01/Cricket games",
			"X01 games, sorted by PPD",
		}

		for _, marker := range alternatePossibleMarkers {
			startIndex = strings.Index(htmlContent, marker)
			if startIndex != -1 {
				log.Printf("Using alternative start marker: '%s'", marker)
				break
			}
		}

		if startIndex == -1 {
			log.Printf("No suitable start marker found in HTML")
			return playerStats, teamStats
		}
	}

	endIndex := strings.Index(htmlContent[startIndex:], endMarker)
	if endIndex == -1 {
		// If end marker not found, try to go to the end of the document
		endIndex = len(htmlContent) - startIndex
		log.Printf("End marker not found, using rest of document (%d bytes)", endIndex)
	} else {
		endIndex += startIndex // Adjust for the substring offset
	}

	// Extract the section between markers
	sectionHTML := htmlContent[startIndex:endIndex]
	log.Printf("Found player stats section (length: %d characters)", len(sectionHTML))

	// For debugging, save this section to a file
	err := saveToFile("player_stats_section.html", sectionHTML)
	if err == nil {
		log.Printf("Saved player stats section to player_stats_section.html")
	}

	// Parse the HTML section with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(sectionHTML))
	if err != nil {
		log.Printf("Error parsing player stats section: %v", err)
		return playerStats, teamStats
	}

	// Try direct extraction from table structures first
	playerStats = extractPlayerStatsFromTable(doc, teamName)

	// If no players found, try line-by-line parsing
	if len(playerStats) == 0 {
		log.Println("Table extraction found no players, trying line-by-line parsing...")

		// Process the HTML to extract player stats
		lines := strings.Split(sectionHTML, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)

			// If line contains a team name (usually in all caps with no other data)
			if isTeamNameLine(line) {
				teamName = extractTeamName(line)
				log.Printf("Found team: %s", teamName)
				continue
			}

			// Skip empty lines and header lines
			if line == "" || strings.Contains(line, "Player") ||
				strings.Contains(line, "-----") || strings.Contains(line, "Team Totals:") {
				continue
			}

			// Try to parse a player stat line
			playerStat := parsePlayerStatsLine(line)
			if playerStat.PlayerName != "" {
				playerStat.Team = teamName
				playerStats = append(playerStats, playerStat)
				log.Printf("Added player: %s (Team: %s, PPD: %.2f)",
					playerStat.PlayerName, playerStat.Team, playerStat.PPD)
			}

			// Check for team totals line
			if strings.Contains(line, "Team Totals:") {
				teamStat := parseTeamTotalsLine(line)
				if teamStat.TeamName != "" {
					teamStat.TeamName = teamName
					teamStats = append(teamStats, teamStat)
					log.Printf("Added team totals for: %s (PPD: %.2f)", teamStat.TeamName, teamStat.PPD)
				}
			}
		}
	}

	// Post-processing to correct team assignments for specific players
	for i := range playerStats {
		// Special case for Steve Wheelock - always assign to Bridge Inn 2
		if strings.ToUpper(playerStats[i].PlayerName) == "STEVE WHEELOCK" {
			playerStats[i].Team = "BRIDGE INN 2"
			log.Printf("Reassigned %s to team: %s", playerStats[i].PlayerName, playerStats[i].Team)
		}
	}

	log.Printf("Extracted %d player stats and %d team stats", len(playerStats), len(teamStats))
	return playerStats, teamStats
}

// extractPlayerStatsFromTable attempts to extract player stats from tables in the document
func extractPlayerStatsFromTable(doc *goquery.Document, defaultTeam string) []PlayerStat {
	var playerStats []PlayerStat

	// Find all tables in the document
	doc.Find("table").Each(func(i int, table *goquery.Selection) {
		log.Printf("Analyzing table #%d", i)

		// Check if this table has player stats headers
		headers := []string{}
		table.Find("tr:first-child td, tr:first-child th").Each(func(j int, cell *goquery.Selection) {
			headerText := strings.TrimSpace(cell.Text())
			headers = append(headers, headerText)
		})

		// Check if headers match player stats structure
		hasPlayerColumn := false
		hasPPDColumn := false
		teamNameFromHeader := ""

		for _, header := range headers {
			if strings.Contains(header, "Player") {
				hasPlayerColumn = true
			}
			if strings.Contains(header, "PPD") {
				hasPPDColumn = true
				// Check if the header contains a team name
			}
			if strings.Contains(header, "BRIDGE INN") {
				if strings.Contains(header, "1") {
					teamNameFromHeader = "BRIDGE INN 1"
				} else if strings.Contains(header, "2") {
					teamNameFromHeader = "BRIDGE INN 2"
				} else {
					teamNameFromHeader = "BRIDGE INN"
				}
			}
		}

		if !hasPlayerColumn || !hasPPDColumn {
			log.Printf("Table #%d doesn't appear to be a player stats table", i)
			return
		}

		log.Printf("Found potential player stats table #%d with headers: %v", i, headers)

		// Extract player rows
		var currentTeam string = defaultTeam
		// If we found a team name in the header, use it as the initial team name
		if teamNameFromHeader != "" {
			currentTeam = teamNameFromHeader
			log.Printf("Using team name from header: %s", currentTeam)
		}

		table.Find("tr").Each(func(rowIdx int, row *goquery.Selection) {
			// Skip header row
			if rowIdx == 0 {
				return
			}

			cells := row.Find("td")

			// Check if this is a team header row (usually has fewer cells)
			if cells.Length() <= 3 {
				teamText := strings.TrimSpace(row.Text())
				if isTeamNameLine(teamText) {
					currentTeam = teamText
					log.Printf("Found team name row: %s", currentTeam)
					return
				}
			}

			// Must have at least 7 cells for a valid player row
			if cells.Length() < 7 {
				return
			}

			// Extract cell text
			cellTexts := []string{}
			cells.Each(func(cellIdx int, cell *goquery.Selection) {
				// Get all text from cell and its children
				cellText := strings.TrimSpace(cell.Text())
				cellTexts = append(cellTexts, cellText)
			})

			// Must have content in first cell (player name)
			if len(cellTexts) == 0 || cellTexts[0] == "" ||
				cellTexts[0] == "Player" || strings.Contains(cellTexts[0], "Team Totals") {
				return
			}

			// Skip header rows
			if strings.Contains(strings.ToLower(cellTexts[0]), "player") {
				return
			}

			// Create player stat object
			playerStat := PlayerStat{
				PlayerName: cellTexts[0],
				Team:       currentTeam,
			}

			// Parse remaining fields
			if len(cellTexts) > 1 {
				playerStat.SancPd = cellTexts[1]
			}
			if len(cellTexts) > 2 {
				playerStat.GamesPlayed, _ = strconv.Atoi(sanitizeNumberString(cellTexts[2]))
			}
			if len(cellTexts) > 3 {
				playerStat.GamesWon, _ = strconv.Atoi(sanitizeNumberString(cellTexts[3]))
			}
			if len(cellTexts) > 4 {
				playerStat.PPD, _ = strconv.ParseFloat(sanitizeNumberString(cellTexts[4]), 64)
			}
			if len(cellTexts) > 5 {
				playerStat.MPR, _ = strconv.ParseFloat(sanitizeNumberString(cellTexts[5]), 64)
			}
			if len(cellTexts) > 6 {
				playerStat.HatTricks, _ = strconv.Atoi(sanitizeNumberString(cellTexts[6]))
			}
			if len(cellTexts) > 7 {
				playerStat.HighScore, _ = strconv.Atoi(sanitizeNumberString(cellTexts[7]))
			}
			if len(cellTexts) > 8 {
				playerStat.HighCheckout, _ = strconv.Atoi(sanitizeNumberString(cellTexts[8]))
			}

			// Only add valid player data
			if playerStat.PlayerName != "" && playerStat.PlayerName != "Combined" {
				playerStats = append(playerStats, playerStat)
				log.Printf("Added player from table: %s (Team: %s, Games: %d, PPD: %.2f)",
					playerStat.PlayerName, playerStat.Team, playerStat.GamesPlayed, playerStat.PPD)
			}
		})
	})

	// Try direct parsing of the HTML content as an alternative approach
	if len(playerStats) == 0 {
		log.Println("Attempting direct HTML parsing for player stats...")

		// Find rows that look like player data
		doc.Find("tr").Each(func(i int, row *goquery.Selection) {
			// Get all text in the row
			rowText := strings.TrimSpace(row.Text())

			// Skip irrelevant rows
			if rowText == "" ||
				strings.Contains(strings.ToLower(rowText), "player") ||
				strings.Contains(strings.ToLower(rowText), "team totals") {
				return
			}

			// Check if row contains player data by looking for common names
			if strings.Contains(rowText, "MITCH") ||
				strings.Contains(rowText, "STEVE") ||
				strings.Contains(rowText, "JOHN") ||
				strings.Contains(rowText, "MIKE") {

				// Extract all cell contents
				var cellTexts []string
				row.Find("td").Each(func(j int, cell *goquery.Selection) {
					cellText := strings.TrimSpace(cell.Text())
					cellTexts = append(cellTexts, cellText)
				})

				if len(cellTexts) >= 7 {
					playerStat := PlayerStat{
						PlayerName: cellTexts[0],
						Team:       defaultTeam,
					}

					if len(cellTexts) > 1 {
						playerStat.SancPd = cellTexts[1]
					}
					if len(cellTexts) > 2 {
						playerStat.GamesPlayed, _ = strconv.Atoi(sanitizeNumberString(cellTexts[2]))
					}
					if len(cellTexts) > 3 {
						playerStat.GamesWon, _ = strconv.Atoi(sanitizeNumberString(cellTexts[3]))
					}
					if len(cellTexts) > 4 {
						playerStat.PPD, _ = strconv.ParseFloat(sanitizeNumberString(cellTexts[4]), 64)
					}
					if len(cellTexts) > 5 {
						playerStat.MPR, _ = strconv.ParseFloat(sanitizeNumberString(cellTexts[5]), 64)
					}
					if len(cellTexts) > 6 {
						playerStat.HatTricks, _ = strconv.Atoi(sanitizeNumberString(cellTexts[6]))
					}
					if len(cellTexts) > 7 {
						playerStat.HighScore, _ = strconv.Atoi(sanitizeNumberString(cellTexts[7]))
					}
					if len(cellTexts) > 8 {
						playerStat.HighCheckout, _ = strconv.Atoi(sanitizeNumberString(cellTexts[8]))
					}

					playerStats = append(playerStats, playerStat)
					log.Printf("Added player from direct HTML: %s (Games: %d, PPD: %.2f)",
						playerStat.PlayerName, playerStat.GamesPlayed, playerStat.PPD)
				}
			}
		})
	}

	return playerStats
}

// saveToFile saves content to a file
func saveToFile(filename, content string) error {
	return ioutil.WriteFile(filename, []byte(content), 0644)
}

// processStandingsPage processes a single standings page
func processStandingsPage(url string, week int) (*WeeklyStats, error) {
	// Download the HTML content
	htmlContent, err := ScrapeURL(url)
	if err != nil {
		return nil, fmt.Errorf("error scraping URL: %v", err)
	}

	// Save the HTML content to a file for debugging
	filename := fmt.Sprintf("standings_week_%d.html", week)
	err = saveToFile(filename, htmlContent)
	if err != nil {
		log.Printf("Error saving standings HTML: %v", err)
	} else {
		log.Printf("Saved standings HTML for week %d to %s", week, filename)
	}

	// Extract player and team stats
	playerStats, teamStats := ExtractPlayerStats(htmlContent)

	// Create a WeeklyStats object
	weeklyStats := &WeeklyStats{
		Week:        week,
		PlayerStats: playerStats,
		TeamStats:   teamStats,
	}

	log.Printf("Successfully extracted %d player stats from %s", len(playerStats), url)

	return weeklyStats, nil
}

// displayWeeklyStats prints the player statistics for a given week
func displayWeeklyStats(weeklyStats *WeeklyStats) {
	fmt.Printf("\n=========== PLAYER STATISTICS FOR WEEK %d ===========\n", weeklyStats.Week)
	fmt.Printf("%-26s | %-6s | %-5s | %-4s | %-6s | %-5s | %-3s | %-6s | %-6s\n",
		"Player", "SancPd", "Games", "Wins", "PPD", "MPR", "Hat", "HstTon", "HstOut")
	fmt.Printf("%-26s | %-6s | %-5s | %-4s | %-6s | %-5s | %-3s | %-6s | %-6s\n",
		strings.Repeat("-", 26), strings.Repeat("-", 6), strings.Repeat("-", 5),
		strings.Repeat("-", 4), strings.Repeat("-", 6), strings.Repeat("-", 5),
		strings.Repeat("-", 3), strings.Repeat("-", 6), strings.Repeat("-", 6))

	// Group players by team
	teamPlayers := make(map[string][]PlayerStat)
	for _, player := range weeklyStats.PlayerStats {
		teamPlayers[player.Team] = append(teamPlayers[player.Team], player)
	}

	// Get all team names and sort them
	var teamNames []string
	for team := range teamPlayers {
		teamNames = append(teamNames, team)
	}
	sort.Strings(teamNames)

	// Print players by team, sorted by PPD within each team
	for _, team := range teamNames {
		players := teamPlayers[team]

		// Sort players by PPD (descending)
		sort.Slice(players, func(i, j int) bool {
			return players[i].PPD > players[j].PPD
		})

		// Print team name
		if team != "" {
			fmt.Printf("\n%s\n", team)
		}

		// Print player stats
		for _, player := range players {
			fmt.Printf("%-26s | %-6s | %5d | %4d | %6.2f | %5.2f | %3d | %6d | %6d\n",
				player.PlayerName, player.SancPd, player.GamesPlayed, player.GamesWon,
				player.PPD, player.MPR, player.HatTricks, player.HighScore, player.HighCheckout)
		}
	}

	fmt.Println(strings.Repeat("=", 78))
}

// displayWeeklyStatsWithOpponents prints the player statistics for a given week including opponent information
func displayWeeklyStatsWithOpponents(weeklyStats *WeeklyStats) {
	fmt.Printf("\n=========== PLAYER STATISTICS FOR WEEK %d ===========\n", weeklyStats.Week)
	fmt.Printf("%-26s | %-6s | %-15s | %-5s | %-4s | %-6s | %-5s | %-3s | %-6s | %-6s\n",
		"Player", "SancPd", "Opponent", "Games", "Wins", "PPD", "MPR", "Hat", "HstTon", "HstOut")
	fmt.Printf("%-26s | %-6s | %-15s | %-5s | %-4s | %-6s | %-5s | %-3s | %-6s | %-6s\n",
		strings.Repeat("-", 26), strings.Repeat("-", 6), strings.Repeat("-", 15),
		strings.Repeat("-", 5), strings.Repeat("-", 4), strings.Repeat("-", 6),
		strings.Repeat("-", 5), strings.Repeat("-", 3), strings.Repeat("-", 6),
		strings.Repeat("-", 6))

	// Group players by team
	teamPlayers := make(map[string][]PlayerStat)
	for _, player := range weeklyStats.PlayerStats {
		teamPlayers[player.Team] = append(teamPlayers[player.Team], player)
	}

	// Get all team names and sort them
	var teamNames []string
	for team := range teamPlayers {
		teamNames = append(teamNames, team)
	}
	sort.Strings(teamNames)

	// Print players by team, sorted by PPD within each team
	for _, team := range teamNames {
		players := teamPlayers[team]

		// Sort players by PPD (descending)
		sort.Slice(players, func(i, j int) bool {
			return players[i].PPD > players[j].PPD
		})

		// Print team name
		if team != "" {
			fmt.Printf("\n%s\n", team)
		}

		// Print player stats
		for _, player := range players {
			fmt.Printf("%-26s | %-6s | %-15s | %5d | %4d | %6.2f | %5.2f | %3d | %6d | %6d\n",
				player.PlayerName, player.SancPd, player.Opponent, player.GamesPlayed, player.GamesWon,
				player.PPD, player.MPR, player.HatTricks, player.HighScore, player.HighCheckout)
		}
	}

	fmt.Println(strings.Repeat("=", 78))
}

func main() {
	log.Println("Dart Standings Scraper starting...")

	// PDF schedule URL
	scheduleURL := "https://macdleagues.com/DartSchedules/FALL2024Schedules/FALL2024%2024SUN1.pdf"
	localPDFPath := "fall2024_schedule.pdf"

	// First, attempt to download and process the schedule PDF
	var schedules []MatchSchedule
	log.Printf("Attempting to download schedule PDF from %s", scheduleURL)

	err := DownloadPDF(scheduleURL, localPDFPath)
	if err != nil {
		log.Printf("Error downloading PDF schedule: %v. Using fallback manual schedule.", err)
		schedules = ParseScheduleManually()
	} else {
		// Read and parse the PDF text
		pdfText, err := ReadPDFText(localPDFPath)
		if err != nil {
			log.Printf("Error reading PDF text: %v. Using fallback manual schedule.", err)
			schedules = ParseScheduleManually()
		} else {
			// Extract schedule information from the PDF text
			schedules = ExtractScheduleFromText(pdfText)

			// If no schedules were extracted, fall back to manual parsing
			if len(schedules) == 0 {
				log.Printf("No schedules extracted from PDF. Using fallback manual schedule.")
				schedules = ParseScheduleManually()
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
	for i, url := range urls {
		log.Printf("Processing URL %d of %d: %s", i+1, len(urls), url)

		// Download and extract standings links
		htmlContent, err := ScrapeURL(url)
		if err != nil {
			log.Printf("Error scraping URL: %v", err)
			continue
		}

		log.Println("Extracting standings links...")
		standingsLinks := ExtractStandingsLinks(htmlContent)

		// Convert relative links to absolute URLs
		baseURL := filepath.Dir(url) + "/"
		var standingsURLs []string
		for _, link := range standingsLinks {
			// Ensure proper URL construction
			var absURL string
			if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
				// Link is already absolute
				absURL = link
			} else if strings.HasPrefix(link, "//") {
				// Protocol-relative URL
				absURL = "https:" + link
			} else if strings.HasPrefix(link, "/") {
				// Site-root-relative URL
				urlParts := strings.Split(url, "://")
				if len(urlParts) > 1 {
					protocol := urlParts[0]
					hostWithPath := urlParts[1]
					host := strings.Split(hostWithPath, "/")[0]
					absURL = protocol + "://" + host + link
				} else {
					// Fallback in case of malformed URL
					absURL = "https://macdleagues.com" + link
				}
			} else {
				// Path-relative URL
				absURL = baseURL + link
			}

			// Ensure URL doesn't have spaces (encode them)
			absURL = NormalizeURL(absURL)

			log.Printf("Converted link: %s to absolute URL: %s", link, absURL)
			standingsURLs = append(standingsURLs, absURL)
		}

		log.Printf("Found %d standings links to process", len(standingsURLs))

		// Process each standings page
		var weeklyStats []*WeeklyStats

		for j, standingsURL := range standingsURLs {
			// Extract the week number from the URL
			week := j + 1
			re := regexp.MustCompile(`Wk(\d+)`)
			matches := re.FindStringSubmatch(standingsURL)
			if len(matches) > 1 {
				weekNum, err := strconv.Atoi(matches[1])
				if err == nil {
					week = weekNum
				}
			}

			log.Printf("Fetching standings link %d of %d: %s (Week: %d)", j+1, len(standingsURLs), standingsURL, week)

			// Skip live fetching for testing if we already have the file locally
			localFilename := fmt.Sprintf("standings_week_%d.html", week)
			if _, err := os.Stat(localFilename); err == nil {
				log.Printf("Using local file for week %d", week)
				htmlContent, err := ioutil.ReadFile(localFilename)
				if err == nil {
					playerStats, teamStats := ExtractPlayerStats(string(htmlContent))

					// Add opponent information to each player
					for i := range playerStats {
						opponent := FindOpponent(playerStats[i].Team, week, schedules)
						playerStats[i].Opponent = opponent
					}

					stats := &WeeklyStats{
						Week:        week,
						PlayerStats: playerStats,
						TeamStats:   teamStats,
					}
					weeklyStats = append(weeklyStats, stats)

					// Use the new display function that includes opponent information
					displayWeeklyStatsWithOpponents(stats)
					continue
				}
			}

			// Process the standings page
			stats, err := processStandingsPage(standingsURL, week)
			if err != nil {
				log.Printf("Error processing standings page: %v", err)
				continue
			}

			// Add opponent information to each player
			for i := range stats.PlayerStats {
				opponent := FindOpponent(stats.PlayerStats[i].Team, week, schedules)
				stats.PlayerStats[i].Opponent = opponent
			}

			// Add to weekly stats
			weeklyStats = append(weeklyStats, stats)

			// Display the stats for this week with opponent information
			displayWeeklyStatsWithOpponents(stats)
		}
	}

	log.Println("Scraping complete")
}
