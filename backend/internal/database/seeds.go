package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"
)

//go:embed seeds/**/*.sql seeds/*/*.sql
var SeedFiles embed.FS

type SeedRunner struct {
	db *sql.DB
}

func NewSeedRunner(db *sql.DB) *SeedRunner {
	return &SeedRunner{db: db}
}

// normalizePair converts "en-es" to "en_es" format for filesystem lookup.
func normalizePair(pair string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(pair)), "-", "_")
}

// Run executes all pending SQL seeds for the specified language pair (e.g. "en_es" or "en-es").
func (r *SeedRunner) Run(ctx context.Context, pair string) ([]string, error) {
	norm := normalizePair(pair)
	pattern := fmt.Sprintf("seeds/%s/*.sql", norm)
	entries, err := fs.Glob(SeedFiles, pattern)
	if err != nil {
		return nil, fmt.Errorf("glob seeds for pair %s: %w", pair, err)
	}
	if len(entries) == 0 {
		// Also try recursive match in case of nested structures
		all, _ := fs.Glob(SeedFiles, "seeds/*/*.sql")
		for _, e := range all {
			if strings.Contains(e, norm) {
				entries = append(entries, e)
			}
		}
	}

	sort.Strings(entries)
	var applied []string

	for _, path := range entries {
		content, err := SeedFiles.ReadFile(path)
		if err != nil {
			return applied, fmt.Errorf("read seed %s: %w", path, err)
		}
		sum := fmt.Sprintf("%x", sha256.Sum256(content))

		var stored string
		err = r.db.QueryRowContext(ctx, `SELECT checksum FROM seed_checksums WHERE file = $1`, path).Scan(&stored)
		if err == nil && stored == sum {
			continue // Unchanged file skipped with single fast query
		}

		log.Printf("[SeedRunner] Applying seed: %s (hash: %s)", path, sum[:8])
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return applied, fmt.Errorf("begin tx for %s: %w", path, err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			_ = tx.Rollback()
			return applied, fmt.Errorf("execute seed %s: %w", path, err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO seed_checksums (file, checksum, run_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (file) DO UPDATE SET checksum = EXCLUDED.checksum, run_at = NOW()
		`, path, sum)
		if err != nil {
			_ = tx.Rollback()
			return applied, fmt.Errorf("record checksum %s: %w", path, err)
		}

		if err := tx.Commit(); err != nil {
			return applied, fmt.Errorf("commit seed %s: %w", path, err)
		}
		applied = append(applied, path)
	}

	return applied, nil
}

// RunAll executes seeds for all available language pairs.
func (r *SeedRunner) RunAll(ctx context.Context) ([]string, error) {
	entries, err := fs.Glob(SeedFiles, "seeds/*/*.sql")
	if err != nil {
		return nil, fmt.Errorf("glob all seeds: %w", err)
	}

	pairs := make(map[string]bool)
	for _, e := range entries {
		parts := strings.Split(e, "/")
		if len(parts) >= 2 {
			pairs[parts[1]] = true
		}
	}

	var allApplied []string
	for pair := range pairs {
		applied, err := r.Run(ctx, pair)
		if err != nil {
			return allApplied, err
		}
		allApplied = append(allApplied, applied...)
	}
	return allApplied, nil
}

// Check inspects whether any seeds would change without executing them.
func (r *SeedRunner) Check(ctx context.Context, pair string) (map[string]bool, error) {
	norm := normalizePair(pair)
	entries, err := fs.Glob(SeedFiles, fmt.Sprintf("seeds/%s/*.sql", norm))
	if err != nil {
		return nil, err
	}
	sort.Strings(entries)

	results := make(map[string]bool)
	for _, path := range entries {
		content, err := SeedFiles.ReadFile(path)
		if err != nil {
			return nil, err
		}
		sum := fmt.Sprintf("%x", sha256.Sum256(content))

		var stored string
		err = r.db.QueryRowContext(ctx, `SELECT checksum FROM seed_checksums WHERE file = $1`, path).Scan(&stored)
		if err != nil || stored != sum {
			results[path] = true // Needs apply
		} else {
			results[path] = false // Unchanged
		}
	}
	return results, nil
}

// Status returns all applied checksums currently stored in the database.
func (r *SeedRunner) Status(ctx context.Context) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT file, checksum FROM seed_checksums ORDER BY file`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	status := make(map[string]string)
	for rows.Next() {
		var f, c string
		if err := rows.Scan(&f, &c); err == nil {
			status[f] = c
		}
	}
	return status, rows.Err()
}
