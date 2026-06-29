package testenv

import (
	"net/url"
	"strings"
	"testing"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/stretchr/testify/require"
)

// TestPDS represents a running PDS container for testing.
type TestPDS struct {
	rawHost string
}

// RawHost returns a host:port string for the PDS.
func (p *TestPDS) RawHost() string {
	return p.rawHost
}

// HTTPHost returns the http:// URL for the PDS.
func (p *TestPDS) HTTPHost() string {
	u := url.URL{Scheme: "http", Host: p.rawHost}
	return u.String()
}

// MustNewUser creates a new account on the PDS and returns a TestUser.
// Handles must end in ".tpds".
func (p *TestPDS) MustNewUser(t *testing.T, handle string) *TestUser {
	t.Helper()

	require.Truef(t, strings.HasSuffix(handle, ".tpds"), "handle %s must end with .tpds", handle)

	ctx := t.Context()

	c := &xrpc.Client{Host: p.HTTPHost()}
	email := handle + "@test.invalid"
	pass := "password"
	out, err := comatproto.ServerCreateAccount(ctx, c, &comatproto.ServerCreateAccount_Input{
		Email:    &email,
		Handle:   handle,
		Password: &pass,
	})
	require.NoError(t, err, "creating test user %q", handle)

	c.Auth = &xrpc.AuthInfo{
		AccessJwt:  out.AccessJwt,
		RefreshJwt: out.RefreshJwt,
		Handle:     out.Handle,
		Did:        out.Did,
	}

	return &TestUser{
		did:    out.Did,
		client: c,
	}
}
