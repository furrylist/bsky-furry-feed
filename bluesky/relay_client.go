package bluesky

import (
	"bytes"
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/repo"
	"github.com/bluesky-social/indigo/xrpc"
	typegen "github.com/whyrusleeping/cbor-gen"
)

const DefaultRelayHost = "https://bsky.network"

type RelayClient struct {
	RelayHost string
}

func (c *RelayClient) xrpcClient() *xrpc.Client {
	host := c.RelayHost
	if host == "" {
		host = DefaultRelayHost
	}
	return &xrpc.Client{
		Host:      host,
		UserAgent: new(UserAgent),
	}
}

// SyncGetRecord invokes the `SyncGetRecord` RPC against the Relay, and then
// parses the returned CAR to retrieve the record and the current repo rev.
func (c *RelayClient) SyncGetRecord(
	ctx context.Context, collection string, actorDID string, rkey string,
) (record typegen.CBORMarshaler, repoRev string, err error) {
	xc := c.xrpcClient()

	blocks, err := atproto.SyncGetRecord(ctx, xc, collection, actorDID, rkey)
	if err != nil {
		return nil, "", fmt.Errorf("calling SyncGetRecord: %w", err)
	}

	rr, err := repo.ReadRepoFromCar(ctx, bytes.NewReader(blocks))
	if err != nil {
		return nil, "", fmt.Errorf("reading repo from car: %w", err)
	}

	_, record, err = rr.GetRecord(ctx, collection+"/"+rkey)
	if err != nil {
		return nil, "", fmt.Errorf("getting record: %w", err)
	}

	return record, rr.SignedCommit().Rev, nil
}
