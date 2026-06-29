package testenv

import (
	"bytes"
	"testing"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/stretchr/testify/require"
)

// ExtractClientFromTestUser returns the xrpc client for a test user.
func ExtractClientFromTestUser(user *TestUser) *xrpc.Client {
	return user.client
}

// TestUser represents a user account on the test PDS.
type TestUser struct {
	did    string
	client *xrpc.Client
}

// DID returns the user's DID.
func (u *TestUser) DID() string {
	return u.did
}

// Client returns the authenticated xrpc client for this user.
func (u *TestUser) Client() *xrpc.Client {
	return u.client
}

// MustUploadBlob uploads data with the given MIME type to the PDS and returns
// the resulting [lexutil.LexBlob]. Required when constructing posts with embeds, since
// the PDS validates that blob refs point to previously uploaded blobs.
func (u *TestUser) MustUploadBlob(t *testing.T, mimeType string, data []byte) *lexutil.LexBlob {
	t.Helper()
	var out comatproto.RepoUploadBlob_Output
	err := u.client.LexDo(t.Context(), "POST", mimeType, "com.atproto.repo.uploadBlob", nil, bytes.NewReader(data), &out)
	require.NoError(t, err, "uploading blob")
	return out.Blob
}
