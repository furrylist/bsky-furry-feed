package ingester

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/strideynet/bsky-furry-feed/store/gen"
)

func (fi *FirehoseIngester) handleActorVisibilityDeclaration(
	ctx context.Context,
	repoDID string,
	data *bsky.ActorContentVisibilityDeclaration,
) (err error) {
	ctx, span := tracer.Start(ctx, "firehose_ingester.handle_actor_visibility_declaration")
	defer func() {
		endSpan(span, err)
	}()

	err = fi.store.SetActorRecommendationConsent(ctx, gen.SetActorRecommendationConsentParams{
		DID:                               repoDID,
		RefusesAlgorithmicRecommendations: data.HideFromAlgorithmicRecommendations,
	})
	if err != nil {
		return fmt.Errorf("updating actor visibility declaration: %w", err)
	}

	return nil
}
