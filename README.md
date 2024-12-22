# baycheck

A powerful eBay scraper built with Go that continuously monitors listings based on customizable criteria. Perfect for finding deals and tracking auctions.

## Features

- 🔍 Filter by listing type (Buy Now/Auctions)
- 💰 Price range filtering
- ⏰ Time remaining filtering for auctions
- 👀 Watcher count filtering
- 🔄 Continuous monitoring
- 💾 Persistent storage of found items
- 🐳 Docker support

## Installation

### Local Installation

```bash
# Clone the repository
git clone https://github.com/halsschmerzen/baycheck.git
cd baycheck

# Install dependencies
go mod tidy
```

### Docker Installation

```bash
# Clone the repository
git clone https://github.com/halsschmerzen/baycheck.git
cd baycheck

# Build and run with Docker Compose
docker-compose up -d
```

## Configuration

1. Copy the template configuration:
```bash
cp config.template.json config.json
```

2. Edit `config.json` with your search parameters or run the program for interactive setup:
```bash
go run .
```

Example configuration:
```json
{
    "check_interval_seconds": 300,
    "searches": [
        {
            "query": "iPhone 14",
            "listing_type": 2,         # 1=All, 2=Buy Now, 3=Auctions
            "min_price": 400,          # -1 for no limit
            "max_price": 800,
            "min_watchers": -1,
            "max_watchers": -1,
            "max_time_left": {
                "days": 1,
                "hours": 12,
                "minutes": 0
            }
        }
    ]
}
```

## Usage

### Running Locally

```bash
go run .
```

### Running with Docker

```bash
docker-compose up -d    # Start in background
docker-compose logs -f  # Watch the logs
```

The scraper will:
1. Load configuration from `config.json`
2. Start monitoring eBay listings
3. Save new findings to `findings.json`
4. Display colored output in the terminal

## Output

Found items are:
- Displayed in the terminal with colored output
- Saved to `findings.json` for persistence
- Filtered to show only new items

## Contributing

Feel free to open issues or submit pull requests.

## License

MIT License

## Author

[@halsschmerzen](https://github.com/halsschmerzen)
