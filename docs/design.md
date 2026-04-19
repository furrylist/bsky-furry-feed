# Design

Across the whole service, we refer to accounts as "candidates" and
usually refer to an account's records as "candidate _type_", such as
candidate posts.

An account is uniquely identified by a single DID. The furrylist service
only reads Bluesky-related records.

## Mutual consent

Our feeds are based on consent. We only show accounts on the feed, who
followed us and were followed by after manual review. Accounts must
vaguely signal through their Bluesky profile or behavior to be aligned
with the furry community (furries, pups, therians, bronies, ...).

Some people don't want to keep following us to curate their timeline, so
we allow removals by sending us a DM on Bluesky or raising a ticket in
Discord.

## Data

For now, we store:

- Posts (only top-level, no replies or reposts)
- Likes
- Follows

In future, we may also store other information in order to improve the
feeds or our moderation. We only store information related to the
accounts that asked to be added or when moderation necessitates it (e.g.
banning spam bots proactively).

We obtain feed-relevant data solely from the Bluesky jetstream using
<https://github.com/bluesky-social/jetstream>. The admin panel can also
use data from the Bluesky AppView, API, or a team member's PDS.

All persistent data is stored in Postgres.

## Feed design

We have two types of feed algorithms:

1. Chronological (see `scoring/` for details)
2. Hot

Some feeds are also available in clean or after dark (aka. NSFW)
variants, identified by the 🧼 and 🌙 respectively.
