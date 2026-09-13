package ingester

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"github.com/strideynet/bsky-furry-feed/bluesky"
	bffv1pb "github.com/strideynet/bsky-furry-feed/proto/bff/v1"
	"github.com/strideynet/bsky-furry-feed/store"
)

func (fi *FirehoseIngester) handleFeedLikeCreate(
	ctx context.Context,
	repoDID string,
	recordUri string,
	data *bsky.FeedLike,
) (err error) {
	ctx, span := tracer.Start(ctx, "firehose_ingester.handle_feed_like_create")
	defer func() {
		endSpan(span, err)
	}()

	if fi.shouldSkipLike(data) {
		fi.log.Debug("skipping like", "uri", recordUri)
		return nil
	}

	createdAt, err := bluesky.ParseTime(data.CreatedAt)
	if err != nil {
		return fmt.Errorf("parsing like time: %w", err)
	}
	err = fi.store.CreateLike(ctx, store.CreateLikeOpts{
		URI:        recordUri,
		ActorDID:   repoDID,
		SubjectURI: data.Subject.Uri,
		CreatedAt:  createdAt,
		IndexedAt:  time.Now(),
	})
	if err != nil {
		return fmt.Errorf("creating like: %w", err)
	}

	return nil
}

// shouldSkipLike returns true if a like is for the post of a user, who
// isn't approved and isn't pending.
func (fi *FirehoseIngester) shouldSkipLike(like *bsky.FeedLike) bool {
	uri, err := syntax.ParseATURI(like.Subject.Uri)
	if err != nil {
		slog.Warn("invalid at:// uri", "uri", like.Subject.Uri[:min(50, len(like.Subject.Uri))], "err", err.Error())
		return true
	}

	did := uri.Authority().DID().String()
	actor := fi.actorCache.GetByDID(did)
	return actor == nil || (actor.Status != bffv1pb.ActorStatus_ACTOR_STATUS_APPROVED && actor.Status != bffv1pb.ActorStatus_ACTOR_STATUS_PENDING)
}

func (fi *FirehoseIngester) handleFeedLikeDelete(
	ctx context.Context,
	recordUri string,
) (err error) {
	ctx, span := tracer.Start(ctx, "firehose_ingester.handle_feed_like_delete")
	defer func() {
		endSpan(span, err)
	}()

	if err := fi.store.DeleteLike(
		ctx, store.DeleteLikeOpts{URI: recordUri},
	); err != nil {
		return fmt.Errorf("deleting like: %w", err)
	}

	return nil
}
