package testenv

import _ "embed"

// plcServerScript is an in-memory PLC server that runs as a co-process inside
// the PDS container, so we don't need to create records for test accounts
// in https://plc.directory.
//
// It uses @did-plc/lib (already bundled in the PDS image) for full validation
// including cryptographic signature checking, but it's just a glorified array
// over HTTP.
//
//go:embed plc.js
var plcServerScript []byte

// pdsEntrypoint starts the in-memory PLC server, waits for it to be ready,
// then starts the PDS pointing at it.
const pdsEntrypoint = `
mkdir -p /pds/blocks
node /app/plc.js &
for i in $(seq 1 50); do
  wget -qO- http://127.0.0.1:2582/_health 2>/dev/null && break
  sleep 0.1
done
node --enable-source-maps index.ts
`
