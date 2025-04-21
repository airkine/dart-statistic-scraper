# Dart Statistics Scraper

A Go application that scrapes and analyzes dart league statistics from websites and PDF files. The scraper extracts player stats, team standings, and match schedules from various sources, normalizes the data, and provides organized output.

## Features

- Downloads and parses PDF schedules to extract match information
- Scrapes HTML pages for player and team statistics
- Normalizes team names to handle variations in naming
- Extracts player performance metrics (PPD, MPR, games played/won, etc.)
- Associates players with their teams
- Identifies opponent teams for each match
- Organizes and displays statistics in a readable format
- Saves raw HTML and extracted data for verification and debugging

## Prerequisites

Before you can build and run the application, you need to have Go installed on your system (v1.20 or newer).

### Installing Go

1. Visit the [official Go download page](https://golang.org/dl/)
2. Download the appropriate installer for your operating system
3. Follow the installation instructions for your platform
4. Verify installation by opening a terminal/command prompt and typing:
   ```
   go version
   ```

## Getting Started

### Clone the Repository

```bash
git clone https://github.com/yourusername/dart-statistic-scraper.git
cd dart-statistic-scraper
```

### Build the Application

To build the application, run:

```bash
# Enable module-aware mode (modern Go projects use modules)
export GO111MODULE=on

# Format the code
go fmt ./...

# Run the linter
go vet ./...

# Install dependencies (if needed)
go mod tidy

# Build the binary with version information
go build -ldflags "-X main.version=$(git describe --tags)" -o bin/dart-scraper .
```

### Run the Application

After building, you can run the application:

```bash
# Run directly after building
./bin/dart-scraper

# Or if you've installed it to a location in your PATH
dart-scraper
```

## How It Works

The application performs several steps to gather and process dart league statistics:

1. **Download Schedule PDF**: Fetches the league schedule PDF from the source website
2. **Parse PDF Content**: Extracts match schedules from the PDF (teams, dates, weeks)
3. **Scrape League Standings**: Downloads HTML pages containing weekly standings and player stats
4. **Extract Player Statistics**: Parses HTML to extract player and team statistics
5. **Normalize Data**: Standardizes team names and formats to handle variations
6. **Calculate Matchups**: Associates players with their corresponding opponents for each week
7. **Display Results**: Outputs formatted statistics to the console

## Data Structure

The application works with several key data structures:

- **PlayerStat**: Individual player statistics (name, team, PPD, MPR, etc.)
- **TeamStat**: Team-level statistics (name, games played/won, PPD, MPR)
- **MatchSchedule**: Match scheduling information (week, date, home team, away team)
- **WeeklyStats**: Combined statistics for a specific week

## Project Structure

```
dart-statistic-scraper/
├── bin/                # Compiled binary files
├── cmd/                # Command-line entry points
├── csv/                # Exported CSV data files
├── html/               # Saved HTML files
├── internal/           # Internal application code
│   └── utils/          # Utility functions
├── pdf/                # PDF resources
├── pkg/                # Public library code
│   ├── models/         # Data models
│   ├── parser/         # Parsing logic
│   └── scraper/        # Web scraping functionality
├── test_output/        # Test output files
├── main.go             # Main application entry point
├── go.mod              # Go module definition
├── go.sum              # Go module checksums
├── Dockerfile          # Docker container definition
└── README.md           # This documentation file
```

## Docker Support

You can also build and run the application using Docker:

```bash
# Build the Docker image
docker build -t dart-scraper:latest .
docker build -t dart-scraper:sha-$(git rev-parse --short HEAD) .
docker build -t dart-scraper:$(git describe --tags) .

# Run the Docker container
docker run dart-scraper:latest
```

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

### Adding New Features

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run linters and tests
   ```bash
   go fmt ./...
   go vet ./...
   golangci-lint run
   go test ./...
   ```
5. Submit a pull request

## Troubleshooting

### Common Issues

- **HTML parsing errors**: The scraper relies on specific HTML structure. If the source website changes its layout, the parser may need to be updated.
- **PDF parsing issues**: PDF parsing can be sensitive to formatting changes. If the schedule PDF format changes, the extraction logic may need adjustments.
- **Network errors**: The application requires internet access to fetch data. Check your connection if you encounter network-related errors.

### Debugging

The application uses Go's standard logging package to output detailed information about its operations. Look for messages in the console output to help identify issues.

## License

[Specify your project license here]

## Acknowledgments

- [PuerkitoBio/goquery](https://github.com/PuerkitoBio/goquery) - For HTML parsing
- [ledongthuc/pdf](https://github.com/ledongthuc/pdf) - For PDF processing