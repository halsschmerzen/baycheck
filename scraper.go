/*
Package main implements an eBay scraper that monitors listings based on various criteria.
It supports both auction and buy-now listings with customizable filters.
*/
package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

/*
ListingType represents the type of eBay listing.
It can be All, BuyNow, or Auction.
*/
type ListingType int

// Enumeration of listing types supported by the scraper
const (
	All     ListingType = iota // All listing types
	BuyNow                     // Buy Now listings only
	Auction                    // Auction listings only
)

/*
TimeRange represents a duration with days, hours, and minutes.
Used for tracking auction time remaining and setting time filters.
*/
type TimeRange struct {
	Days    int
	Hours   int
	Minutes int
}

/*
Scraper holds the configuration for filtering eBay listings.
It maintains criteria for prices, listing types, and time limits.
*/
type Scraper struct {
	MinPrice    float64
	MaxPrice    float64
	ListingType ListingType
	MaxTimeLeft *TimeRange
}

/*
Item represents a single eBay listing with all relevant information.
Includes both displayed information and parsed values for filtering.
*/
type Item struct {
	Title      string
	Price      string
	PriceValue float64
	URL        string
	IsAuction  bool
	Watchers   int
	TimeLeft   string
}

// NewScraper creates a new scraper instance with default settings
func NewScraper() *Scraper {
	return &Scraper{
		MinPrice:    -1,
		MaxPrice:    -1,
		ListingType: All,
		MaxTimeLeft: nil,
	}
}

// parsePrice extracts and normalizes the price from an eBay price string
func parsePrice(priceStr string) float64 {
	priceStr = strings.TrimPrefix(priceStr, "EUR")
	priceStr = strings.TrimSpace(priceStr)

	priceStr = strings.ReplaceAll(priceStr, ",", ".")

	re := regexp.MustCompile(`[^0-9.]`)
	cleanPrice := re.ReplaceAllString(priceStr, "")

	price, err := strconv.ParseFloat(cleanPrice, 64)
	if err != nil {
		return -1
	}
	return price
}

// cleanTitle removes common prefixes and normalizes the listing title
func cleanTitle(title string) string {
	title = strings.TrimPrefix(title, "Neues Angebot")
	title = strings.TrimSpace(title)
	return title
}

// isValidItem checks if a listing has all required fields and is not a promotional item
func isValidItem(title, price, url string) bool {
	if title == "" || price == "" || url == "" {
		return false
	}
	if strings.Contains(strings.ToLower(title), "shop on ebay") {
		return false
	}
	if strings.Contains(url, "itmmeta") {
		return false
	}
	return true
}

// isInPriceRange checks if an item's price falls within the configured range
func (s *Scraper) isInPriceRange(price float64) bool {
	if price < 0 {
		return false
	}
	if s.MinPrice >= 0 && price < s.MinPrice {
		return false
	}
	if s.MaxPrice >= 0 && price > s.MaxPrice {
		return false
	}
	return true
}

// isAuction determines if a listing is an auction based on eBay's HTML structure
func isAuction(selection *goquery.Selection) bool {
	// Check for auction-specific elements
	timeLeft := selection.Find(".s-item__time-left").Text()
	bids := selection.Find(".s-item__bids").Text()
	return timeLeft != "" || bids != ""
}

// shouldIncludeItem verifies if an item matches the configured listing type
func (s *Scraper) shouldIncludeItem(item Item) bool {
	switch s.ListingType {
	case BuyNow:
		if item.IsAuction {
			return false
		}
	case Auction:
		if !item.IsAuction {
			return false
		}
	}
	return true
}

// parseWatchers extracts the number of watchers from eBay's watcher text
func parseWatchers(watcherStr string) int {
	// Extract number from strings like "12 watchers"
	re := regexp.MustCompile(`(\d+)`)
	matches := re.FindStringSubmatch(watcherStr)
	if len(matches) > 1 {
		count, err := strconv.Atoi(matches[1])
		if err == nil {
			return count
		}
	}
	return 0
}

