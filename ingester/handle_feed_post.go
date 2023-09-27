package ingester

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/srinathh/hashtag"
	"github.com/strideynet/bsky-furry-feed/bluesky"
	"github.com/strideynet/bsky-furry-feed/store"
	"golang.org/x/exp/maps"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// postTextWithAlts appends the alt texts of images to the text itself. This
// lets us detect hashtags within an alt text.
func postTextWithAlts(data *bsky.FeedPost) string {
	text := data.Text
	if data.Embed != nil && data.Embed.EmbedImages != nil && data.Embed.EmbedImages.Images != nil {
		for _, image := range data.Embed.EmbedImages.Images {
			if image.Alt != "" {
				text = text + "\n" + image.Alt
			}
		}
	}
	return text
}

func hasMedia(data *bsky.FeedPost) bool {
	return data.Embed != nil && data.Embed.EmbedImages != nil && len(data.Embed.EmbedImages.Images) > 0
}

func extractFacetsHashtags(facets []*bsky.RichtextFacet) []string {
	var hashtags []string
	for _, facet := range facets {
		for _, feature := range facet.Features {
			tag := feature.RichtextFacet_Tag
			if tag == nil {
				continue
			}
			hashtags = append(hashtags, tag.Tag)
		}
	}
	return hashtags
}

func extractHashtags(post *bsky.FeedPost) []string {
	var hashtags []string
	hashtags = append(hashtags, post.Tags...)
	hashtags = append(hashtags, extractFacetsHashtags(post.Facets)...)
	hashtags = append(hashtags, hashtag.ExtractHashtags(postTextWithAlts(post))...)
	return hashtags
}

func normalizeHashtags(hashtags []string, langs []string) []string {
	// Casing gets kind of wacky, so we try to compute all possible hashtag casings and store them:
	// - First, we use the default Unicode lowercasing algorithm, e.g. AEIOU -> aeiou.
	// - Then, we lowercase for all languages marked explicitly in the post, e.g. for Turkish, AEIOU -> aeıou.
	// That way, we'll catch both language-sensitive hashtags and language-insensitive hashtags.
	casers := make([]cases.Caser, len(langs))
	for i, lang := range langs {
		casers[i] = cases.Lower(language.Make(lang))
	}

	hashtagsSet := make(map[string]bool)
	for _, hashtag := range hashtags {
		hashtagsSet[strings.ToLower(hashtag)] = true
		for _, caser := range casers {
			hashtagsSet[caser.String(hashtag)] = true
		}
	}

	return maps.Keys(hashtagsSet)
}

func (fi *FirehoseIngester) handleFeedPostCreate(
	ctx context.Context,
	repoDID string,
	recordUri string,
	data *bsky.FeedPost,
) (err error) {
	ctx, span := tracer.Start(ctx, "firehose_ingester.handle_feed_post_create")
	defer func() {
		endSpan(span, err)
	}()

	if data.Reply != nil {
		span.AddEvent("ignoring post as it is a reply")
		return
	}

	createdAt, err := bluesky.ParseTime(data.CreatedAt)
	if err != nil {
		return fmt.Errorf("parsing post time: %w", err)
	}

	selfLabels := []string{}
	if data.Labels != nil && data.Labels.LabelDefs_SelfLabels != nil {
		for _, label := range data.Labels.LabelDefs_SelfLabels.Values {
			selfLabels = append(selfLabels, label.Val)
		}
	}

	err = fi.store.CreatePost(
		ctx,
		store.CreatePostOpts{
			URI:        recordUri,
			ActorDID:   repoDID,
			CreatedAt:  createdAt,
			IndexedAt:  time.Now(),
			Raw:        data,
			Hashtags:   normalizeHashtags(extractHashtags(data), data.Langs),
			HasMedia:   hasMedia(data),
			SelfLabels: selfLabels,
		},
	)
	if err != nil {
		return fmt.Errorf("creating post: %w", err)
	}

	return nil
}

func (fi *FirehoseIngester) handleFeedPostDelete(
	ctx context.Context,
	recordUri string,
) (err error) {
	ctx, span := tracer.Start(ctx, "firehose_ingester.handle_feed_post_delete")
	defer func() {
		endSpan(span, err)
	}()

	if err := fi.store.DeletePost(
		ctx, store.DeletePostOpts{URI: recordUri},
	); err != nil {
		return fmt.Errorf("deleting post: %w", err)
	}

	return nil
}
