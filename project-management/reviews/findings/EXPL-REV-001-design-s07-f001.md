# Review Finding: EXPL-REV-001-D-S07-F001

- Finding ID: `EXPL-REV-001-D-S07-F001`
- Reviewer identity: `/root/design_security`
- Seat and lens: Seat 7 — Security
- Stage: design
- Packet integrity identifier: `sha256:fb7897d9ca63620c69dd7bd77ecf82b113ab51a947be85ec5462be700f9042b7`
- Severity: major
- Claim: Protocol completion retention stores 256 complete snapshots rather
  than compact terminal metadata.
- Consequence: Repeated observations can retain 7,372,800 cell records plus
  control trees at maximum geometry, creating a practical memory-exhaustion
  path despite bounded wire responses.
- Evidence: `Completion` owns `*SnapshotV1`
  (`automation/protocol.go:226-236`); every completion receives a snapshot
  (`automation/server.go:513-522`); the entire completion is retained
  (`automation/server.go:562-580`) although queries use only summary fields
  (`automation/server.go:594-604`). Limits permit 256 completions, 28,800
  cells, and 4,096 controls (`automation/protocol.go:99-104`).
  `App.Snapshot` deep-copies snapshot slices (`app.go:131-136`;
  `types.go:267-288`).
- Recommendation: Retain only `RetainedCompletion` metadata and reference
  separately bounded snapshot history by sequence. Establish and test a
  worst-case aggregate memory budget.
- Related lenses: Performance; reliability; testability.
- Disposition: duplicate
- Disposition rationale: Duplicate of `EXPL-REV-001-D-S02-F004`; retaining a
  full snapshot inside each completion is the memory-bound consequence of the
  same duplicated completion/snapshot ownership.
- Resolution: pending
- Resolution evidence or operator record: pending
