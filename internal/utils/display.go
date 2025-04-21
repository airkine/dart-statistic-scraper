// Package utils provides utility functions for the dart-statistic-scraper
package utils

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/myusername/dart-statistic-scraper/pkg/models"
)

// DisplayWeeklyStatsWithOpponents prints the player statistics for a given week including opponent information
func DisplayWeeklyStatsWithOpponents(weeklyStats *models.WeeklyStats) {
	fmt.Printf("\n=========== PLAYER STATISTICS FOR WEEK %d ===========\n", weeklyStats.Week)
	fmt.Printf("%-26s | %-6s | %-15s | %-5s | %-4s | %-6s | %-5s | %-3s | %-6s | %-6s\n",
		"Player", "SancPd", "Opponent", "Games", "Wins", "PPD", "MPR", "Hat", "HstTon", "HstOut")
	fmt.Printf("%-26s | %-6s | %-15s | %-5s | %-4s | %-6s | %-5s | %-3s | %-6s | %-6s\n",
		strings.Repeat("-", 26), strings.Repeat("-", 6), strings.Repeat("-", 15),
		strings.Repeat("-", 5), strings.Repeat("-", 4), strings.Repeat("-", 6),
		strings.Repeat("-", 5), strings.Repeat("-", 3), strings.Repeat("-", 6),
		strings.Repeat("-", 6))

	// Group players by team
	teamPlayers := make(map[string][]models.PlayerStat)
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

// SaveWeeklyStatsToCSV saves the player statistics for a given week to a CSV file
func SaveWeeklyStatsToCSV(weeklyStats *models.WeeklyStats, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	// Write CSV header
	_, err = fmt.Fprintf(f, "Week,Player,Team,Opponent,SancPd,GamesPlayed,GamesWon,PPD,MPR,HatTricks,HighScore,HighCheckout\n")
	if err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write player stats
	for _, player := range weeklyStats.PlayerStats {
		_, err = fmt.Fprintf(f, "%d,%s,%s,%s,%s,%d,%d,%.2f,%.2f,%d,%d,%d\n",
			weeklyStats.Week, player.PlayerName, player.Team, player.Opponent, player.SancPd,
			player.GamesPlayed, player.GamesWon, player.PPD, player.MPR, player.HatTricks,
			player.HighScore, player.HighCheckout)
		if err != nil {
			return fmt.Errorf("failed to write player data: %w", err)
		}
	}

	return nil
}
