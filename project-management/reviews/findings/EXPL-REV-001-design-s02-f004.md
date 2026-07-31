# Review Finding: EXPL-REV-001-D-S02-F004

- Finding ID: `EXPL-REV-001-D-S02-F004`
- Reviewer identity: `/root/design_orthogonality`
- Seat and lens: Seat 2 — orthogonality, special-case coupling, extension
  seams, design elegance
- Stage: design
- Packet integrity identifier:
  `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: App and automation Server independently own request identity,
  duplicate detection, completion retention, and snapshot retention under
  incompatible limits and lifetimes.
- Consequence: The two authorities can disagree: the protocol may forget an
  ID that App still rejects, a completion may reference an unavailable
  snapshot, and a sufficiently long process irreversibly exhausts App's
  lifetime request-ID budget.
- Evidence:
  - `docs/specifications/go-api-v0.md:577-581` makes IDs unique for the App
    lifetime and retains 4096 without eviction.
  - `docs/specifications/automation-protocol-v1.md:624-631` permits result
    metadata and its exact snapshot to expire independently.
  - `docs/specifications/automation-protocol-v1.md:727-759` documents a
    separate 256-entry FIFO and the resulting App/server disagreement.
  - `automation/server.go:30-59` holds server-owned request and snapshot
    ledgers.
  - `automation/server.go:562-653` separately retains completions and
    snapshots, then falls through to App history.
  - `app.go:189-198` maintains the App-owned lifetime request map.
  - `types.go:11-19` and `automation/protocol.go:87-108` define different App
    and protocol retention limits.
- Recommendation: Establish one authority and one lifecycle. Either let the
  protocol translate reusable wire IDs into session-unique internal tokens,
  or expose one bounded App ledger that the server uses for duplicate
  detection and queries. Define result/snapshot retention atomically or
  return an immutable retained result that does not depend on a second cache.
- Related lenses: automation correctness, bounded resources, lifecycle
  semantics
- Disposition: confirmed
- Disposition rationale: App and server maintain separate request,
  completion, and snapshot ledgers with the cited incompatible capacities,
  expiry rules, and process lifetimes.
- Resolution: pending
- Resolution evidence or operator record: pending
