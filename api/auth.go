package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/strideynet/bsky-furry-feed/bluesky"
	"github.com/strideynet/bsky-furry-feed/internal/bfflog"
	v1 "github.com/strideynet/bsky-furry-feed/proto/bff/v1"
	"github.com/strideynet/bsky-furry-feed/store"
)

type actorGetter interface {
	GetActorByDID(ctx context.Context, did string) (*v1.Actor, error)
}

func BSkyTokenValidator(defaultPdsHost string) func(ctx context.Context, token, pdsHost string) (did string, err error) {
	// Check the presented token is valid against the real bsky.
	// This also lets us introspect information about the user - we can't just
	// parse the JWT as they do not use public key signing for the JWT.
	return func(ctx context.Context, token, pdsHost string) (did string, err error) {
		if pdsHost == "" {
			pdsHost = defaultPdsHost
		}
		ua := bluesky.UserAgent
		res, err := getBlueskySession(ctx, &xrpc.Client{
			Host:      pdsHost,
			UserAgent: &ua,
			Auth:      &xrpc.AuthInfo{AccessJwt: token},
		})
		if err != nil {
			return "", fmt.Errorf("get session: %w", err)
		}
		return res.Did, nil
	}
}

type ServerGetSession_Output struct {
	Did            string  `json:"did" cborgen:"did"`
	Email          *string `json:"email,omitempty" cborgen:"email,omitempty"`
	EmailConfirmed *bool   `json:"emailConfirmed,omitempty" cborgen:"emailConfirmed,omitempty"`
	Handle         string  `json:"handle" cborgen:"handle"`
}

// Workaround until Bluesky’s util.LexiconTypeDecoder for
// DidDoc doesn’t always error with `unrecognized type: ""`
func getBlueskySession(ctx context.Context, c *xrpc.Client) (*ServerGetSession_Output, error) {
	var out ServerGetSession_Output
	if err := c.Do(ctx, xrpc.Query, "", "com.atproto.server.getSession", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// authenticatedUserPermissions are granted to any user who is authenticated.
var authenticatedUserPermissions = []string{
	"/bff.v1.ModerationService/Ping",
	"/bff.v1.UserService/GetMe",
	"/bff.v1.UserService/JoinApprovalQueue",
}

var approverPermissions = []string{
	"/bff.v1.ModerationService/GetActor",
	"/bff.v1.ModerationService/ListActors",
	"/bff.v1.ModerationService/ListAuditEvents",
	"/bff.v1.ModerationService/ProcessApprovalQueue",
	"/bff.v1.ModerationService/CreateCommentAuditEvent",
	"/bff.v1.ModerationService/HoldBackPendingActor",
	"/bff.v1.ModerationService/ListRoles",
	"/bff.v1.ModerationService/CreateAttachmentAuditEvent",
	"/bff.v1.ModerationService/GetAttachment",
}

var moderatorPermissions = append([]string{
	"/bff.v1.ModerationService/UnapproveActor",
	"/bff.v1.ModerationService/ForceApproveActor",
}, approverPermissions...)

var adminPermissions = append([]string{
	"/bff.v1.ModerationService/BanActor",
	"/bff.v1.ModerationService/CreateActor",
	"/bff.v1.ModerationService/AssignRoles",
}, moderatorPermissions...)

var roleToPermissions = map[string][]string{
	"admin":     adminPermissions,
	"moderator": moderatorPermissions,
	"approver":  approverPermissions,
}

// AuthEngine helps authenticate requests made by users and apply authorization
// rules based on the identity found during authentication.
type AuthEngine struct {
	// ActorGetter provides a way for the AuthEngine to fetch the Actor data
	// associated with a given DID.
	ActorGetter actorGetter
	// TokenValidator validates a given token and returns the DID associated
	// with that token.
	TokenValidator func(ctx context.Context, token, pdsHost string) (did string, err error)
	// IdentityDirectory allows resolving the DID to a PDS.
	IdentityDirectory identity.Directory

	Log *slog.Logger
}

type authContext struct {
	// DID is the did extracted from the token supplied by the user.
	DID string
	// Actor is the actor fetched from the database during authz/authn. This
	// should be used carefully, and if necessary the actor should be fetched
	// again within a transaction if mutation is occurring.
	//
	// This will be nil if the actor does not exist.
	Actor *v1.Actor
}

// TODO: Allow a authOpts to be passed in with a description of attempted
// action.
func (a *AuthEngine) auth(ctx context.Context, req connect.AnyRequest) (*authContext, error) {
	// Extract the token from the headers
	token, userDID, err := extractTokenAndSubject(req.Header())
	if err != nil {
		return nil, err
	}

	endpoint, err := a.resolveDIDToPDS(ctx, userDID)
	switch {
	case errors.Is(err, identity.ErrDIDNotFound):
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user does not exist: %w", err))
	case err != nil:
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("resolving DID to PDS: %w", err))
	}

	// Validate the token from the header
	_, err = a.TokenValidator(ctx, token.Raw, endpoint)
	if err != nil {
		return nil, fmt.Errorf("validating token: %w", err)
	}

	// Try to fetch the actor to find any roles they have associated with them.
	// If they don't exist - we continue - so act with caution, actor may be
	// nil.
	actor, err := a.ActorGetter.GetActorByDID(ctx, userDID)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("fetching actor for token: %w", err)
	}

	// Default to no roles if the actor does not exist.
	var actorRoles []string
	if actor != nil {
		actorRoles = actor.Roles
	}

	// Convert the actors roles to a quasi-set of permitted RPCs
	permissions := map[string]bool{}
	for _, permission := range authenticatedUserPermissions {
		permissions[permission] = true
	}
	for _, role := range actorRoles {
		rolePerms, ok := roleToPermissions[role]
		if !ok {
			// Gracefully handle an unrecognized role
			a.Log.Warn(
				"unrecognized role",
				slog.String("role", role),
				bfflog.ActorDID(userDID),
			)
			continue
		}
		for _, permission := range rolePerms {
			permissions[permission] = true
		}
	}

	// Check user has permission for target RPC
	procedureName := req.Spec().Procedure
	if !permissions[procedureName] {
		return nil, connect.NewError(
			connect.CodePermissionDenied,
			fmt.Errorf("user (%s) does not have permissions for %q", userDID, procedureName),
		)
	}

	return &authContext{
		DID:   userDID,
		Actor: actor,
	}, nil
}

func (a *AuthEngine) resolveDIDToPDS(ctx context.Context, did string) (string, error) {
	parsedDID, err := syntax.ParseDID(did)
	if err != nil {
		return "", err
	}
	identity, err := a.IdentityDirectory.LookupDID(ctx, parsedDID)
	if err != nil {
		return "", err
	}
	return identity.PDSEndpoint(), nil
}

func extractTokenAndSubject(header http.Header) (*jwt.Token, string, error) {
	authHeader := header.Get("Authorization")
	if authHeader == "" {
		return nil, "", connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("no token provided"))
	}
	authTyp, token, ok := strings.Cut(authHeader, " ")
	if !ok {
		return nil, "", connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("malformed header"))
	}
	if authTyp != "Bearer" {
		return nil, "", connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("only Bearer auth supported"))
	}

	data, err := jwt.Parse(token, func(token *jwt.Token) (any, error) {
		return nil, nil
	}, jwt.WithoutClaimsValidation())
	if data == nil {
		return nil, "", connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("jwt parsing failed: %w", err))
	}

	subject, err := data.Claims.GetSubject()
	if err != nil {
		return nil, "", connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing DID (sub) in JWT: %w", err))
	}

	return data, subject, nil
}
