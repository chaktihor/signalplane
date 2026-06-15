package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultPostgresSnapshotID = "default"

type PostgresOptions struct {
	URL        string
	SnapshotID string
	Timeout    time.Duration
}

type postgresStateStore struct {
	pool       *pgxpool.Pool
	snapshotID string
	timeout    time.Duration
}

func newPostgresStateStore(options PostgresOptions) (*postgresStateStore, error) {
	if options.URL == "" {
		return nil, errors.New("postgres store backend requires SIGNALPLANE_POSTGRES_URL")
	}
	timeout := options.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	snapshotID := options.SnapshotID
	if snapshotID == "" {
		snapshotID = defaultPostgresSnapshotID
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	pool, err := pgxpool.New(ctx, options.URL)
	if err != nil {
		return nil, err
	}
	store := &postgresStateStore{pool: pool, snapshotID: snapshotID, timeout: timeout}
	if err := store.ensureSchema(); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

func (store *postgresStateStore) ensureSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), store.timeout)
	defer cancel()
	_, err := store.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS runtime_snapshots (
  id TEXT PRIMARY KEY,
  payload JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`)
	return err
}

func (store *postgresStateStore) Close() {
	store.pool.Close()
}

func (store *postgresStateStore) Load() (snapshot, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), store.timeout)
	defer cancel()
	var payload []byte
	err := store.pool.QueryRow(ctx, `SELECT payload FROM runtime_snapshots WHERE id = $1`, store.snapshotID).Scan(&payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return snapshot{}, false, nil
	}
	if err != nil {
		return snapshot{}, false, err
	}
	var snap snapshot
	if err := json.Unmarshal(payload, &snap); err != nil {
		return snapshot{}, false, err
	}
	return snap, true, nil
}

func (store *postgresStateStore) Save(snap snapshot) error {
	payload, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), store.timeout)
	defer cancel()
	_, err = store.pool.Exec(ctx, `
INSERT INTO runtime_snapshots (id, payload, updated_at)
VALUES ($1, $2, now())
ON CONFLICT (id) DO UPDATE
SET payload = EXCLUDED.payload,
    updated_at = now()`, store.snapshotID, payload)
	return err
}
