package bluesky_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/strideynet/bsky-furry-feed/bluesky"
)

func TestParseTime(t *testing.T) {
	t.Parallel()

	t.Run("zulu", func(t *testing.T) {
		t.Parallel()

		zulu, err := bluesky.ParseTime("2026-07-02T19:50:20.130Z")
		require.NoError(t, err)
		require.Equal(t, zulu, time.Date(2026, time.July, 2, 19, 50, 20, 130000000, time.UTC))
	})

	t.Run("local", func(t *testing.T) {
		t.Parallel()

		local, err := bluesky.ParseTime("2026-07-02T21:50:20+03:00")
		require.NoError(t, err)
		require.Equal(t, local, time.Date(2026, time.July, 2, 21, 50, 20, 0, time.FixedZone("", 3*60*60)))
	})
}

func TestFormatTime(t *testing.T) {
	t.Parallel()

	time := time.Date(2026, time.July, 2, 21, 50, 20, 0, time.FixedZone("", 3*60*60))

	str := bluesky.FormatTime(time)
	require.Equal(t, "2026-07-02T18:50:20Z", str)
}
