package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrNotFound indicates that no resource was found during a store call.
	ErrNotFound = fmt.Errorf("not found")
)

type DirectConnector struct {
	URI string
}

func (c *DirectConnector) poolConfig(ctx context.Context) (*pgxpool.Config, error) {
	pgxCfg, err := pgxpool.ParseConfig(c.URI)
	if err != nil {
		return nil, fmt.Errorf("parsing db url: %w", err)
	}
	return pgxCfg, nil
}
