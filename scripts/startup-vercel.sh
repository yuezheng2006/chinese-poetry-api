#!/bin/sh
set -e

# Configuration
DATA_DIR="data"
DB_FILE="poetry.db"
DB_PATH="${DATA_DIR}/${DB_FILE}"
DB_GZ="${DB_PATH}.gz"
CHECKSUM_FILE="${DATA_DIR}/checksums.txt"
GITHUB_RELEASE_URL="https://github.com/palemoky/chinese-poetry-api/releases/latest/download"

echo "=== Chinese Poetry API Startup (Vercel Optimized) ==="

# Create data directory if it doesn't exist
mkdir -p "${DATA_DIR}"

# Start server immediately in background to meet Vercel's 15s timeout requirement
echo "Starting API server in background..."
./server &
SERVER_PID=$!
echo "Server started with PID: $SERVER_PID"

# Function to download and verify database
download_database() {
    echo "[Background] Downloading database and checksums..."

    # Download both files
    if ! curl -Lfo "${DB_GZ}" "${GITHUB_RELEASE_URL}/${DB_FILE}.gz"; then
        echo "[Background] ERROR: Failed to download database"
        return 1
    fi

    if ! curl -Lfo "${CHECKSUM_FILE}" "${GITHUB_RELEASE_URL}/checksums.txt"; then
        echo "[Background] ERROR: Failed to download checksums"
        rm -f "${DB_GZ}"
        return 1
    fi

    # Verify downloaded .gz file
    echo "[Background] Verifying download integrity..."
    expected_checksum=$(grep "${DB_FILE}.gz" "${CHECKSUM_FILE}" | awk '{print $1}')

    if [ -z "$expected_checksum" ]; then
        echo "[Background] ERROR: Could not find checksum for ${DB_FILE}.gz"
        rm -f "${DB_GZ}" "${CHECKSUM_FILE}"
        return 1
    fi

    actual_checksum=$(sha256sum "${DB_GZ}" | awk '{print $1}')

    if [ "$actual_checksum" != "$expected_checksum" ]; then
        echo "[Background] ERROR: Checksum mismatch!"
        echo "  Expected: $expected_checksum"
        echo "  Actual:   $actual_checksum"
        rm -f "${DB_GZ}" "${CHECKSUM_FILE}"
        return 1
    fi

    echo "[Background] ✓ Download verified"

    # Extract database
    echo "[Background] Extracting ${DB_FILE}..."
    gunzip -f "${DB_GZ}"

    echo "[Background] ✓ Database ready: $DB_PATH"
    return 0
}

# Function to check for updates
check_for_updates() {
    echo "[Background] Checking for updates..."

    # Download latest checksums
    temp_checksum=$(mktemp)
    if ! curl -Lfo "$temp_checksum" "${GITHUB_RELEASE_URL}/checksums.txt"; then
        echo "[Background] Warning: Could not fetch latest checksums, skipping update check"
        rm -f "$temp_checksum"
        return 1
    fi

    # Compare with local checksums
    if [ -f "$CHECKSUM_FILE" ]; then
        if cmp -s "$temp_checksum" "$CHECKSUM_FILE"; then
            echo "[Background] ✓ Database is up to date"
            rm -f "$temp_checksum"
            return 0
        else
            echo "[Background] → New database version available"
            # Show what changed
            remote_checksum=$(grep "${DB_FILE}.gz" "$temp_checksum" | awk '{print $1}')
            local_checksum=$(grep "${DB_FILE}.gz" "$CHECKSUM_FILE" | awk '{print $1}')
            echo "  Local:  ${local_checksum:0:16}..."
            echo "  Remote: ${remote_checksum:0:16}..."
        fi
    fi

    rm -f "$temp_checksum"
    return 1
}

# Database management in background (non-blocking)
(
    if [ -f "$DB_PATH" ] && [ -f "$CHECKSUM_FILE" ]; then
        echo "[Background] Database found: $DB_PATH"

        # Check for updates
        if ! check_for_updates; then
            echo "[Background] Updating database..."
            download_database
        fi
    else
        echo "[Background] Database not found, downloading..."
        download_database
    fi
) &

DB_DOWNLOAD_PID=$!
echo "Database download started in background with PID: $DB_DOWNLOAD_PID"
echo "API server is running and ready to accept requests"

# Wait for the server process (main process)
# The database download continues in background
wait $SERVER_PID
