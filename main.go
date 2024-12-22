/*
Package main provides continuous monitoring of eBay listings.
It loads configuration from a file and saves results for tracking.
*/
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

// Color configurations for terminal output
var (
	titleColor   = color.New(color.FgCyan, color.Bold)
	priceColor   = color.New(color.FgGreen)
	auctionColor = color.New(color.FgYellow)
	buyNowColor  = color.New(color.FgBlue)
	watcherColor = color.New(color.FgMagenta)
	urlColor     = color.New(color.FgWhite, color.Underline)
	headerColor  = color.New(color.FgHiWhite, color.Bold)
)

/*
SavedItem represents a found listing with metadata about when it was discovered.
Used for persistent storage and tracking of listings over time.
*/
type SavedItem struct {
	Item      Item      `json:"item"`
	Found     time.Time `json:"found"`
	QueryTerm string    `json:"query"`
}

/*
Config holds the runtime configuration loaded from config.json.
Defines search criteria and monitoring behavior.
*/
type SearchConfig struct {
	Query       string      `json:"query"`
	ListingType ListingType `json:"listing_type"`
	MinPrice    float64     `json:"min_price"`
	MaxPrice    float64     `json:"max_price"`
	MinWatchers int         `json:"min_watchers"`
	MaxWatchers int         `json:"max_watchers"`
	MaxTimeLeft *TimeRange  `json:"max_time_left"`
}

type Config struct {
	CheckInterval int            `json:"check_interval_seconds"`
	Searches      []SearchConfig `json:"searches"`
}

// loadConfig reads and parses the configuration file
func loadConfig() (*Config, error) {
	// Try to load config.json
	data, err := os.ReadFile("config.json")
	if err != nil {
		// If config doesn't exist, try to copy template
		templateData, templateErr := os.ReadFile("config.template.json")
		if templateErr == nil {
			if copyErr := os.WriteFile("config.json", templateData, 0644); copyErr == nil {
				data = templateData
			}
		}
		if data == nil {
			return nil, err
		}
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// getDailyLogFile returns a file handle for today's log file
func getDailyLogFile() (*os.File, error) {
	today := time.Now().Format("2006-01-02")
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}
	return os.OpenFile(
		filepath.Join(logDir, fmt.Sprintf("findings_%s.json", today)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
}

// printItem displays a single item in the terminal with color formatting
func printItem(item Item, query string) {
	fmt.Printf("\n%s\n", strings.Repeat("-", 80))
	titleColor.Printf("Title: %s\n", item.Title)
	priceColor.Printf("Price: %s\n", item.Price)

	listingType := buyNowColor.Sprint("Buy Now")
	if item.IsAuction {
		listingType = auctionColor.Sprintf("Auction - %s remaining", item.TimeLeft)
	}

	fmt.Printf("Type: %s", listingType)
	if item.Watchers > 0 {
		watcherColor.Printf(" (%d watchers)", item.Watchers)
	}
	fmt.Println()

	urlColor.Printf("URL: %s\n", item.URL)
	headerColor.Printf("Query: %s\n", query)
}

// saveNewItems persists newly found items to both daily log and findings.json
func saveNewItems(items []Item, query string, seenItems map[string]bool) {
	// Save to findings.json
	file, err := os.OpenFile("findings.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening findings.json: %v", err)
		return
	}
	defer file.Close()

	// Get daily log file
	dailyLog, err := getDailyLogFile()
	if err != nil {
		log.Printf("Error opening daily log: %v", err)
		return
	}
	defer dailyLog.Close()

	encoder := json.NewEncoder(file)
	dailyEncoder := json.NewEncoder(dailyLog)
	now := time.Now()

	for _, item := range items {
		if !seenItems[item.URL] {
			seenItems[item.URL] = true
			savedItem := SavedItem{
				Item:      item,
				Found:     now,
				QueryTerm: query,
			}

			// Save to both files
			if err := encoder.Encode(savedItem); err != nil {
				log.Printf("Error saving to findings.json: %v", err)
			}
			if err := dailyEncoder.Encode(savedItem); err != nil {
				log.Printf("Error saving to daily log: %v", err)
			}

			// Print to terminal
			printItem(item, query)
		}
	}
}

// getFloat prompts for and validates floating point input
func getFloat(prompt string) float64 {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(prompt)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			return -1 // no limit
		}

		value, err := strconv.ParseFloat(input, 64)
		if err == nil && value >= 0 {
			return value
		}
		fmt.Println("Please enter a valid number or press enter for no limit")
	}
}

// getListingType prompts for and validates listing type selection
func getListingType() ListingType {
	for {
		fmt.Print("Select listing type:\n1. All\n2. Buy Now only\n3. Auctions only\nEnter choice (1-3): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			return All
		case "2":
			return BuyNow
		case "3":
			return Auction
		default:
			fmt.Println("Please enter a valid choice (1-3)")
		}
	}
}

// getMinWatchers prompts for and validates minimum watcher count
func getMinWatchers() int {
	for {
		fmt.Print("Enter minimum watchers (press enter for no limit): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			return -1
		}

		value, err := strconv.Atoi(input)
		if err == nil && value >= 0 {
			return value
		}
		fmt.Println("Please enter a valid number or press enter for no limit")
	}
}

