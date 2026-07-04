package tristate_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/strideynet/bsky-furry-feed/internal/tristate"
)

func TestTristate(t *testing.T) {
	t.Parallel()

	require.Equal(t, true, *tristate.True)
	require.Equal(t, false, *tristate.False)
	require.Nil(t, tristate.Maybe)
}
