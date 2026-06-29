package testenv

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
	"github.com/strideynet/bsky-furry-feed/store"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func startPDS(ctx context.Context, t *testing.T) *TestPDS {
	t.Helper()

	const pdsPort = "3000/tcp"

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "ghcr.io/bluesky-social/pds:0.4.5009",
			ExposedPorts: []string{pdsPort},
			Env: map[string]string{
				"PDS_HOSTNAME":       "localhost",
				"PDS_JWT_SECRET":     "test-jwt-secret-not-for-production",
				"PDS_ADMIN_PASSWORD": "admin",
				"PDS_PLC_ROTATION_KEY_K256_PRIVATE_KEY_HEX": "3ee68ca7de7f9af37d9e02a21f3c49de87d4e7c2aa6c6e8a1c26e2f1e8bb8a7f",
				"PDS_DATA_DIRECTORY":                        "/pds",
				"PDS_BLOBSTORE_DISK_LOCATION":               "/pds/blocks",
				"PDS_DID_PLC_URL":                           "http://127.0.0.1:2582",
				"PDS_DEV_MODE":                              "true",
				"PDS_INVITE_REQUIRED":                       "false",
				"PDS_SERVICE_HANDLE_DOMAINS":                ".tpds",
				"NODE_ENV":                                  "production",

				// just in case we want to run against the public app view
				// "PDS_BSKY_APP_VIEW_URL": "https://api.bsky.app",
				// "PDS_BSKY_APP_VIEW_DID": "did:web:api.bsky.app",
			},
			Entrypoint: []string{"sh", "-c", pdsEntrypoint},
			Files: []testcontainers.ContainerFile{
				{
					Reader:            bytes.NewReader(plcServerScript),
					ContainerFilePath: "/app/plc.js",
					FileMode:          0o644,
				},
			},
			WaitingFor: wait.ForHTTP("/xrpc/com.atproto.server.describeServer").WithPort(nat.Port(pdsPort)),
		},
		Started: true,
	})
	require.NoError(t, err, "starting PDS container")
	t.Cleanup(func() {
		require.NoError(t, container.Terminate(context.Background()))
	})

	mappedPort, err := container.MappedPort(ctx, nat.Port(pdsPort))
	require.NoError(t, err, "getting PDS port")

	host, err := container.Host(ctx)
	require.NoError(t, err, "getting PDS host")

	return &TestPDS{
		rawHost: fmt.Sprintf("%s:%d", host, mappedPort.Int()),
	}
}

func StartDatabase(ctx context.Context, t *testing.T) (url string) {
	t.Helper()

	waitStrategy := wait.ForSQL("5432/tcp", "postgres", func(host string, port nat.Port) string {
		return fmt.Sprintf("postgres://bff:bff@%s:%d/bff?sslmode=disable", host, port.Int())
	})
	container, err := postgres.Run(ctx,
		"postgres:16.13-alpine",
		postgres.WithDatabase("bff"),
		postgres.WithUsername("bff"),
		postgres.WithPassword("bff"),
		testcontainers.WithWaitStrategy(waitStrategy),
	)
	require.NoError(t, err, "starting postgres container")
	t.Cleanup(func() {
		require.NoError(t, container.Terminate(context.Background()))
	})

	port, err := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err, "getting postgres port")

	host, err := container.Host(ctx)
	require.NoError(t, err, "getting postgres host")

	url = fmt.Sprintf("postgres://bff:bff@%s:%d/bff?sslmode=disable", host, port.Int())

	migrator, err := migrate.New("file://../store/migrations", url)
	require.NoError(t, err, "initializing migration runner")
	require.NoError(t, migrator.Up(), "applying migrations")

	return url
}

type Harness struct {
	PDS   *TestPDS
	Store *store.PGXStore
}

func StartHarness(ctx context.Context, t *testing.T) *Harness {
	dbURL := StartDatabase(ctx, t)
	pds := startPDS(ctx, t)

	pgxStore, err := store.ConnectPGXStore(
		ctx,
		slog.Default(),
		&store.DirectConnector{URI: dbURL},
	)
	require.NoError(t, err)
	t.Cleanup(pgxStore.Close)

	return &Harness{
		PDS:   pds,
		Store: pgxStore,
	}
}
