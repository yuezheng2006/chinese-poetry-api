package database

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/palemoky/chinese-poetry-api/internal/classifier"
)

// DB wraps the gorm.DB connection
type DB struct {
	*gorm.DB
}

// Open opens a connection to the SQLite database using GORM
// maxOpenConns: maximum number of open connections (0 = use default of 1 for safety)
// maxIdleConns: maximum number of idle connections (0 = use default of 1)
func Open(path string, maxOpenConns, maxIdleConns int) (*DB, error) {
	// Configure GORM
	config := &gorm.Config{
		Logger:  logger.Default.LogMode(logger.Silent), // Change to logger.Info for debugging
		NowFunc: time.Now,
		// Prepare statements for better performance
		PrepareStmt: true,
	}

	// SQLite connection string with optimizations for concurrent writes
	// _busy_timeout: wait up to 5 seconds if database is locked
	// _journal_mode=WAL: Write-Ahead Logging for better concurrency
	// _synchronous=NORMAL: balance between safety and performance
	// cache=shared: allow multiple connections to share cache
	// _cache_size=-64000: 64MB page cache (negative = KB, positive = pages)
	// _temp_store=MEMORY: use memory for temporary tables and indices
	dsn := path + "?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&cache=shared&_cache_size=-64000&_temp_store=MEMORY"

	// Open database with GORM SQLite driver
	db, err := gorm.Open(sqlite.Open(dsn), config)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Get underlying sql.DB for connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Set connection pool settings
	// Default to 1 connection for safety (data processing)
	// Can be increased for read-heavy API serving
	if maxOpenConns <= 0 {
		maxOpenConns = 1 // Safe default for write-heavy workloads
	}
	if maxIdleConns <= 0 {
		maxIdleConns = 1
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
}

// NewDBFromGorm wraps an existing gorm.DB connection.
// This is useful for testing with custom database configurations.
func NewDBFromGorm(db *gorm.DB) *DB {
	return &DB{db}
}

// Migrate creates all tables, indexes, and initial data for both language variants
func (db *DB) Migrate() error {
	// Create metadata table first
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS metadata (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`).Error; err != nil {
		return fmt.Errorf("failed to create metadata table: %w", err)
	}

	// Create tables for both language variants
	for _, lang := range []Lang{LangHans, LangHant} {
		if err := db.migrateTablesForLang(lang); err != nil {
			return fmt.Errorf("failed to migrate tables for %s: %w", lang, err)
		}

		// Insert initial data for this language variant
		if err := db.insertInitialDataForLang(lang); err != nil {
			return fmt.Errorf("failed to insert initial data for %s: %w", lang, err)
		}
	}

	// Update schema version
	if err := db.Exec(
		`INSERT OR REPLACE INTO metadata (key, value, updated_at) VALUES (?, ?, ?)`,
		"schema_version",
		fmt.Sprintf("%d", SchemaVersion),
		time.Now(),
	).Error; err != nil {
		return fmt.Errorf("failed to update schema version: %w", err)
	}

	return nil
}

// migrateTablesForLang creates all tables for a specific language variant
func (db *DB) migrateTablesForLang(lang Lang) error {
	dynastyTable := dynastiesTable(lang)
	authorTable := authorsTable(lang)
	poetryTypeTable := poetryTypesTable(lang)
	poemTable := poemsTable(lang)

	// Create dynasties table
	dynastySQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		name_en TEXT,
		start_year INTEGER,
		end_year INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`, dynastyTable)
	if err := db.Exec(dynastySQL).Error; err != nil {
		return fmt.Errorf("failed to create %s: %w", dynastyTable, err)
	}

	// Create authors table
	authorSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		dynasty_id INTEGER,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (dynasty_id) REFERENCES %s(id)
	)`, authorTable, dynastyTable)
	if err := db.Exec(authorSQL).Error; err != nil {
		return fmt.Errorf("failed to create %s: %w", authorTable, err)
	}
	// Create index on dynasty_id
	db.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_dynasty ON %s(dynasty_id)", authorTable, authorTable))

	// Create poetry_types table
	poetryTypeSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		category TEXT NOT NULL,
		lines INTEGER,
		chars_per_line INTEGER,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`, poetryTypeTable)
	if err := db.Exec(poetryTypeSQL).Error; err != nil {
		return fmt.Errorf("failed to create %s: %w", poetryTypeTable, err)
	}

	// Create poems table
	poemSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY,
		type_id INTEGER,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		content_hash TEXT,
		author_id INTEGER,
		dynasty_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (type_id) REFERENCES %s(id),
		FOREIGN KEY (author_id) REFERENCES %s(id),
		FOREIGN KEY (dynasty_id) REFERENCES %s(id)
	)`, poemTable, poetryTypeTable, authorTable, dynastyTable)
	if err := db.Exec(poemSQL).Error; err != nil {
		return fmt.Errorf("failed to create %s: %w", poemTable, err)
	}

	// Create indexes for poems
	db.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_type ON %s(type_id)", poemTable, poemTable))
	db.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_title ON %s(title)", poemTable, poemTable))
	db.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_author ON %s(author_id)", poemTable, poemTable))
	db.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_dynasty ON %s(dynasty_id)", poemTable, poemTable))
	db.Exec(fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS idx_%s_unique ON %s(title, content_hash)", poemTable, poemTable))
	// Composite index for efficient multi-type random selection (type_id IN ... with id range lookups)
	db.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_type_id ON %s(type_id, id)", poemTable, poemTable))

	if err := db.migrateFtsForLang(lang); err != nil {
		return err
	}

	return nil
}

// migrateFtsForLang creates the FTS5 virtual table (and sync triggers) that backs
// full-text search for a poems table, then backfills it if it was just created.
//
// The table is a self-contained (not external-content) FTS5 index: content_text is
// a derived value (the poem's paragraphs flattened out of poems.content, which is a
// JSON array), not a literal column on the poems table, and FTS5's external-content
// mode requires reading column values back from a same-named column on the linked
// table. Duplicating the indexed text costs some extra disk space, but keeps the
// index simple and correct. Triggers below keep it in sync with every
// INSERT/UPDATE/DELETE on the poems table, including the ON CONFLICT upserts used
// by the loader.
//
// The trigram tokenizer is used instead of the default unicode61 tokenizer because
// Chinese text has no whitespace word boundaries, so the standard tokenizer can't
// segment it meaningfully. Trigram indexing lets `col LIKE '%...%'` (arbitrary
// substring queries, including single/double-character CJK terms) be accelerated by
// the FTS index while keeping the exact same substring-match semantics the API
// already exposes via SearchPoems.
func (db *DB) migrateFtsForLang(lang Lang) error {
	poemTable := poemsTable(lang)
	ftsTable := poemsFtsTable(lang)

	// Detect whether this is a brand-new table (needs a one-time backfill) or one
	// that already existed (kept in sync incrementally by triggers, so a full
	// rebuild would just be wasted work every time Migrate runs).
	var existingCount int64
	if err := db.Raw(
		`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, ftsTable,
	).Scan(&existingCount).Error; err != nil {
		return fmt.Errorf("failed to check for existing %s: %w", ftsTable, err)
	}

	ftsSQL := fmt.Sprintf(`CREATE VIRTUAL TABLE IF NOT EXISTS %s USING fts5(
		title,
		content_text,
		tokenize='trigram'
	)`, ftsTable)
	if err := db.Exec(ftsSQL).Error; err != nil {
		return fmt.Errorf(
			"failed to create %s (requires SQLite built with FTS5 support, e.g. the sqlite_fts5 build tag): %w",
			ftsTable, err,
		)
	}

	// content_text is the plain concatenation of the poem's paragraphs, extracted
	// from the JSON array stored in poems.content, so the index (and LIKE queries
	// against it) match on readable text rather than raw JSON punctuation.
	const contentTextExpr = `(SELECT COALESCE(group_concat(value, ''), '') FROM json_each(%s.content))`

	insertTrigger := fmt.Sprintf(`CREATE TRIGGER IF NOT EXISTS %[2]s_fts_ai AFTER INSERT ON %[2]s BEGIN
		INSERT INTO %[1]s(rowid, title, content_text)
		VALUES (new.id, new.title, `+fmt.Sprintf(contentTextExpr, "new")+`);
	END`, ftsTable, poemTable)
	if err := db.Exec(insertTrigger).Error; err != nil {
		return fmt.Errorf("failed to create insert trigger for %s: %w", ftsTable, err)
	}

	// Note: the fts5 special "INSERT INTO fts(fts, rowid, ...) VALUES ('delete', ...)"
	// marker syntax only applies to external-content tables; this table is
	// self-contained (see comment above), so removal is a plain DELETE by rowid.
	deleteTrigger := fmt.Sprintf(`CREATE TRIGGER IF NOT EXISTS %[2]s_fts_ad AFTER DELETE ON %[2]s BEGIN
		DELETE FROM %[1]s WHERE rowid = old.id;
	END`, ftsTable, poemTable)
	if err := db.Exec(deleteTrigger).Error; err != nil {
		return fmt.Errorf("failed to create delete trigger for %s: %w", ftsTable, err)
	}

	updateTrigger := fmt.Sprintf(`CREATE TRIGGER IF NOT EXISTS %[2]s_fts_au AFTER UPDATE ON %[2]s BEGIN
		DELETE FROM %[1]s WHERE rowid = old.id;
		INSERT INTO %[1]s(rowid, title, content_text)
		VALUES (new.id, new.title, `+fmt.Sprintf(contentTextExpr, "new")+`);
	END`, ftsTable, poemTable)
	if err := db.Exec(updateTrigger).Error; err != nil {
		return fmt.Errorf("failed to create update trigger for %s: %w", ftsTable, err)
	}

	// Backfill only on first creation: existing rows predate the triggers, but a
	// table that already existed is already up to date, so skip the (expensive,
	// full-corpus) rebuild on every subsequent migration run.
	//
	// FTS5's built-in 'rebuild' command only works when the fts5 table's columns
	// share names with real columns on the content table, which content_text
	// (a derived expression, not a stored column) does not, so backfill manually
	// with the same expression the triggers use.
	if existingCount == 0 {
		backfillSQL := fmt.Sprintf(`INSERT INTO %[1]s(rowid, title, content_text)
			SELECT id, title, `+fmt.Sprintf(contentTextExpr, poemTable)+`
			FROM %[2]s`, ftsTable, poemTable)
		if err := db.Exec(backfillSQL).Error; err != nil {
			return fmt.Errorf("failed to backfill %s: %w", ftsTable, err)
		}
	}

	return nil
}

