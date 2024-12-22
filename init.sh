#!/bin/sh

# Create logs directory
mkdir -p /app/logs

# Create empty findings.json if it doesn't exist
if [ ! -f /app/findings.json ]; then
    echo "[]" > /app/findings.json
fi

# Create default config.json if it doesn't exist
if [ ! -f /app/config.json ]; then
    echo '{
        "check_interval_seconds": 300,
        "searches": [