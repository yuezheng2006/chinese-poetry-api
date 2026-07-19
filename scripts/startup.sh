#!/bin/sh
set -e

# Configuration
DATA_DIR="data"
DB_FILE="poetry.db"
DB_PATH="${DATA_DIR}/${DB_FILE}"
# Use Tencent Cloud COS for database storage (faster for China)
COS_DB_URL="https://poetry-db-beijing-1251898568.cos.ap-beijing.myqcloud.com/poetry.db"

echo "=== Chinese Poetry API Startup ==="

# Create data directory if it doesn't exist
mkdir -p "${DATA_DIR}"

# Function to download database from Tencent Cloud COS
download_database() {
    echo "Downloading database from Tencent Cloud COS (Beijing)..."
    echo "Source: ${COS_DB_URL}"
    echo "Target: ${DB_PATH}"

    # Download database file directly (uncompressed)
    if ! curl -Lfo "${DB_PATH}" "${COS_DB_URL}"; then
        echo "ERROR: Failed to download database from COS"
        rm -f "${DB_PATH}"
        exit 1
    fi

    # Verify file exists and is not empty
    if [ ! -f "$DB_PATH" ]; then
        echo "ERROR: Downloaded database file not found"
        exit 1
    fi

    if [ ! -s "$DB_PATH" ]; then
        echo "ERROR: Downloaded database is empty"
        rm -f "${DB_PATH}"
        exit 1
    fi

    # Get file size for logging
    db_size=$(du -h "$DB_PATH" | cut -f1)
    echo "✓ Database downloaded: $db_size"
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
