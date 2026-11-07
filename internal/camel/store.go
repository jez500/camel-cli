// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.

package camel

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Schema creates the camel-cli tables. It is idempotent.
const Schema = `
CREATE TABLE IF NOT EXISTS camel_snapshots (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	asin TEXT NOT NULL,
	locale TEXT NOT NULL,
	price_type TEXT NOT NULL,
	fetched_at TEXT NOT NULL,
	title TEXT,
	current REAL,
	current_date TEXT,
	lowest REAL,
	lowest_date TEXT,
	highest REAL,
	highest_date TEXT,
	average REAL
);
CREATE INDEX IF NOT EXISTS camel_snapshots_key ON camel_snapshots(asin, locale, price_type, fetched_at);
CREATE TABLE IF NOT EXISTS camel_watchlist (
	asin TEXT NOT NULL,
	locale TEXT NOT NULL,
	title TEXT,
	added_at TEXT NOT NULL,
	PRIMARY KEY (asin, locale)
);
CREATE TABLE IF NOT EXISTS camel_drops_seen (
	asin TEXT NOT NULL,
	locale TEXT NOT NULL,
	period TEXT NOT NULL,
	price REAL NOT NULL,
	title TEXT,
	drop_pct REAL,
	first_seen TEXT NOT NULL,
	last_seen TEXT NOT NULL,
	seen_count INTEGER NOT NULL DEFAULT 1,
	PRIMARY KEY (asin, locale, period, price)
);
`

// EnsureSchema creates the tables if needed.
func EnsureSchema(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, Schema); err != nil {
		return fmt.Errorf("creating camel tables: %w", err)
	}
	return nil
}

