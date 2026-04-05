package models

import (
	"github.com/goccy/go-json"

	_ "embed"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
)

//go:embed zstd_dictionary
var ZSTDDictionary []byte

type Event struct {
	Did      string                                  `json:"did"`
	TimeUS   int64                                   `json:"time_us"`
	Kind     string                                  `json:"kind,omitempty"`
	Commit   *Commit                                 `json:"commit,omitempty"`
	Account  *comatproto.SyncSubscribeRepos_Account  `json:"account,omitempty"`
	Identity *comatproto.SyncSubscribeRepos_Identity `json:"identity,omitempty"`
}

type Commit struct {
	Rev        string          `json:"rev,omitempty"`
	Operation  string          `json:"operation,omitempty"`
	Collection string          `json:"collection,omitempty"`
	RKey       string          `json:"rkey,omitempty"`
	Record     json.RawMessage `json:"record,omitempty"`
	CID        string          `json:"cid,omitempty"`
}

var (
	EventKindCommit   = "commit"
	EventKindAccount  = "account"
	EventKindIdentity = "identity"

	CommitOperationCreate = "create"
	CommitOperationUpdate = "update"
	CommitOperationDelete = "delete"
)