// insertInitialDataForLang inserts initial data for a specific language variant
func (db *DB) insertInitialDataForLang(lang Lang) error {
	dynastyTable := dynastiesTable(lang)
	poetryTypeTable := poetryTypesTable(lang)

	// Prepare SQL - convert to traditional if needed
	dynastiesSQL := strings.ReplaceAll(InitialDynastiesSQL, "dynasties", dynastyTable)
	poetryTypesSQL := strings.ReplaceAll(InitialPoetryTypesSQL, "poetry_types", poetryTypeTable)

	if lang == LangHant {
		var err error
		dynastiesSQL, err = convertSQLToTraditional(dynastiesSQL)
		if err != nil {
			return fmt.Errorf("failed to convert dynasties SQL: %w", err)
		}
		poetryTypesSQL, err = convertSQLToTraditional(poetryTypesSQL)
		if err != nil {
			return fmt.Errorf("failed to convert poetry types SQL: %w", err)
		}
	}

	// Insert dynasties
	if err := db.Exec(dynastiesSQL).Error; err != nil {
		return fmt.Errorf("failed to insert dynasties: %w", err)
	}

	// Insert poetry types
	if err := db.Exec(poetryTypesSQL).Error; err != nil {
		return fmt.Errorf("failed to insert poetry types: %w", err)
	}

	return nil
}

// convertSQLToTraditional converts Chinese characters in SQL string to traditional
// Preserves SQL syntax and only converts Chinese text within quotes
func convertSQLToTraditional(sql string) (string, error) {
	// Split by single quotes to find string literals
	parts := strings.Split(sql, "'")

	for i := range parts {
		// Only convert odd-indexed parts (inside quotes)
		if i%2 == 1 {
			converted, err := classifier.ToTraditional(parts[i])
			if err != nil {
				return "", err
			}
			parts[i] = converted
		}
	}

	return strings.Join(parts, "'"), nil
}

// GetSchemaVersion returns the current schema version
func (db *DB) GetSchemaVersion() (int, error) {
	var version int
	err := db.Raw(`SELECT value FROM metadata WHERE key = ?`, "schema_version").Scan(&version).Error
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return version, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