// parseTimeLeft converts eBay's time remaining text into a structured TimeRange
func parseTimeLeft(timeStr string) *TimeRange {
	if timeStr == "" {
		return nil
	}

	// Extract days, hours, and minutes
	daysRe := regexp.MustCompile(`(\d+)T`)      // Match "5T" format
	hoursRe := regexp.MustCompile(`(\d+)Std`)   // Match "12Std" format
	minsRe := regexp.MustCompile(`(\d+)\s*Min`) // Match "30 Min" format

	days := 0
	hours := 0
	mins := 0

	if matches := daysRe.FindStringSubmatch(timeStr); len(matches) > 1 {
		days, _ = strconv.Atoi(matches[1])
	}
	if matches := hoursRe.FindStringSubmatch(timeStr); len(matches) > 1 {
		hours, _ = strconv.Atoi(matches[1])
	}
	if matches := minsRe.FindStringSubmatch(timeStr); len(matches) > 1 {
		mins, _ = strconv.Atoi(matches[1])
	}

	// Handle "Noch XTage YStd" format
	if days == 0 && hours == 0 && mins == 0 {
		parts := strings.Fields(timeStr)
		for i, part := range parts {
			if strings.HasPrefix(part, "T") && i > 0 {
				days, _ = strconv.Atoi(parts[i-1])
			}
			if strings.HasPrefix(part, "Std") && i > 0 {
				hours, _ = strconv.Atoi(parts[i-1])
			}
		}
	}

	return &TimeRange{Days: days, Hours: hours, Minutes: mins}
}

// toMinutes converts a TimeRange into total minutes for comparison
func (tr *TimeRange) toMinutes() int {
	return (tr.Days * 24 * 60) + (tr.Hours * 60) + tr.Minutes
}

// isInTimeRange checks if an item's remaining time is within configured limits
func (s *Scraper) isInTimeRange(timeLeft *TimeRange) bool {
	if s.MaxTimeLeft == nil {
		return true
	}
	if timeLeft == nil {
		return false
	}

	maxMinutes := s.MaxTimeLeft.toMinutes()
	itemMinutes := timeLeft.toMinutes()

	return itemMinutes <= maxMinutes
}

// shouldCheckTime determines if time filtering should be applied
func (s *Scraper) shouldCheckTime() bool {
	return s.ListingType == Auction && s.MaxTimeLeft != nil
}

// Scrape performs the actual web scraping of eBay search results
func (s *Scraper) Scrape(url string) ([]Item, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var items []Item
	doc.Find(".s-item").Each(func(i int, selection *goquery.Selection) {
		title := selection.Find(".s-item__title").Text()
		price := selection.Find(".s-item__price").Text()
		url, _ := selection.Find("a.s-item__link").Attr("href")
		watchersText := selection.Find(".s-item__watchcount").Text()
		timeLeft := selection.Find(".s-item__time-left").Text()

		title = cleanTitle(title)
		priceValue := parsePrice(price)
		isAuction := isAuction(selection)
		watchers := parseWatchers(watchersText)

		item := Item{
			Title:      title,
			Price:      price,
			PriceValue: priceValue,
			URL:        url,
			IsAuction:  isAuction,
			Watchers:   watchers,
			TimeLeft:   timeLeft,
		}

		timeRange := parseTimeLeft(timeLeft)
		if isValidItem(title, price, url) &&
			s.isInPriceRange(priceValue) &&
			s.shouldIncludeItem(item) &&
			s.isInTimeRange(timeRange) {
			items = append(items, item)
		}
	})

	return items, nil
}

// ScrapeQuery constructs the eBay search URL and initiates scraping
func (s *Scraper) ScrapeQuery(query string) ([]Item, error) {
	url := fmt.Sprintf("https://www.ebay.de/sch/i.html?_nkw=%s", strings.ReplaceAll(query, " ", "+"))
	return s.Scrape(url)
}
