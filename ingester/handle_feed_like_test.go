package ingester

import (
	"context"
	"log/slog"
	"testing"
	"time"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	bffv1pb "github.com/strideynet/bsky-furry-feed/proto/bff/v1"
	"github.com/strideynet/bsky-furry-feed/store"
	"github.com/strideynet/bsky-furry-feed/testenv"
)

func countLikes(ctx context.Context, t *testing.T, pool *pgxpool.Pool, uri string) int {
	t.Helper()
	var n int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM candidate_likes WHERE uri = $1", uri).Scan(&n)
	require.NoError(t, err)
	return n
}

func TestFirehoseIngester_FeedLike(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	harness := testenv.StartHarness(ctx, t)

	pool := harness.Store.GetPool()

	approvedFurry := harness.PDS.MustNewUser(t, "approvedFurry.tpds")
	_, err := harness.Store.CreateActor(ctx, store.CreateActorOpts{
		Status: bffv1pb.ActorStatus_ACTOR_STATUS_APPROVED,
		DID:    approvedFurry.DID(),
	})
	require.NoError(t, err)

	nonFurry := harness.PDS.MustNewUser(t, "nonFurry.tpds")

	cac := NewActorCache(slog.Default(), harness.Store)
	require.NoError(t, cac.Sync(ctx))
	fi := NewFirehoseIngester(
		slog.Default(), harness.Store, cac, "ws://"+harness.PDS.RawHost(),
	)

	now := time.Now().UTC()

	postResp, err := comatproto.RepoCreateRecord(ctx, testenv.ExtractClientFromTestUser(approvedFurry), &comatproto.RepoCreateRecord_Input{
		Collection: "app.bsky.feed.post",
		Repo:       approvedFurry.DID(),
		Record: &lexutil.LexiconTypeDecoder{Val: &bsky.FeedPost{
			LexiconTypeID: "app.bsky.feed.post",
			Text:          "hewwo :3",
			CreatedAt:     now.Format(time.RFC3339),
		}},
	})
	require.NoError(t, err)

	likeURI := "at://" + approvedFurry.DID() + "/app.bsky.feed.like/selflike"

	//nolint:paralleltest // order-dependent test
	t.Run("like of approved furry post is stored", func(t *testing.T) {
		err := fi.handleFeedLikeCreate(ctx, approvedFurry.DID(), likeURI, &bsky.FeedLike{
			CreatedAt: now.Format(time.RFC3339),
			Subject: &comatproto.RepoStrongRef{
				Uri: postResp.Uri,
				Cid: postResp.Cid,
			},
		})
		require.NoError(t, err)
		require.Equal(t, 1, countLikes(ctx, t, pool, likeURI))
	})

	//nolint:paralleltest // order-dependent test
	t.Run("like of non-furry post is skipped", func(t *testing.T) {
		nonFurryLikeURI := "at://" + approvedFurry.DID() + "/app.bsky.feed.like/nonfurrylike"
		err := fi.handleFeedLikeCreate(ctx, approvedFurry.DID(), nonFurryLikeURI, &bsky.FeedLike{
			CreatedAt: now.Format(time.RFC3339),
			Subject: &comatproto.RepoStrongRef{
				Uri: "at://" + nonFurry.DID() + "/app.bsky.feed.post/somepost",
				Cid: postResp.Cid,
			},
		})
		require.NoError(t, err)
		require.Equal(t, 0, countLikes(ctx, t, pool, nonFurryLikeURI))
	})

	//nolint:paralleltest // order-dependent test
	t.Run("like delete removes the row", func(t *testing.T) {
		err := fi.handleFeedLikeDelete(ctx, likeURI)
		require.NoError(t, err)
		require.Equal(t, 0, countLikes(ctx, t, pool, likeURI))
	})
}
