package store

import (
	"cloud.google.com/go/cloudsqlconn"
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"net"
)

type DirectConnectorConfig struct {
	URI string
}

func (c *DirectConnectorConfig) poolConfig() (*pgxpool.Config, error) {
	pgxCfg, err := pgxpool.ParseConfig(c.URI)
	if err != nil {
		return nil, fmt.Errorf("parsing db url: %w", err)
	}
	return pgxCfg, nil
}

type CloudSQLConnectorConfig struct {
	Instance string
	Database string
	// TODO: Determine user from the app default service credentials
	Username string
}

func (c *CloudSQLConnectorConfig) poolConfig(ctx context.Context) (*pgxpool.Config, error) {
	d, err := cloudsqlconn.NewDialer(ctx, cloudsqlconn.WithIAMAuthN())
	if err != nil {
		return nil, fmt.Errorf("creating cloud sql dialer: %w", err)
	}
	pgxCfg, err := pgxpool.ParseConfig(fmt.Sprintf("user=%s database=%s", c.Username, c.Database))
	if err != nil {
		return nil, fmt.Errorf("parsing cloud sql config: %w", err)
	}
	pgxCfg.ConnConfig.DialFunc = func(ctx context.Context, _, _ string) (net.Conn, error) {
		return d.Dial(ctx, c.Instance)
	}
	return pgxCfg, nil
}

type QueriesWithTX struct {
	db *pgxpool.Pool
	*Queries
}

func (q *QueriesWithTX) WithTx(ctx context.Context) (queries *Queries, commit func(ctx context.Context) error, rollback func(), err error) {
	tx, err := q.db.Begin(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	return q.Queries.WithTx(tx), tx.Commit, func() {
		// For now, we just sink this error - we should log this in future.
		_ = tx.Rollback(context.Background())
	}, nil
}

type Config struct {
	CloudSQL *CloudSQLConnectorConfig
	Direct   *DirectConnectorConfig
}

func (c *Config) Connect(ctx context.Context) (*QueriesWithTX, func(), error) {
	var err error
	var pgxCfg *pgxpool.Config
	switch {
	case c.CloudSQL != nil && c.Direct != nil:
		return nil, nil, fmt.Errorf("direct and cloudsql connectors are mutually exclusive")
	case c.CloudSQL != nil:
		pgxCfg, err = c.CloudSQL.poolConfig(ctx)
	case c.Direct != nil:
		pgxCfg, err = c.Direct.poolConfig()
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to db: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgxCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create pool: %w", err)
	}
	return &QueriesWithTX{
		db:      pool,
		Queries: New(pool),
	}, pool.Close, nil
}

func ConfigFromEnv() (*Config, error) {
	return nil, fmt.Errorf("not yet implemented")
}
