#!/bin/sh
set -e

# Configuration
DATA_DIR="data"
DB_FILE="poetry.db"
DB_PATH="${DATA_DIR}/${DB_FILE}"
DB_GZ="${DB_PATH}.gz"
CHECKSUM_FILE="${DATA_DIR}/checksums.txt"
GITHUB_RELEASE_URL="https://github.com/palemoky/chinese-poetry-api/releases/latest/download"
MAX_WAIT=10  # Maximum seconds to wait for database before starting server

echo "=== Chinese Poetry API Startup (Vercel Optimized) ==="

# Create data directory if it doesn't exist
mkdir -p "${DATA_DIR}"

# Function to download and verify database
download_database() {
    echo "Downloading database and checksums..."

    # Download both files
    if ! curl -Lfo "${DB_GZ}" "${GITHUB_RELEASE_URL}/${DB_FILE}.gz"; then
        echo "ERROR: Failed to download database"
        return 1
    fi

    if ! curl -Lfo "${CHECKSUM_FILE}" "${GITHUB_RELEASE_URL}/checksums.txt"; then
        echo "ERROR: Failed to download checksums"
        rm -f "${DB_GZ}"
        return 1
    fi

    # Verify downloaded .gz file
    echo "Verifying download integrity..."
    expected_checksum=$(grep "${DB_FILE}.gz" "${CHECKSUM_FILE}" | awk '{print $1}')

    if [ -z "$expected_checksum" ]; then
        echo "ERROR: Could not find checksum for ${DB_FILE}.gz"
        rm -f "${DB_GZ}" "${CHECKSUM_FILE}"
        return 1
    fi

    actual_checksum=$(sha256sum "${DB_GZ}" | awk '{print $1}')

    if [ "$actual_checksum" != "$expected_checksum" ]; then
        echo "ERROR: Checksum mismatch!"
        echo "  Expected: $expected_checksum"
        echo "  Actual:   $actual_checksum"
        rm -f "${DB_GZ}" "${CHECKSUM_FILE}"
        return 1
    fi

    echo "✓ Download verified"

    # Extract database using gzip -d -c (more reliable than gunzip)
    echo "Extracting ${DB_FILE}..."
    if ! gzip -d -c "${DB_GZ}" > "${DB_PATH}"; then
        echo "ERROR: Failed to extract database"
        rm -f "${DB_PATH}"
        return 1
    fi

    # Verify extracted file is not empty
    if [ ! -s "$DB_PATH" ]; then
        echo "ERROR: Extracted database is empty"
        rm -f "${DB_PATH}"
        return 1
    fi

    # Get file size for logging
    db_size=$(du -h "$DB_PATH" | cut -f1)
    echo "✓ Database extracted: $db_size"

    # Clean up .gz file after successful extraction
    rm -f "${DB_GZ}"

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