// getMaxWatchers prompts for and validates maximum watcher count
func getMaxWatchers() int {
	for {
		fmt.Print("Enter maximum watchers (press enter for no limit): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			return -1 // no limit
		}

		value, err := strconv.Atoi(input)
		if err == nil && value >= 0 {
			return value
		}
		fmt.Println("Please enter a valid number or press enter for no limit")
	}
}

// getMaxTimeRemaining prompts for and validates time remaining limit
func getMaxTimeRemaining() *TimeRange {
	for {
		fmt.Print("Enter maximum time remaining (format: DD:HH:MM or enter for no limit): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			return nil
		}

		parts := strings.Split(input, ":")
		if len(parts) == 3 {
			days, errD := strconv.Atoi(parts[0])
			hours, errH := strconv.Atoi(parts[1])
			minutes, errM := strconv.Atoi(parts[2])

			if errD == nil && errH == nil && errM == nil &&
				days >= 0 && hours >= 0 && hours < 24 &&
				minutes >= 0 && minutes < 60 {
				return &TimeRange{
					Days:    days,
					Hours:   hours,
					Minutes: minutes,
				}
			}
		}
		fmt.Println("Please enter time in format DD:HH:MM or press enter for no limit")
	}
}

// promptForSearch asks user for search parameters and returns a SearchConfig
func promptForSearch() SearchConfig {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter search query: ")
	query, _ := reader.ReadString('\n')
	query = strings.TrimSpace(query)

	return SearchConfig{
		Query:       query,
		ListingType: getListingType(),
		MinPrice:    getFloat("Enter minimum price (press enter for no limit): "),
		MaxPrice:    getFloat("Enter maximum price (press enter for no limit): "),
		MinWatchers: getMinWatchers(),
		MaxWatchers: getMaxWatchers(),
		MaxTimeLeft: getMaxTimeRemaining(),
	}
}

// saveConfig writes the current configuration to config.json
func saveConfig(config *Config) error {
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile("config.json", data, 0644)
}

// main initializes and runs the continuous monitoring process
func main() {
	var config Config
	config.CheckInterval = 300 // Default check interval

	// Try to load existing config
	if existingConfig, err := loadConfig(); err == nil {
		fmt.Println("Found existing configuration.")
		fmt.Print("Do you want to (1) use existing config, (2) add new searches, or (3) start fresh? [1/2/3]: ")
		reader := bufio.NewReader(os.Stdin)
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			config = *existingConfig
		case "2":
			config = *existingConfig
			fmt.Println("Adding new searches to existing configuration...")
		case "3":
			fmt.Println("Starting with fresh configuration...")
		default:
			log.Fatal("Invalid choice")
		}
	}

	if len(config.Searches) == 0 {
		fmt.Print("How many items do you want to search for? ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		numSearches, _ := strconv.Atoi(strings.TrimSpace(input))

		for i := 0; i < numSearches; i++ {
			fmt.Printf("\nConfiguring search %d/%d:\n", i+1, numSearches)
			config.Searches = append(config.Searches, promptForSearch())
		}

		// Ask for check interval
		fmt.Print("\nHow often to check (in seconds)? [300]: ")
		input, _ = reader.ReadString('\n')
		interval := strings.TrimSpace(input)
		if interval != "" {
			if intervalInt, err := strconv.Atoi(interval); err == nil {
				config.CheckInterval = intervalInt
			}
		}
	}

	// Save the configuration
	if err := saveConfig(&config); err != nil {
		log.Printf("Warning: Could not save configuration: %v", err)
	} else {
		fmt.Println("Configuration saved to config.json")
	}

	// Continue with existing monitoring code
	seenItems := make(map[string]map[string]bool)
	for _, search := range config.Searches {
		seenItems[search.Query] = make(map[string]bool)
	}

	headerColor.Printf("Starting continuous monitoring for %d searches\n", len(config.Searches))
	headerColor.Printf("Checking every %d seconds\n", config.CheckInterval)
	headerColor.Printf("Saving results to findings.json and daily logs in ./logs/\n\n")

	for {
		for _, search := range config.Searches {
			scraper := NewScraper()
			scraper.ListingType = search.ListingType
			scraper.MinPrice = search.MinPrice
			scraper.MaxPrice = search.MaxPrice
			scraper.MaxTimeLeft = search.MaxTimeLeft

			results, err := scraper.ScrapeQuery(search.Query)
			if err != nil {
				log.Printf("Error scraping '%s': %v", search.Query, err)
				continue
			}

			var filteredResults []Item
			for _, item := range results {
				if (search.MinWatchers <= 0 || item.Watchers >= search.MinWatchers) &&
					(search.MaxWatchers <= 0 || item.Watchers <= search.MaxWatchers) {
					filteredResults = append(filteredResults, item)
				}
			}

			// Save new items
			saveNewItems(filteredResults, search.Query, seenItems[search.Query])

			// Print results for this search
			now := time.Now().Format("2006-01-02 15:04:05")
			newItems := 0
			for _, item := range filteredResults {
				if !seenItems[search.Query][item.URL] {
					newItems++
				}
			}

			if newItems > 0 {
				headerColor.Printf("\n[%s] Query '%s': Found %d new items!\n",
					now,
					search.Query,
					newItems)
			} else {
				headerColor.Printf("[%s] Query '%s': No new items\n",
					now,
					search.Query)
			}
		}

		time.Sleep(time.Duration(config.CheckInterval) * time.Second)
	}
}
