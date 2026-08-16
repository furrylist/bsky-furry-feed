package api

import (
	"context"
	"log/slog"
	"reflect"
	"testing"
	"unsafe"

	"connectrpc.com/connect"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	v1 "github.com/strideynet/bsky-furry-feed/proto/bff/v1"
	"github.com/strideynet/bsky-furry-feed/store"
	"github.com/strideynet/bsky-furry-feed/testenv"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

type memoryActorGetter map[string]*v1.Actor

func (mag memoryActorGetter) GetActorByDID(_ context.Context, did string) (*v1.Actor, error) {
	v, ok := mag[did]
	if !ok {
		return nil, store.ErrNotFound
	}
	return proto.Clone(v).(*v1.Actor), nil
}

func setSpec(req connect.AnyRequest, spec connect.Spec) {
	specVal := reflect.ValueOf(req).
		Elem().FieldByName("spec")

	reflect.NewAt(
		specVal.Type(),
		unsafe.Pointer(specVal.UnsafeAddr())).Elem().Set(reflect.ValueOf(spec))
}

func TestAuthEngine(t *testing.T) {
	t.Parallel()

	const adminDID = "did:web:feed.test.furryli.st"
	const otherDID = "did:web:user.test.furryli.st"
	adminToken := testenv.NewTokenForDID(t, adminDID)
	otherUserToken := testenv.NewTokenForDID(t, otherDID)
	unknownUserToken := testenv.NewTokenForDID(t, "did:web:unknown.test.furryli.st")

	directory := identity.NewMockDirectory()
	directory.Insert(identity.Identity{
		DID:    syntax.DID(adminDID),
		Handle: syntax.Handle("feed.test.furryli.st"),
	})
	directory.Insert(identity.Identity{
		DID:    syntax.DID(otherDID),
		Handle: syntax.Handle("user.test.furryli.st"),
	})

	tests := []struct {
		name string

		headerKey     string
		headerValue   string
		actor         *v1.Actor
		procedureName string

		want    *authContext
		wantErr string
	}{
		{
			name:          "success",
			headerKey:     "Authorization",
			headerValue:   "Bearer " + adminToken,
			procedureName: "/bff.v1.ModerationService/CreateActor",
			actor: &v1.Actor{
				Did:   adminDID,
				Roles: []string{"admin"},
			},
			want: &authContext{
				DID: adminDID,
				Actor: &v1.Actor{
					Did:   adminDID,
					Roles: []string{"admin"},
				},
			},
		},
		{
			name:          "success: existent but unregistered user",
			headerKey:     "Authorization",
			headerValue:   "Bearer " + otherUserToken,
			procedureName: "/bff.v1.ModerationService/Ping",
			want: &authContext{
				DID: otherDID,
			},
		},
		{
			name:          "success: non-existent user",
			headerKey:     "Authorization",
			headerValue:   "Bearer " + unknownUserToken,
			procedureName: "/bff.v1.ModerationService/Ping",
			wantErr:       "unauthenticated: user does not exist: DID not found",
		},
		{
			name:          "no header",
			procedureName: "/bff.v1.ModerationService/CreateActor",
			wantErr:       "unauthenticated: no token provided",
		},
		{
			name:          "malformed header",
			headerKey:     "Authorization",
			headerValue:   "rewgwegnmwerogkmowergiopwergiopwergop",
			procedureName: "/bff.v1.ModerationService/CreateActor",
			wantErr:       "unauthenticated: malformed header",
		},
		{
			name:          "unsupported auth type",
			headerKey:     "Authorization",
			headerValue:   "OtherType foo",
			procedureName: "/bff.v1.ModerationService/CreateActor",
			wantErr:       "unauthenticated: only Bearer auth supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			mag := memoryActorGetter{}
			if tt.actor != nil {
				mag[tt.actor.Did] = tt.actor
			}
			ae := &AuthEngine{
				ActorGetter: mag,
				TokenValidator: func(ctx context.Context, token, pdsHost string) (did string, err error) {
					return token, nil
				},
				IdentityDirectory: directory,
				Log:               slog.Default(),
			}

			req := connect.NewRequest(&v1.PingRequest{})
			if tt.headerKey != "" {
				req.Header().Set(tt.headerKey, tt.headerValue)
			}
			setSpec(req, connect.Spec{Procedure: tt.procedureName})
			got, err := ae.auth(ctx, req)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Empty(t, cmp.Diff(tt.want, got, protocmp.Transform()))
		})
	}
}
