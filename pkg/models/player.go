// Package models contains data structures for dart league statistics
package models

// PlayerStat holds statistics for a player
type PlayerStat struct {
	PlayerName   string
	Team         string
	Opponent     string
	SancPd       string
	GamesPlayed  int
	GamesWon     int
	PPD          float64
	MPR          float64
	HatTricks    int
	HighScore    int
	HighCheckout int
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

// MatchSchedule holds scheduling information for a match
type MatchSchedule struct {
	Week     int
	Date     string
	HomeTeam string
	AwayTeam string
}
