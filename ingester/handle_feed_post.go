package ingester

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/jackc/pgx/v5/pgtype"
	bff "github.com/strideynet/bsky-furry-feed"
	"github.com/strideynet/bsky-furry-feed/bluesky"
	"github.com/strideynet/bsky-furry-feed/store"
	"go.uber.org/zap"
)

func hasImage(data *bsky.FeedPost) bool {
	return data.Embed != nil && data.Embed.EmbedImages != nil && len(data.Embed.EmbedImages.Images) > 0
}

func hasKeyword(data *bsky.FeedPost, keywords ...string) bool {
	text := strings.ToLower(data.Text)
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func isFursuitMedia(data *bsky.FeedPost) bool {
	return hasImage(data) && hasKeyword(data, "#fursuitfriday", "#fursuit")
}

func isArt(data *bsky.FeedPost) bool {
	return hasImage(data) && hasKeyword(data, "#art")
}

func isNSFW(data *bsky.FeedPost) bool {
	return hasKeyword(data, "#nsfw")
}

func (fi *FirehoseIngester) handleFeedPostCreate(
	ctx context.Context,
	log *zap.Logger,
	repoDID string,
	recordUri string,
	data *bsky.FeedPost,
) error {
	ctx, span := tracer.Start(ctx, "firehose_ingester.handle_feed_post_create")
	defer span.End()
	if data.Reply == nil {
		createdAt, err := bluesky.ParseTime(data.CreatedAt)
		if err != nil {
			return fmt.Errorf("parsing post time: %w", err)
		}

		// TODO: Break this out in a more extensible way
		tags := []string{}
		if isFursuitMedia(data) {
			tags = append(tags, bff.TagFursuitMedia)
		}
		if isArt(data) {
			tags = append(tags, bff.TagArt)
		}
		if isNSFW(data) {
			tags = append(tags, bff.TagNSFW)
		}

		err = fi.queries.CreateCandidatePost(
			ctx,
			store.CreateCandidatePostParams{
				URI:      recordUri,
				ActorDID: repoDID,
				CreatedAt: pgtype.Timestamptz{
					Time:  createdAt,
					Valid: true,
				},
				IndexedAt: pgtype.Timestamptz{
					Time:  time.Now(),
					Valid: true,
				},
				Tags: tags,
			},
		)
		if err != nil {
			return fmt.Errorf("creating candidate post: %w", err)
		}
	} else {
		log.Info("ignoring reply")
	}
	return nil
}

func (fi *FirehoseIngester) handleFeedPostDelete(
	ctx context.Context,
	log *zap.Logger,
	recordUri string,
) error {
	ctx, span := tracer.Start(ctx, "firehose_ingester.handle_feed_post_delete")
	defer span.End()

	if err := fi.queries.SoftDeleteCandidatePost(
		ctx, recordUri,
	); err != nil {
		return fmt.Errorf("deleting candidate post: %w", err)
	}

	return nil
}