func nullF(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

func nullS(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// SaveSnapshot stores one row per price type that has data.
func SaveSnapshot(ctx context.Context, db *sql.DB, p *Product, at time.Time) error {
	ts := at.UTC().Format(time.RFC3339)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	for _, ps := range p.Prices {
		if !ps.HasData() {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO camel_snapshots
			(asin, locale, price_type, fetched_at, title, current, current_date, lowest, lowest_date, highest, highest_date, average)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
			p.ASIN, p.Locale, ps.Type, ts, p.Title, nullF(ps.Current), nullS(ps.CurrentDate), nullF(ps.Lowest), nullS(ps.LowestDate),
			nullF(ps.Highest), nullS(ps.HighestDate), nullF(ps.Average)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("saving snapshot: %w", err)
		}
	}
	return tx.Commit()
}

// Snapshot is one stored lookup for one price type.
type Snapshot struct {
	ASIN        string   `json:"asin"`
	Locale      string   `json:"locale"`
	Type        string   `json:"type"`
	FetchedAt   string   `json:"fetched_at"`
	Title       string   `json:"title,omitempty"`
	Current     *float64 `json:"current"`
	CurrentDate string   `json:"current_date,omitempty"`
	Lowest      *float64 `json:"lowest"`
	LowestDate  string   `json:"lowest_date,omitempty"`
	Highest     *float64 `json:"highest"`
	HighestDate string   `json:"highest_date,omitempty"`
	Average     *float64 `json:"average"`
}

func fp(n sql.NullFloat64) *float64 {
	if !n.Valid {
		return nil
	}
	v := n.Float64
	return &v
}

func scanSnapshots(rows *sql.Rows) ([]Snapshot, error) {
	out := make([]Snapshot, 0)
	for rows.Next() {
		var s Snapshot
		var title, cd, ld, hd sql.NullString
		var cur, low, high, avg sql.NullFloat64
		if err := rows.Scan(&s.ASIN, &s.Locale, &s.Type, &s.FetchedAt, &title, &cur, &cd, &low, &ld, &high, &hd, &avg); err != nil {
			_ = rows.Close()
			return nil, err
		}
		s.Title, s.CurrentDate, s.LowestDate, s.HighestDate = title.String, cd.String, ld.String, hd.String
		s.Current, s.Lowest, s.Highest, s.Average = fp(cur), fp(low), fp(high), fp(avg)
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	return out, rows.Close()
}

const snapCols = `asin, locale, price_type, fetched_at, title, current, current_date, lowest, lowest_date, highest, highest_date, average`

// Snapshots returns stored snapshots for one product, oldest first.
// An empty priceType returns all types.
func Snapshots(ctx context.Context, db *sql.DB, asin, locale, priceType string) ([]Snapshot, error) {
	q := `SELECT ` + snapCols + ` FROM camel_snapshots WHERE asin = ? AND locale = ?`
	args := []any{asin, locale}
	if priceType != "" {
		q += ` AND price_type = ?`
		args = append(args, priceType)
	}
	q += ` ORDER BY fetched_at, id`
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	return scanSnapshots(rows)
}

// LatestSnapshots returns the newest snapshot per (asin, locale, type) for
// every watched product.
func LatestSnapshots(ctx context.Context, db *sql.DB) ([]Snapshot, error) {
	rows, err := db.QueryContext(ctx, `SELECT s.asin, s.locale, s.price_type, s.fetched_at, COALESCE(s.title, w.title), s.current, s.current_date, s.lowest, s.lowest_date, s.highest, s.highest_date, s.average
		FROM camel_snapshots s
		JOIN camel_watchlist w ON w.asin = s.asin AND w.locale = s.locale
		WHERE s.id = (SELECT id FROM camel_snapshots s2 WHERE s2.asin = s.asin AND s2.locale = s.locale AND s2.price_type = s.price_type ORDER BY s2.fetched_at DESC, s2.id DESC LIMIT 1)
		ORDER BY s.asin, s.locale, s.price_type`)
	if err != nil {
		return nil, err
	}
	return scanSnapshots(rows)
}

// WatchItem is one watchlist row.
type WatchItem struct {
	ASIN    string `json:"asin"`
	Locale  string `json:"locale"`
	Title   string `json:"title,omitempty"`
	AddedAt string `json:"added_at"`
}

// AddWatch inserts or updates a watchlist row. It reports whether the row is new.
func AddWatch(ctx context.Context, db *sql.DB, w WatchItem) (bool, error) {
	res, err := db.ExecContext(ctx, `INSERT INTO camel_watchlist (asin, locale, title, added_at) VALUES (?,?,?,?)
		ON CONFLICT(asin, locale) DO NOTHING`, w.ASIN, w.Locale, nullS(w.Title), w.AddedAt)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	if n == 0 && w.Title != "" {
		_, err = db.ExecContext(ctx, `UPDATE camel_watchlist SET title = ? WHERE asin = ? AND locale = ?`, w.Title, w.ASIN, w.Locale)
	}
	return n > 0, err
}

// RemoveWatch deletes a watchlist row. It reports whether a row existed.
func RemoveWatch(ctx context.Context, db *sql.DB, asin, locale string) (bool, error) {
	res, err := db.ExecContext(ctx, `DELETE FROM camel_watchlist WHERE asin = ? AND locale = ?`, asin, locale)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// Watchlist returns all watched products.
func Watchlist(ctx context.Context, db *sql.DB) ([]WatchItem, error) {
	rows, err := db.QueryContext(ctx, `SELECT asin, locale, COALESCE(title, ''), added_at FROM camel_watchlist ORDER BY added_at, asin`)
	if err != nil {
		return nil, err
	}
	out := make([]WatchItem, 0)
	for rows.Next() {
		var w WatchItem
		if err := rows.Scan(&w.ASIN, &w.Locale, &w.Title, &w.AddedAt); err != nil {
			_ = rows.Close()
			return nil, err
		}
		out = append(out, w)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	return out, rows.Close()
}

// RecordDrops stores drops and returns, for each input drop, how many times
// it was seen before this call (0 = new).
func RecordDrops(ctx context.Context, db *sql.DB, locale, period string, drops []Drop, at time.Time) ([]int, error) {
	ts := at.UTC().Format(time.RFC3339)
	prior := make([]int, len(drops))
	// Read prior counts first (drain-first), then write.
	for i, d := range drops {
		if d.ASIN == "" || d.Price == nil {
			continue
		}
		var n sql.NullInt64
		err := db.QueryRowContext(ctx, `SELECT seen_count FROM camel_drops_seen WHERE asin=? AND locale=? AND period=? AND price=?`, d.ASIN, locale, period, *d.Price).Scan(&n)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		prior[i] = int(n.Int64)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	for _, d := range drops {
		if d.ASIN == "" || d.Price == nil {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO camel_drops_seen (asin, locale, period, price, title, drop_pct, first_seen, last_seen, seen_count)
			VALUES (?,?,?,?,?,?,?,?,1)
			ON CONFLICT(asin, locale, period, price) DO UPDATE SET last_seen = excluded.last_seen, seen_count = seen_count + 1`,
			d.ASIN, locale, period, *d.Price, nullS(d.Title), nullF(d.DropPct), ts, ts); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	return prior, tx.Commit()
}
