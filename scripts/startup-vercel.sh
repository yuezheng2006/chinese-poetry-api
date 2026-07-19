#!/bin/sh
set -e

# Configuration
DATA_DIR="data"
DB_FILE="poetry.db"
DB_PATH="${DATA_DIR}/${DB_FILE}"
# Use Tencent Cloud COS for database storage (faster for China)
COS_DB_URL="https://poetry-db-beijing-1251898568.cos.ap-beijing.myqcloud.com/poetry.db"
MAX_WAIT=5  # Maximum seconds to wait for database before starting server (quick start, download continues in background)

echo "=== Chinese Poetry API Startup (Vercel Optimized) ==="

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
        return 1
    fi

    # Verify file exists and is not empty
    if [ ! -f "$DB_PATH" ]; then
        echo "ERROR: Downloaded database file not found"
        return 1
    fi

    if [ ! -s "$DB_PATH" ]; then
        echo "ERROR: Downloaded database is empty"
        rm -f "${DB_PATH}"
        return 1
    fi

    # Get file size for logging
    db_size=$(du -h "$DB_PATH" | cut -f1)
    echo "✓ Database downloaded: $db_size"

    echo "✓ Database ready: $DB_PATH"
    return 0
}

# Check if database exists and has content
if [ -f "$DB_PATH" ] && [ -s "$DB_PATH" ]; then
    echo "✓ Database found: $DB_PATH"
    db_size=$(du -h "$DB_PATH" | cut -f1)
    echo "  Size: $db_size"
else
    echo "Database not found or empty, downloading (with ${MAX_WAIT}s timeout for Vercel)..."

    # Start download in background with timeout
    (
        download_database
    ) &
    DOWNLOAD_PID=$!

    # Wait for download with timeout
    WAITED=0
    while [ $WAITED -lt $MAX_WAIT ]; do
        if [ -f "$DB_PATH" ] && [ -s "$DB_PATH" ]; then
            echo "✓ Database downloaded successfully in ${WAITED}s"
            break
        fi

        # Check if download process is still running
        if ! kill -0 $DOWNLOAD_PID 2>/dev/null; then
            echo "⚠ Download process completed"
            break
        fi

        sleep 1
        WAITED=$((WAITED + 1))
    done

    # If database still doesn't exist after timeout, start server anyway
    # (it will create an empty DB, but at least server will be responsive)
    if [ ! -f "$DB_PATH" ] || [ ! -s "$DB_PATH" ]; then
        echo "⚠ Warning: Database not ready after ${MAX_WAIT}s"
        echo "⚠ Starting server anyway (database will continue downloading in background)"
    fi
fi

# Start server (foreground)
echo "Starting API server on port ${PORT:-80}..."
exec ./server
