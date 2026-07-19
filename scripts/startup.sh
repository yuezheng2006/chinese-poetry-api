#!/bin/sh
set -e

# Configuration
DATA_DIR="data"
DB_FILE="poetry.db"
DB_PATH="${DATA_DIR}/${DB_FILE}"
DB_GZ="${DB_PATH}.gz"
# Use GitHub Release (global CDN, faster from international locations)
GITHUB_RELEASE_URL="https://github.com/palemoky/chinese-poetry-api/releases/latest/download"

echo "=== Chinese Poetry API Startup ==="

# Create data directory if it doesn't exist
mkdir -p "${DATA_DIR}"

# Function to download database from GitHub Release
download_database() {
    echo "Downloading database from GitHub Release (global CDN)..."
    echo "Source: ${GITHUB_RELEASE_URL}/${DB_FILE}.gz"
    echo "Target: ${DB_PATH}"

    # Download compressed database file
    if ! curl -Lfo "${DB_GZ}" "${GITHUB_RELEASE_URL}/${DB_FILE}.gz"; then
        echo "ERROR: Failed to download database from GitHub"
        rm -f "${DB_GZ}"
        exit 1
    fi

    # Verify compressed file exists and is not empty
    if [ ! -f "$DB_GZ" ] || [ ! -s "$DB_GZ" ]; then
        echo "ERROR: Downloaded compressed file not found or empty"
        rm -f "${DB_GZ}"
        exit 1
    fi

    # Decompress database
    echo "Decompressing database..."
    if ! gunzip -f "${DB_GZ}"; then
        echo "ERROR: Failed to decompress database"
        rm -f "${DB_GZ}"
        exit 1
    fi

    # Verify decompressed file
    if [ ! -f "$DB_PATH" ] || [ ! -s "$DB_PATH" ]; then
        echo "ERROR: Decompressed database file not found or empty"
        exit 1
    fi

    # Get file size for logging
    db_size=$(du -h "$DB_PATH" | cut -f1)
    echo "✓ Database downloaded and decompressed: $db_size"
    echo "✓ Database ready: $DB_PATH"
}

# Main logic - simple: if database exists, use it; otherwise download
if [ -f "$DB_PATH" ] && [ -s "$DB_PATH" ]; then
    echo "✓ Database found: $DB_PATH"
    db_size=$(du -h "$DB_PATH" | cut -f1)
    echo "  Size: $db_size"
else
    echo "Database not found or empty, downloading..."
    download_database
fi

echo "Starting API server on port ${PORT:-1279}..."
exec ./server
