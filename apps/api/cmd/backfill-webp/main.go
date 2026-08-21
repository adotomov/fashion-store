// Command backfill-webp is a one-off maintenance tool that re-encodes every
// image already stored in object storage to WebP, shrinking the multi-megabyte
// JPEG/PNG originals uploaded before the on-upload optimizer existed.
//
// For each image column across every table it: downloads the object, transcodes
// it to WebP, overwrites the SAME object key with the WebP bytes (so no object
// key or foreign reference has to change), and updates the row's content_type
// and size_bytes. It is idempotent — rows already stored as image/webp, and any
// object that can't be decoded, are skipped — so it is safe to re-run.
//
// Usage (run against the target environment's DB + storage config):
//
//	go run ./cmd/backfill-webp            # convert in place
//	go run ./cmd/backfill-webp -dry-run   # report what would change, write nothing
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/app"
	"github.com/adotomov/fashion-store/apps/api/internal/platform/storage"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/imageopt"
)

// imageColumns names one image slot: the table and its bucket / object-key /
// content-type / size columns. All values are compile-time constants below, so
// interpolating them into SQL is safe (no user input).
type imageColumns struct {
	table  string
	bucket string
	key    string
	ctype  string
	size   string
}

var targets = []imageColumns{
	{"store_settings", "logo_bucket", "logo_object_key", "logo_content_type", "logo_size_bytes"},
	{"store_settings", "about_cover_bucket", "about_cover_object_key", "about_cover_content_type", "about_cover_size_bytes"},
	{"store_settings", "store_image_bucket", "store_image_object_key", "store_image_content_type", "store_image_size_bytes"},
	{"hero_settings", "background_image_bucket", "background_image_object_key", "background_image_content_type", "background_image_size_bytes"},
	{"editorial_banner_settings", "image_bucket", "image_object_key", "image_content_type", "image_size_bytes"},
	{"categories", "thumbnail_bucket", "thumbnail_object_key", "thumbnail_content_type", "thumbnail_size_bytes"},
	{"product_media", "bucket", "object_key", "content_type", "size_bytes"},
}

type tally struct{ converted, skipped, failed int }

func main() {
	dryRun := flag.Bool("dry-run", false, "report what would change without writing anything")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	bootstrapped, err := app.Bootstrap(ctx)
	if err != nil {
		slog.Error("bootstrap failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer bootstrapped.DB.Close()

	log := bootstrapped.Logger
	store := storage.NewClient(
		bootstrapped.Config.Storage.Endpoint,
		bootstrapped.Config.Storage.InsecureSkipTLS,
		bootstrapped.Config.Storage.ProjectID,
	)

	if *dryRun {
		log.Info("backfill-webp starting in DRY-RUN mode — no writes will be made")
	} else {
		log.Info("backfill-webp starting")
	}

	total := tally{}
	for _, tgt := range targets {
		t := processTarget(ctx, log, bootstrapped.DB, store, tgt, *dryRun)
		total.converted += t.converted
		total.skipped += t.skipped
		total.failed += t.failed
	}

	log.Info("backfill-webp finished",
		slog.Int("converted", total.converted),
		slog.Int("skipped", total.skipped),
		slog.Int("failed", total.failed),
	)
	if total.failed > 0 {
		os.Exit(1)
	}
}

// imageRow is one stored object to consider converting.
type imageRow struct {
	bucket string
	key    string
	ctype  string
}

func processTarget(ctx context.Context, log *slog.Logger, db *pgxpool.Pool, store *storage.Client, tgt imageColumns, dryRun bool) tally {
	rows, err := collectRows(ctx, db, tgt)
	if err != nil {
		// A not-yet-migrated environment won't have every image column. Treat a
		// missing column/table as a skip, not a failure, so the tool stays safe
		// to run before or after this change set's migrations land.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && (pgErr.Code == "42703" || pgErr.Code == "42P01") {
			log.Warn("skipping missing column/table (not migrated yet)",
				slog.String("table", tgt.table), slog.String("column", tgt.key))
			return tally{}
		}
		log.Error("list images failed", slog.String("table", tgt.table), slog.String("column", tgt.key), slog.Any("error", err))
		return tally{failed: 1}
	}

	t := tally{}
	for _, row := range rows {
		converted, err := convertOne(ctx, db, store, tgt, row, dryRun)
		switch {
		case err != nil:
			t.failed++
			log.Error("convert failed",
				slog.String("table", tgt.table), slog.String("key", row.key), slog.Any("error", err))
		case converted:
			t.converted++
			log.Info("converted", slog.String("table", tgt.table), slog.String("key", row.key))
		default:
			t.skipped++
		}
	}
	if len(rows) > 0 {
		log.Info("table done",
			slog.String("table", tgt.table), slog.String("column", tgt.key),
			slog.Int("converted", t.converted), slog.Int("skipped", t.skipped), slog.Int("failed", t.failed))
	}
	return t
}

func collectRows(ctx context.Context, db *pgxpool.Pool, tgt imageColumns) ([]imageRow, error) {
	// Column and table names come only from the constant `targets` slice.
	query := fmt.Sprintf(
		`SELECT %s, %s, COALESCE(%s, '') FROM %s WHERE %s IS NOT NULL AND %s <> ''`,
		tgt.bucket, tgt.key, tgt.ctype, tgt.table, tgt.key, tgt.key,
	)
	pgRows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer pgRows.Close()

	var out []imageRow
	for pgRows.Next() {
		var r imageRow
		if err := pgRows.Scan(&r.bucket, &r.key, &r.ctype); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, pgRows.Err()
}

// convertOne downloads, transcodes, re-uploads and records one object. It
// returns (false, nil) when the object was skipped (already WebP or not a
// decodable raster image).
func convertOne(ctx context.Context, db *pgxpool.Pool, store *storage.Client, tgt imageColumns, row imageRow, dryRun bool) (bool, error) {
	reader, _, err := store.Open(ctx, row.bucket, row.key)
	if err != nil {
		return false, fmt.Errorf("open object: %w", err)
	}
	data, err := io.ReadAll(reader)
	reader.Close()
	if err != nil {
		return false, fmt.Errorf("read object: %w", err)
	}

	webpData, ok := imageopt.ConvertToWebP(data, row.ctype)
	if !ok {
		return false, nil // already WebP, or not a decodable image
	}

	if dryRun {
		return true, nil
	}

	// Overwrite the same object key with the WebP bytes and content type; the
	// object name keeping its old extension is cosmetic — serving is driven by
	// the stored content type, not the key.
	size, err := store.Upload(ctx, row.bucket, row.key, "image/webp", bytes.NewReader(webpData))
	if err != nil {
		return false, fmt.Errorf("upload webp: %w", err)
	}

	update := fmt.Sprintf(`UPDATE %s SET %s = 'image/webp', %s = $1 WHERE %s = $2`,
		tgt.table, tgt.ctype, tgt.size, tgt.key)
	if _, err := db.Exec(ctx, update, size, row.key); err != nil {
		return false, fmt.Errorf("update row: %w", err)
	}
	return true, nil
}
