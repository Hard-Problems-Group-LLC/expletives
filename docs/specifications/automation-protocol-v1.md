# Attached Automation Protocol Version 1

- Status: Directed, implemented contract
- Protocol name: `expletives.automation`
- Protocol version: `1`
- Snapshot version: `1`
- Last implementation audit: 2026-07-31

Related specifications:

- [`go-api-v0.md`](go-api-v0.md);
- [`implementation-baseline-v0.md`](implementation-baseline-v0.md);
- [`expletives-test.md`](expletives-test.md);
- [`concurrency-and-thread-safety.md`](concurrency-and-thread-safety.md); and
- [`../Limited-Unicode-Support.md`](../Limited-Unicode-Support.md).

## Purpose And Scope

This document is the wire and client contract for the first attached
drive-and-observe implementation in:

- the public Go package
  `github.com/Hard-Problems-Group-LLC/expletives/automation`;
- the explicitly enabled automation server in `expletives-test`; and
- the `expletivesctl` reference client.

Version 1 lets one trusted local controller observe immutable semantic
snapshots, inject logical key lifecycle events, invoke semantic commands,
query retained results, and request orderly shutdown. It is intentionally
small and sequential. It is not a generic RPC facility, a terminal byte
injection interface, or a secure remote-control protocol.

The `automation` package owns every version 1 wire type, including
`automation.SnapshotV1`. The server explicitly projects a root-package
`expletives.Snapshot` into that DTO. The temporary root
`expletives.SnapshotV1` source-compatibility alias does not alias the
automation type or make the root representation the wire schema.

The key words **MUST**, **MUST NOT**, **SHOULD**, and **MAY** state requirements
for interoperating version 1 implementations and callers. Where this contract
describes an implementation limit rather than a general design preference,
that limit is part of version 1.

## Trust And Security Boundary

Attached automation is disabled by default. `expletives-test` constructs no
automation server or socket unless the operator supplies an explicit,
per-process option:

```text
expletives-test --automation /absolute/path/to/expletives.sock
```

When the option is present, `expletives-test` writes a conspicuous warning to
standard error:

```text
WARNING: unauthenticated automation is active at PATH; any connector can observe and drive this process.
```

The fixture also identifies the active mode in its intended frame with a
default-visible StatusBar segment whose exact text is:

```text
UNAUTHENTICATED AUTOMATION ENABLED
```

The checked File/Automation Notice menu command may hide or restore this
visual segment without changing the endpoint's running state. The warning
must not consume ordinary content space or appear in a dedicated content
Panel. Standard error and the server's `hello` record remain authoritative
for the exact endpoint path; the `hello` record sets
`"unauthenticated": true`.

Version 1 performs no peer authentication, peer-credential check, capability
authorization, elevated-process refusal, or per-command authorization at the
transport boundary. The random session ID is correlation data, not a
credential. Socket mode `0600` is defense in depth and is not a portable
authentication guarantee for pathname-based Unix sockets.

Consequently:

- the operator **MUST** use version 1 only in a trusted, non-risky,
  operator-controlled context;
- the operator **MUST** choose a private parent directory and socket path that
  other principals cannot replace or traverse unexpectedly;
- version 1 **MUST NOT** be represented as safe for hostile multi-user,
  elevated, or security-sensitive use; and
- visible frame and semantic data **MUST** be treated as potentially
  sensitive.

Authentication, capability authorization, hardened multi-user operation,
elevated-process policy, and remote transports require a separate approved
proposal. They are not silently added to version 1.

## Endpoint Creation And Lifetime

The transport is a Unix stream socket.

`NewServer` requires:

- a non-nil `*expletives.App`;
- a nonempty absolute socket path;
- a socket path no longer than 107 bytes;
- an existing immediate parent that `Lstat` identifies as a directory rather
  than a symbolic link; and
- no existing filesystem object at the requested path.

The server refuses a collision. It never removes a pre-existing file, socket,
or link to make room. After binding, it disables automatic unlink-on-close,
verifies that the created object is a Unix socket, changes its mode to
`0600`, and records its filesystem identity.

`Server.Close`:

1. marks the server stopping and closes the listener;
2. closes an idle active controller, but preserves the connection while an
   already accepted request is in its execute-and-complete window; and
3. removes the endpoint only if its current object is still a Unix socket
   with the recorded identity.

An absent endpoint is already clean. If another object replaced the endpoint,
cleanup fails closed and leaves the replacement untouched. The caller owns
`Server.Close` even if `Serve` was never called. `Serve` may be called only
once and also closes the server when its context ends.

`Close` initiates shutdown but does not itself join a preserved controller
worker. `Serve` waits for every controller worker before it returns. A caller
that runs `Serve` in another goroutine **MUST** wait for that call to return
before destroying the App or allowing the hosting process to exit.

The implementation checks the immediate parent and endpoint identity. It does
not establish the ownership or mode of every path component on the caller's
behalf; choosing a protected location remains part of the explicit operator
trust decision.

### Post-Acceptance Teardown

After reserving a request, the controller worker marks the connection
processing before it attempts to write `accepted`. Server teardown then
preserves that controller connection while the worker:

1. executes the request with its normal cancellation context;
2. retains the terminal completion;
3. attempts the completion write with the advertised five-second write
   deadline; and
4. clears the processing marker and closes the connection without reading
   another request.

The processing transition and the stopping check share the server mutex.
Exactly one side wins: either stopping is already set and the worker writes no
`accepted` record, or processing is already set when `Close` decides whether
to close the controller.

Stopping still closes the listener and removes the verified socket path
immediately, so no new controller can join during the drain. Cancellation can
change the accepted operation's result to a truthful failed or cancelled
completion; it does not convert that result into success.

This ordering removes an accepted-response-before-drain-protection interval:
if the controller can observe `accepted`, the processing marker was already
set. A `Close` that wins before that marker may close the connection, but no
`accepted` response could yet have been written. The request remains
indeterminate to a client that submitted it.

The drain protects the already-processing completion from server-initiated
connection closure. It cannot guarantee delivery after a peer disconnect,
process termination, transport failure, or write timeout. A client that does
not receive the completion still follows the indeterminate-result rules
below.

## Connection Ownership

One server process accepts at most one active controller connection.

On the accepted controller connection, the server's first record is `hello`.
A concurrent second connection instead receives:

```json
{
  "protocol": "expletives.automation",
  "version": 1,
  "type": "protocol_error",
  "code": "controller_busy",
  "message": "protocol version 1 permits one controller",
  "retryable": true
}
```

The server then closes the second connection. After the active controller
disconnects, another controller may connect to the same server session.
Request-completion retention survives that reconnect.

Held-key state belongs to the server's automation input source. The server
clears that source when the controller disconnects, so a disconnected
controller cannot leave a modifier held for its successor.

## Framing And Common Envelope

Each record is one UTF-8 JSON value followed by one line-feed byte (`LF`,
`0x0A`). Embedded record delimiters and arbitrary terminal byte streams are
not supported. Size limits exclude the terminating line feed.

Every record has this exact envelope:

```json
{
  "protocol": "expletives.automation",
  "version": 1,
  "type": "record_type"
}
```

For version 1:

- `protocol` **MUST** equal `expletives.automation`;
- `version` **MUST** be the JSON number `1`;
- `type` **MUST** be one of the record types defined below;
- operation requests and operational responses **MUST NOT** contain unknown
  fields;
- `hello` is the sole additive capability preface: a version 1 client ignores
  unknown `hello` fields after validating the required fields;
- objects at any nesting level **MUST NOT** repeat a field name;
- one line **MUST** contain exactly one JSON value;
- JSON nesting depth **MUST NOT** exceed the advertised limit; and
- required values **MUST** have the documented JSON type. `null` is not a
  substitute for an omitted optional field.

Malformed request records that can still be framed receive a
`protocol_error`, and the controller loop normally remains available for the
next record. An overlong or non-newline-terminated record receives a framing
error when possible and closes the connection. A blank line, idle read
timeout, transport error, or unframeable input closes the connection.

## Identifiers

Every operation request has a `request_id`. Request IDs and the other bounded
wire identifiers use this ASCII grammar:

```text
[A-Za-z0-9._:-]+
```

They are measured in bytes, are nonempty, and may contain at most the
advertised number of bytes. The same grammar applies to command IDs,
nonempty `target_key` values, application IDs, and logical internal source
IDs. An optional target key is represented by omitting `target_key` or by
`""`, never by `null`.

`automation.NewRequestID` returns 16 random bytes encoded as 32 lowercase
hexadecimal characters. Callers may choose their own IDs when they obey the
grammar and uniqueness rules below.

## Initial `hello` And Fixed Limits

The server writes exactly one `hello` before reading requests:

```json
{
  "protocol": "expletives.automation",
  "version": 1,
  "type": "hello",
  "application": "expletives-test",
  "session_id": "32-lowercase-hex-characters",
  "unauthenticated": true,
  "supported_versions": [1],
  "scenario": "toolkit.catalog",
  "latest_frame_sequence": 1,
  "final": false,
  "operations": [
    "observe",
    "wait_snapshot",
    "inject_input",
    "invoke_command",
    "query_result",
    "reset_input",
    "shutdown"
  ],
  "commands": [
    "app.interrupt",
    "app.quit",
    "catalog.controls.collections",
    "catalog.controls.headers_footers",
    "catalog.controls.input",
    "catalog.controls.navigation",
    "catalog.controls.progress",
    "catalog.controls.scrolling",
    "catalog.controls.selection",
    "catalog.controls.status",
    "catalog.dialogs.confirm",
    "catalog.dialogs.input",
    "catalog.dialogs.message",
    "catalog.dialogs.progress",
    "catalog.layouts.absolute",
    "catalog.menus.context",
    "catalog.menus.panel",
    "catalog.panels.scrollbars",
    "fixture.toggle",
    "fixture.unavailable",
    "layout.layer.lower",
    "layout.layer.raise",
    "layout.panel.lower",
    "layout.panel.raise",
    "overflow.dismiss",
    "scenario.reset",
    "view.about",
    "view.actions",
    "view.home",
    "view.layouts.box",
    "view.layouts.grid",
    "view.menus",
    "view.panels.core",
    "view.panels.styles",
    "view.text"
  ],
  "limits": {
    "request_line_bytes": 65536,
    "response_line_bytes": 41943040,
    "json_depth": 16,
    "request_id_bytes": 64,
    "identifier_bytes": 64,
    "clients": 1,
    "outstanding_per_connection": 1,
    "retained_results": 3,
    "frame_width": 4194304,
    "frame_height": 4194304,
    "frame_cells": 4194304,
    "frame_runs": 16384,
    "controls": 4096,
    "layouts": 1024,
    "layout_items": 4096,
    "held_keys": 8,
    "read_timeout_ms": 30000,
    "write_timeout_ms": 5000
  }
}
```

The numeric values shown are the fixed version 1 limits. A version 1 server
accepts the all-zero `ServerOptions.Limits` value as a request for these
defaults; it rejects any non-default limit set. A version 1 client validates
that every advertised limit is positive, no larger than its version 1
maximum, and exactly one client and one outstanding request are selected.
It also requires every documented `hello` field to be present, non-null, and
of the documented JSON type; requires `unauthenticated` to be true; validates
the application, session, and scenario identifiers; requires a positive
latest frame sequence; requires version 1 in the supported-version inventory;
requires the complete unique version 1 operation set; and validates a bounded,
unique command inventory.

The fields mean:

| Field | Contract |
| --- | --- |
| `application` | Required nonempty bounded host-selected application identity. The reusable server supplies no fixture default. |
| `session_id` | Random per-server correlation identity. It is not secret and is not authentication. |
| `unauthenticated` | Always `true` in version 1. |
| `supported_versions` | Nonempty unique bounded list of versions supported by the server; it includes version 1 on this connection. |
| `scenario` | Current stable fixture/application scenario identity. |
| `latest_frame_sequence` | Current monotonically increasing App snapshot sequence at connection time. |
| `final` | Whether that current snapshot is final. |
| `operations` | Exactly the complete unique version 1 operation set; array order is not semantic. |
| `commands` | Sorted App registry entries whose `Automation` flag is true; at most `MaxControls` (4,096) bounded IDs. `expletives-test` advertises the example entries shown. |
| `request_line_bytes` | Maximum request JSON bytes before `LF`. |
| `response_line_bytes` | Maximum response JSON bytes before `LF`. |
| `json_depth` | Maximum object/array nesting depth. |
| `request_id_bytes` | Maximum request-ID bytes. |
| `identifier_bytes` | Maximum bytes for other protocol identifiers. |
| `clients` | Maximum simultaneous controller connections. |
| `outstanding_per_connection` | Maximum requests being executed for one connection. |
| `retained_results` | Number of completed protocol results retained for duplicate detection and `query_result`; metadata and its exact immutable snapshot enter and leave this one FIFO together. |
| `frame_width`, `frame_height`, `frame_cells` | Maximum snapshot geometry and aggregate expanded row-major cell count accepted by the reference client. Aggregate cells, not a conventional terminal axis, govern. |
| `frame_runs` | Maximum runs in the compact row-major frame carried on the wire. |
| `controls` | Maximum controls, aggregate child references, input-source records, and overflow records in one snapshot. |
| `layouts` | Maximum flat Layout records in one snapshot. |
| `layout_items` | Maximum aggregate Panel/Layout item references across Layout records. |
| `held_keys` | Maximum held logical keys in one input source. |
| `read_timeout_ms` | Idle record-read limit and default request execution/wait context deadline. App dispatch returns at this bound even if a router retains one bounded callback slot. |
| `write_timeout_ms` | Response-write limit. |

`ServerOptions.Application` is required. The command inventory is derived
from the host App's command registry and may differ on a later connection if
the host changes that registry. It is capability information, not transport
authorization: enabled state and command policy can still reject an
advertised command.

## Request Lifecycle

A well-formed, nonduplicate request produces two records in order:

1. `accepted`, confirming registration; then
2. one terminal `completion`.

`accepted` is not evidence that the operation succeeded:

```json
{
  "protocol": "expletives.automation",
  "version": 1,
  "type": "accepted",
  "request_id": "toggle-1",
  "operation": "inject_input"
}
```

An invalid or duplicate request receives one `protocol_error` instead and was
not accepted through this protocol lifecycle.

Before writing `accepted`, the server marks that controller as processing.
While that marker is set, server-initiated teardown follows the
post-acceptance drain above instead of closing the controller underneath the
accepted or completion write.

The terminal record has this shape:

```json
{
  "protocol": "expletives.automation",
  "version": 1,
  "type": "completion",
  "request_id": "toggle-1",
  "operation": "inject_input",
  "outcome": "applied",
  "frame_sequence": 4,
  "snapshot": {
    "version": 1,
    "sequence": 4,
    "final": false,
    "frame": {},
    "cursor": {},
    "controls": [],
    "overflows": []
  }
}
```

The abbreviated `snapshot` above is illustrative. Its actual fields and
bounded typed contents are those of the automation-owned
`automation.SnapshotV1` DTO. The server constructs that DTO through an
explicit field-by-field projection from the current root-package local
snapshot.

Every completion:

- repeats the accepted `request_id` and operation;
- has exactly one recognized outcome;
- has a nonzero `frame_sequence`;
- includes a complete immutable `automation.SnapshotV1`;
- has `snapshot.sequence == frame_sequence`;
- optionally includes a bounded structured `error`; and
- includes `result` only for `query_result`.

The outer protocol completion pairs the operation with the exact immutable
snapshot named by `frame_sequence`. For input, direct-command, reset, and
shutdown operations that reach correlated App dispatch, the App also
publishes the transition's request association in `snapshot.completion`.
Boundary rejection or dispatch failure carries current state without
manufacturing such an association. Observation and query likewise do not
publish App state merely to create a nested association; their outer
completion still carries an exact atomic snapshot.

An unrelated redraw does not substitute for the correlated completion of
injected input or a direct command. `wait_snapshot` is the explicit exception:
it intentionally completes on any newer App snapshot. A no-op completes
explicitly. An exit-producing command publishes its final snapshot before its
protocol completion is sent.

### Outcomes

| Outcome | Meaning |
| --- | --- |
| `applied` | The requested transition or observation was applied successfully. |
| `no_op` | The request completed successfully without the requested state changing. |
| `rejected` | Policy or current state declined the request. |
| `cancelled` | Application command policy reports cancellation. |
| `interrupted` | Application policy completed through its distinct interrupt path; the associated App snapshot is final. |
| `exited` | Application policy completed through orderly exit; the associated App snapshot is final. |
| `failed` | Dispatch, waiting, association, or application execution failed. |

The outcome is authoritative. The optional `error` supplies bounded public
detail. Its shape is:

```json
{
  "code": "command_disabled",
  "message": "command is disabled",
  "retryable": false
}
```

`retryable` never authorizes replay with the same request ID. A new attempt
uses a new ID after the caller has established that doing so is safe.

## Operations

All request examples omit no required common-envelope field.

### `observe`

Request the latest snapshot:

```json
{
  "protocol": "expletives.automation",
  "version": 1,
  "type": "observe",
  "request_id": "observe-latest"
}
```

Request an exact retained snapshot:

```json
{
  "protocol": "expletives.automation",
  "version": 1,
  "type": "observe",
  "request_id": "observe-17",
  "frame_sequence": 17
}
```

If present, `frame_sequence` must be a positive integer. Zero and `null` are
invalid. Omission means latest.

A latest or retained exact result completes `applied`. If the requested
sequence is unavailable, the request completes `rejected` with
`snapshot_not_retained` and carries the current snapshot. Retention is
bounded; clients must not assume that every sequence between two known
sequences remains available.

### `wait_snapshot`

```json
{
  "protocol": "expletives.automation",
  "version": 1,
  "type": "wait_snapshot",
  "request_id": "wait-after-17",
  "after_sequence": 17,
  "timeout_ms": 5000
}
```

`after_sequence` is required, non-null, and may be zero. The operation waits
for an App snapshot whose sequence is strictly greater than that value.

`timeout_ms` is optional. Omitted or zero selects the 30,000 ms server
execution bound. A supplied value must be from 1 through 30,000. On success,
the operation completes `applied` with the newer snapshot. A server-side
deadline or cancellation completes `failed` with `wait_timeout`,
`retryable: true`, and the then-current snapshot. Another App wait failure,
including waiting past an already final current sequence, completes `failed`
with `wait_failed`.

The Go client's `WaitSnapshot` derives `timeout_ms` from its context deadline,
rounding a positive remainder up to milliseconds and capping it at the
server's read bound. A caller-side context or read deadline may make the
client result indeterminate before a server timeout completion is received.

### `inject_input`

```json
{
  "protocol": "expletives.automation",
  "version": 1,
  "type": "inject_input",
  "request_id": "control-down",
  "event": {
    "kind": "key_down",
    "key": "control"
  }
}
```

`event` has exactly two fields:

- `kind`: `key_down`, `key_up`, or `key_press`; and
- `key`: one supported lower-case logical key identity.

The complete version 1 key set is:

```text
a through z
0 through 9
[ ]
control alt shift meta
space enter escape tab backspace
up down left right
home end page_up page_down insert delete
f1 through f12
```

These are device-independent logical key events before chord and command
resolution. They are not terminal escape bytes, text input, paste, signals,
or backend numeric key codes.

Held state is source-local:

- a first `key_down` adds the key and normally completes `applied`;
- another `key_down` for an already held key completes `no_op`;
- a ninth distinct held key completes `rejected`;
- the first held key for a new source completes `rejected` if the App's
  bounded held-source inventory is full;
- `key_up` removes a held key and completes `applied`;
- `key_up` for a key not held completes `no_op`; and
- `key_press` is one-shot and never adds or removes held state.

`key_press` resolves a chord from the pressed non-modifier key and the
currently held `control`, `alt`, `shift`, and `meta` modifiers. An unbound
press, including a one-shot modifier press, completes `no_op`. A bound press
reports the actual outcome returned by the same App command router used for
direct commands.

For example, the fixture's Control-R chord is three independently correlated
operations in this order:

```text
key_down:control
key_press:r
key_up:control
```

The middle completion's associated snapshot demonstrates the scene
transition. The final release remains required even though `key_press` itself
does not alter held state.

### `invoke_command`

```json
{
  "protocol": "expletives.automation",
  "version": 1,
  "type": "invoke_command",
  "request_id": "reset-1",
  "command": "scenario.reset",
  "target_key": "root"
}
```

`command` is a required bounded identifier. `target_key` is an optional stable
automation key and may instead be omitted or `""`; `null` is invalid. The
server resolves it to the active control's runtime ID immediately before App
invocation. A missing key, or destruction racing the lookup, completes
`rejected` with `target_not_found` and the then-current snapshot.

The command enters the App's normal bounded dispatch and command-registry
policy. A command need not be in `hello.commands` to be syntactically valid,
but the structured App router rejects unregistered or disabled commands.

`expletives-test` advertises its bounded catalog command inventory, including
fixture toggle/reset, Panel and Layout stacking, Core/Text/Actions screen
selection, interrupt, and quit commands. Disabled demonstration and future
screen commands remain advertised so clients can inspect and truthfully
observe `command_disabled`.

### `query_result`

```json
{
  "protocol": "expletives.automation",
  "version": 1,
  "type": "query_result",
  "request_id": "query-toggle-1",
  "target_request_id": "toggle-1"
}
```

The query itself completes `applied`. For a completed target, the top-level
query completion carries the target's exact retained snapshot and uses that
snapshot's sequence as its own `frame_sequence`. Its operation-specific result
is:

```json
{
  "result": {
    "query": {
      "target_request_id": "toggle-1",
      "status": "completed",
      "completion": {
        "request_id": "toggle-1",
        "operation": "inject_input",
        "outcome": "applied",
        "frame_sequence": 4
      }
    }
  }
}
```

`status` is exactly one of:

- `pending`: the server still has an active target record;
- `completed`: the bounded retained terminal metadata is present; or
- `unknown_or_expired`: the target was never observed by this server or its
  retention entry has expired.

Only `completed` includes `completion`. The retained completion contains
request ID, operation, outcome, frame sequence, and optional error. It omits
the nested target snapshot and result, preventing recursive query-result
growth; the target's exact immutable snapshot is already carried once at the
query completion's top level.

For `pending` or `unknown_or_expired`, no exact target evidence exists and the
query completion instead carries current App state.

A query requires its own fresh request ID and itself consumes one retained
result entry. Version 1 defines no `cancel` operation: with one sequential
request, a later same-connection record cannot preempt current work. Closing
the Go client interrupts its own wait, but submitted work remains
indeterminate until reconciled with `query_result`.

### `reset_input`

```json
{
  "protocol": "expletives.automation",
  "version": 1,
  "type": "reset_input",
  "request_id": "reset-input-1"
}
```

The request clears all held keys for the controller's automation source. It
completes `applied` if any held state was cleared and `no_op` otherwise. The
server also clears source-held state automatically on disconnect. Automatic
disconnect cleanup publishes a new App snapshot only when held state
actually changed and has no request association.

### `shutdown`

```json
{
  "protocol": "expletives.automation",
  "version": 1,
  "type": "shutdown",
  "request_id": "shutdown-1"
}
```

`shutdown` invokes `app.quit` through the normal command-registry/router
policy with no `target_key`. It does not bypass application policy.

If the command completes `exited`, the App first publishes a final associated
snapshot. The hosting event loop may observe that final snapshot and begin
teardown before the automation worker has encoded its outer completion.
Because the accepted request is still marked processing, `Server.Close`
leaves that controller connected; the worker sends the exact final completion
and then closes the controller. Other outcomes are reported normally; only
`exited` makes this protocol operation close the server automatically.

`expletives-test` cancels its automation context, calls `Server.Close`, and
waits for `Serve` to return before the process exits. Consequently, an
ordinary `expletivesctl shutdown` receives its `exited` completion and final
snapshot instead of losing them to process teardown.

## Serialization, Ordering, And Time Bounds

The server executes requests from one controller synchronously in wire order.
A conforming version 1 controller waits for a request's completion before
sending another. The public Go client is safe for concurrent method callers,
but an internal mutex serializes their complete write/accept/completion
cycles. Which concurrent caller acquires that mutex first is unspecified.

Injected input and direct commands use the same App dispatch serialization as
other application input. Version 1 promises mutual exclusion and exact
per-request completion, but it defines no stronger fairness or priority
between an automation request and human or other App callers beyond actual
arrival at that serialization boundary. Atomic view Transactions use the
App's separate bounded mutation gate and do not enter command-dispatch
ordering.

Each connection read has a 30-second idle deadline. Normal request execution
receives a context with a 30-second deadline. The App command router runs
outside App state locks on a bounded four-slot callback executor. A router
that ignores cancellation may retain one slot, but the request returns at its
deadline and releases the App dispatch gate. Callback saturation produces a
bounded failure rather than an unbounded replacement goroutine. Each server
response write has a five-second deadline. The Go client uses the earlier of
its caller context deadline and the advertised read/write fallback. A
connection or client deadline is not proof that submitted work did or did not
execute.

The response encoder enforces `response_line_bytes` while emitting JSON and
stops when the limit is crossed. It does not first materialize an arbitrarily
oversized JSON record merely to measure and reject it.

## Request Identity, Retention, And Indeterminate Results

Request IDs are not an idempotency replay mechanism.

Before sending `accepted`, the server reserves the ID. An ID that is active
or present in the protocol retained-results ledger receives:

```json
{
  "protocol": "expletives.automation",
  "version": 1,
  "type": "protocol_error",
  "request_id": "toggle-1",
  "code": "duplicate_request_id",
  "message": "request ID is active or retained",
  "retryable": false
}
```

The server retains the three most recently completed protocol results in FIFO
completion order. One retained entry contains bounded terminal metadata and
its exact immutable `automation.SnapshotV1`; both are inserted and evicted
together. The three fixed slots and the 40 MiB per-response bound remain below
the version 1 128 MiB aggregate encoded-evidence budget. Active entries are
not evicted. Once an entry expires,
`query_result` returns `unknown_or_expired`, duplicate detection no longer
recognizes its ID, and that ID can be accepted again.

This server ledger is the single request-identity and reconciliation
authority. The root App receives the ID as completion-correlation metadata but
does not retain IDs, reject duplicates, or own protocol results. Clients
SHOULD still generate a fresh ID for every attempt: reuse after eviction does
not provide idempotency and makes operational reconciliation ambiguous.

The reference client returns `*automation.ResponseError` for a structured
pre-acceptance protocol rejection. It returns
`*automation.IndeterminateError` when a write or the wait for acceptance or
completion fails after submission could have occurred. It then closes the
connection. It never automatically retries.

After an indeterminate result:

1. retain the original request ID;
2. reconnect after the server releases the former controller;
3. issue `query_result` with a new query request ID; and
4. inspect the target's retained outcome and exact snapshot carried by the
   completed query.

`unknown_or_expired` cannot distinguish “never accepted” from “accepted but
no longer retained.” The caller must use application knowledge before
deciding whether a new operation is safe. It must never silently replay the
original ID or assume timeout means success.

## Errors

### Pre-Acceptance `protocol_error` Codes

| Code | Meaning | Connection |
| --- | --- | --- |
| `controller_busy` | Another controller owns the server. | Rejected connection closes. |
| `line_too_long` | Request line exceeds 65,536 bytes. | Framing-level occurrence closes. |
| `truncated_record` | EOF arrived after a record without `LF`. | Closes. |
| `invalid_utf8` | Framed request is not valid UTF-8. | Controller loop continues if the response can be written. |
| `malformed_json` | Invalid JSON, duplicate object key, excessive nesting, or multiple JSON values. | Normally continues. |
| `invalid_envelope` | Header could not be decoded into its required JSON types. | Normally continues. |
| `unsupported_protocol` | `protocol` is not `expletives.automation`. | Normally continues. |
| `unsupported_version` | `version` is not `1`. | Normally continues. |
| `unknown_type` | Request `type` is not a version 1 operation. | Normally continues. |
| `invalid_request` | Strict field, identifier, key, range, required-field, or nullability validation failed. | Normally continues. |
| `duplicate_request_id` | ID is active or retained. | Normally continues. |

Messages are trimmed, valid UTF-8, and at most 256 bytes. `request_id` is
included only when the decoder can safely recover a valid bounded ID.

### Accepted-Request Error Codes

| Code | Typical outcome | Meaning |
| --- | --- | --- |
| `snapshot_not_retained` | `rejected` | Exact observed sequence is unavailable. |
| `wait_timeout` | `failed` | Wait context ended or reached its server deadline. |
| `wait_failed` | `failed` | App wait failed for another reason. |
| `dispatch_failed` | `failed` | App key, command, or reset dispatch returned an error. |
| `associated_snapshot_unavailable` | `failed` | Exact App transition sequence could no longer be recovered. |
| `dispatch_panic` | `failed` | Server recovered a panic while dispatching the accepted request. |
| `target_not_found` | `rejected` | Optional `target_key` did not resolve to an active control. |
| `application_result` | application outcome | App supplied a bounded explanatory completion message. |
| `internal_unknown_operation` | `failed` | Defensive failure for an impossible accepted operation. |

The App input and command paths may return another valid bounded public code,
such as `held_key_capacity`, `input_source_capacity`, `command_unknown`,
`command_disabled`, `command_unhandled`, `handler_capacity`, or
`deadline_exceeded`. The outcome remains authoritative.

## Automation-Owned Snapshot DTO And Validation

The reference client rejects and closes on a completion that does not carry a
valid bounded `automation.SnapshotV1`. This DTO is declared in the automation
package and populated through explicit projection; it is not a type alias for
the root package's local snapshot. The client checks:

- `snapshot.version == 1`;
- nonnegative frame dimensions whose product is no greater than 4,194,304;
- at most 16,384 positive row-major runs whose counts sum exactly to
  `width * height`;
- a decoded expanded cell array length equal to `width * height`;
- every grapheme is one canonical cell of at most 64 UTF-8 bytes, and every
  semantic ID, resolved color, attribute set, and owner is bounded;
- at most 4,096 controls and at most 4,096 aggregate child references;
- at most 1,024 Layouts and 4,096 aggregate Layout item references, with
  bounded identities, geometry, arrangement indices, and stack indices;
- kind-consistent typed control details, including the generic zero-inset
  Container detail for root, Panel, Header, and Footer; canonical bounded
  border titles; Label/StaticText values and target/mnemonic associations;
  and Separator/Rule orientation, form, title, alignment, and wrapping state;
- generic focus plus Button label, command, enabled/disabled reason, checked,
  mnemonic, pressed/default/cancel state, and HotkeyBar ordered command state
  with canonical structured bindings;
- kind-consistent FocusGuideBar target kind—including ScrollBar,
  TabbedPanel, and Notebook—exact bounded resolved text, and
  empty/append/override customization state;
- at most 64 entries in one HotkeyBar and at most 4,096 aggregate HotkeyBar
  entries across one snapshot;
- at most one MenuBar, 512 aggregate flat Menu entries, 64 direct siblings,
  256 Menu models, and depth eight, with unique keys, depth-first
  parent-before-child structure, sibling mnemonic uniqueness, kind-consistent
  command/separator/submenu state, bounded structured chords, and consistent
  selected/open paths;
- at most one StatusBar and 64 copied ordered segments, with unique bounded
  keys, canonical effective labels, kind-consistent static or command state,
  structured chords, deterministic contiguous rendered Bounds, and explicit
  omission/clipping state;
- kind-consistent Checkbox, RadioButton/RadioGroup, and
  CycleField/SelectField detail members with canonical stable values,
  enabled/disabled reasons, exclusive radio selection, selected-index
  consistency, copied option state, at most 256 fixed options or Tabs per
  control, and at most 1,024 aggregate RadioButton/fixed-option/Tab records;
- kind-consistent TextField details with canonical bounded text, caret,
  selection, and horizontal view position, editing/valid/enabled state,
  copied validator policy, hard-validator consistency, and password value
  redaction; aggregate editor values plus validator sets are bounded to
  262,144 UTF-8 bytes;
- kind-consistent NumberField/SpinBox details with finite committed values,
  canonical current text, fixed precision, ordered copied bounds,
  validity/reason consistency, zero NumberField step, and positive SpinBox
  step;
- kind-consistent TextArea details with canonical LF-separated multiline
  text, logical line and element counts, caret/selection, visual caret and
  private viewport positions, wrapping, validation, and password redaction;
- kind-consistent ProgressBar, Meter, Spinner, and ActivityDots details with
  recognized status, exact bounded numeric state, finite ordered Meter
  ranges, canonical absolute ticks, reduced-motion state, text policy,
  orientation, and geometry-derived frame index;
- kind-consistent ScrollBar details with bounded viewport state, exact maximum
  offset, positive arrow/page steps, enabled/disabled policy, optional change
  command, orientation, and control-size-derived track/thumb geometry;
- kind-consistent TabbedPanel/Notebook details with unique ordered copied Tab
  keys, values, mnemonics, and direct-page references; exactly one selected
  and current record for a nonempty model; enabled-current policy; exact
  rendered/omitted/clipped strip geometry; selected-page Bounds; and
  nonselected effective invisibility;
- kind-consistent Viewport/ScrollablePanel details with canonical bounded
  ContentSize/Offset state, exact maximum offset, positive arrow/page steps,
  valid enabled/disabled and bar-visibility policy, an in-owner
  ViewportBounds, the exact sole managed Content ID/key/bounds relationship,
  and matching horizontal/vertical ScrollBarDetails state and geometry
  whenever those integrated bars are visible. Shared enabled policy,
  disabled reason, and change command occur once on the containing Scrollable
  record rather than being repeated by its integrated bar subrecords;
- kind-consistent MarkdownView details with bounded source metrics, block and
  rendered-row counts, no more than four first-two/last-two structural block
  summaries, truncation state, exact derived viewport geometry and bar
  visibility, and no retained source, managed-content identity, or repeated
  integrated-bar records;
- kind-consistent LogView details with valid bounded record/byte capacity,
  retained counts within that capacity, canonical retained byte accounting,
  cumulative drop counters, valid optional first/last retained keys, explicit
  follow state, and exact compact content-viewport geometry without retained
  record text;
- kind-consistent StreamView details with valid bounded line/byte capacity,
  retained and partial raw/canonical counts, pending truncation state,
  cumulative line/byte drops, explicit follow state, and exact compact
  content-viewport geometry without complete off-screen stream content;
- kind-consistent ListBox details with ready/loading/error evidence, bounded
  item/enabled/retained counts, stable current identity/index, exact selection
  cardinality/endpoints/SHA-256 digest, required-selection implications,
  enabled and command policy, and compact viewport geometry without retained
  item labels or status text;
- kind-consistent TreeView details with ready/loading/error evidence, bounded
  node/visible/enabled/retained counts, stable current visible identity/index,
  exact selection and expansion cardinality/endpoints/SHA-256 digests,
  required-selection implications, enabled and command policy, and compact
  viewport geometry without retained recursive nodes or status text;
- kind-consistent DropDown details with bounded item/enabled/retained counts,
  stable current and selected identities/indices, collapsed or exact open
  popup geometry, provisional current/selection, row cap, enabled policy,
  disabled-reason byte count, and optional change/activation commands without
  retained item labels;
- kind-consistent ComboBox details combining that compact popup record with a
  valid exact TextField-compatible editor record, matching enabled/disabled
  and change-command policy, no password mode, and no simultaneous popup-open
  and editor-editing state;
- aggregate ListBox, TreeView, DropDown, and ComboBox retained bytes within
  `MaxCollectionAggregateBytes`;
- aggregate retained Markdown, LogView, and StreamView content, including
  StreamView partial-line storage, within `MaxContentAggregateBytes`;
- at most 4,096 bounded input-source and overflow records, and at most eight
  valid held keys per source;
- exact equality of snapshot and completion frame sequences;
- recognized outcomes and query statuses; and
- bounded completion errors.

The DTO contains semantic cells, resolved style colors/attributes, controls,
Layouts, cursor, input sources, overflow state, and an optional sanitized
snapshot completion association. Server completions encode cells as
row-major runs of identical complete `Cell` values. The public Go client
validates and expands those runs into `Frame.Cells`; `expletivesctl` therefore
continues to emit an indexable `cells` array for inspection and also exposes
the compact `runs` evidence. If both views are decoded, they must agree
exactly. The DTO never contains a local-only command `Cause`, captured
terminal escape bytes, or physical-terminal degradation.

Version 1 additionally bounds each projected border title and display-control
text to 256 UTF-8 bytes, 256 normalized cells, and 64 bytes per canonical
cell. Control-detail union members must match the control kind. A nested public
snapshot-completion message is at most 1,024 UTF-8 bytes without NUL. A
top-level protocol `error.message` is trimmed to at most 256 valid UTF-8
bytes. Every encoded response must also fit the 40 MiB response-line limit.
The conservative legal-maximum completion proof is 41,391,622 JSON bytes;
three such records total 124,174,866 bytes and remain below the 128 MiB
aggregate encoded-evidence budget.

## Public Go Client Contract

The `automation` package exposes:

```go
func Dial(ctx context.Context, socketPath string) (*Client, error)
func NewRequestID() (string, error)

func (c *Client) Hello() Hello
func (c *Client) Observe(ctx context.Context, id string, sequence *uint64) (Completion, error)
func (c *Client) WaitSnapshot(ctx context.Context, id string, after uint64) (Completion, error)
func (c *Client) InjectInput(ctx context.Context, id string, event KeyEvent) (Completion, error)
func (c *Client) InvokeCommand(ctx context.Context, id, command, targetKey string) (Completion, error)
func (c *Client) QueryResult(ctx context.Context, id, targetID string) (Completion, error)
func (c *Client) ResetInput(ctx context.Context, id string) (Completion, error)
func (c *Client) Shutdown(ctx context.Context, id string) (Completion, error)
func (c *Client) Close() error
```

`Dial` reads and validates every required `hello` invariant before returning.
`Hello` returns a copy, including copied supported-version, operation, and
command slices. Client methods reject a nil context and validate request IDs
before writing. Methods are safe for concurrent callers but serialize
requests. `Close` is idempotent, interrupts an in-flight method, and normally
makes that method return an indeterminate error.

The client validates strict response envelopes, limits, matching operation
and request IDs, terminal completion shape, and snapshot bounds. A response
framing or correlation violation closes the client; callers must not continue
on a stream whose request/response alignment is uncertain.

## Public Go Server Contract

The public construction and lifetime surface is:

```go
type ServerOptions struct {
    App         *expletives.App
    SocketPath  string
    Application string
    Limits      Limits
}

func NewServer(options ServerOptions) (*Server, error)
func (s *Server) SocketPath() string
func (s *Server) Serve(ctx context.Context) error
func (s *Server) Close() error
```

`Application` is required and must be a bounded identifier. The reusable
server supplies no application or fixture-command defaults. It derives the
advertised command inventory from `App.Commands`, selecting definitions whose
`Automation` flag is true; more than 64 such entries or an invalid ID rejects
construction or the affected connection. The all-zero `Limits` value selects
`DefaultLimits`; every other supplied value must equal that fixed default
exactly.

`NewServer` binds the socket before returning, so its caller must call
`Close` after every successful construction. `SocketPath` returns the exact
path the server owns. `Serve` rejects a nil context, may be invoked only once,
accepts controllers until cancellation or shutdown, waits for controller
workers during teardown, and joins a cleanup failure into its returned error.
`Close` is idempotent. It does not wait for a processing controller itself;
the return of `Serve` is the worker-teardown join point.

## `expletivesctl` Contract

Global options precede the command:

```text
--socket PATH
--request-id ID
--dial-timeout DURATION
--timeout DURATION
--pretty=BOOL
--version
--help
```

`--socket` is required for every connected command and is never discovered
from an environment variable. `--dial-timeout` defaults to five seconds,
must be positive, and covers connection plus `hello`. `--timeout` defaults to
35 seconds, must be positive, and covers the complete requested operation or
multi-event sequence after dialing. That operation default leaves the
30-second server execution budget plus the response margin selected by the
correction design. `--pretty` defaults to `true`. `--version` and `--help`
require no socket.

Supported commands are:

```text
hello
snapshot [--sequence N]
observe [--sequence N]
wait [--after N]
key down|up|press KEY
keys down:KEY|up:KEY|press:KEY [...]
command COMMAND [TARGET_KEY]
result TARGET_REQUEST_ID
reset-input
shutdown
```

`snapshot` and `observe` are aliases. `key` accepts the short event kinds
`down`, `up`, and `press`; it also accepts their wire names. `keys` uses one
connection and issues its events sequentially, which preserves held
modifier state for a chord:

```text
expletivesctl \
  --socket /tmp/expletives.sock \
  --request-id toggle-chord \
  keys down:control press:r up:control
```

Each `expletivesctl` invocation otherwise opens and closes its own controller
connection. A chord must therefore use one `keys` invocation; held state from
a standalone `key down` is cleared before a later process can connect.

With a caller-selected base ID and multiple events, IDs are
`BASE-1`, `BASE-2`, and so on. With one event, the exact base is used. Without
`--request-id`, every operation or event receives an independent random ID.
Derived IDs must still fit the protocol's 64-byte identifier bound.

`hello` writes the `hello` object. Other commands write one completion object;
`keys` writes a JSON array when it has more than one completion. Output is
indented unless `--pretty=false`. Requests are never automatically retried.
A multi-event `keys` command has no transaction or rollback: an error can
occur after earlier events completed, and normal connection cleanup then
clears remaining held state.

Exit statuses are:

| Status | Meaning |
| --- | --- |
| `0` | `hello`, help, version, or every returned completion had outcome `applied`, `no_op`, or `exited`. |
| `1` | Dial, ordinary client/protocol, encoding/output, command-dispatch, or post-connect command-syntax error. |
| `2` | Global option/usage error, missing socket, nonpositive dial or operation timeout, missing command, invalid `hello` arguments, or `--version` with a command. |
| `3` | A returned completion had `rejected`, `cancelled`, `interrupted`, `failed`, or another non-success outcome. |
| `4` | The client reported `IndeterminateError`. |

A pre-acceptance `protocol_error`, including `duplicate_request_id`, is a
client error and therefore status 1. A completed request with a rejected or
failed outcome is valid JSON output and therefore status 3.

## Validation Evidence

The post-acceptance teardown contract was audited against the implementation
and revalidated on 2026-07-30 with:

```text
go test ./automation ./cmd/expletives-test
go test -race ./automation ./cmd/expletives-test
go test ./automation \
  -run '^TestCloseDrainsAcceptedProcessingCompletion$' -count=100
go test ./cmd/expletives-test \
  -run '^TestHeadlessAutomationShutdownDeliversFinalCompletion$' -count=100
```

All four commands passed.

The corrected response, text, content, and retained-evidence bounds were
separately revalidated on 2026-07-31 with:

```text
go test ./automation \
  -run '^(TestMaximumBoundedCompletionFitsResponseLine|TestSnapshotRejectsBorderTitleBeyondBound|TestSnapshotRejectsInvalidCanonicalText|TestSnapshotRejectsInvalidDisplayControlDetails|TestSnapshotRejectsInvalidMenuBarDetails|TestSnapshotRejectsInvalidStatusBarDetails|TestSelectionSnapshotProjectionValidationAndDeepCopy|TestSnapshotProjectsFocusGuideBarDetails|TestSnapshotRejectsInvalidFocusGuideBarDetails|TestSnapshotProjectsAndRedactsTextFieldDetails|TestSnapshotRejectsInvalidTextFieldDetails|TestSnapshotProjectsNumericFieldDetailsAndCopiesBounds|TestSnapshotRejectsInvalidNumberFieldDetails|TestSnapshotProjectsAndRedactsTextAreaDetails|TestSnapshotRejectsInvalidTextAreaDetails|TestSnapshotProjectsProgressDetailsAndCopiesState|TestSnapshotRejectsInvalidProgressDetails|TestSnapshotProjectsMarkdownDetailsAndCopiesBlocks|TestSnapshotRejectsInvalidMarkdownDetails|TestSnapshotProjectsLogAndStreamDetailsAndCopiesState|TestSnapshotRejectsInvalidLogAndStreamDetails|TestSnapshotProjectsListBoxDetailsAndCopiesViewport|TestSnapshotProjectsTreeViewDetailsAndCopiesViewport|TestSnapshotProjectsPopupCollectionDetailsAndCopiesState|TestSnapshotRejectsInvalidListBoxDetails|TestSnapshotRejectsInvalidTreeViewDetails|TestSnapshotRejectsInvalidPopupCollectionDetails|TestSnapshotRejectsAggregateChildReferencesBeyondBound)$' \
  -count=1
```

That command passed. The current maximum fixture encoded to 41,391,622 bytes;
three retained maximum records total 124,174,866 bytes.

The relevant integration assertions are:

- `automation/server_integration_test.go`:
  `TestServerClientClosedLoop` requires `Client.Shutdown` to receive an
  `exited` completion with a final snapshot, then requires `Serve` to return
  and the socket path to be absent;
- `automation/server_integration_test.go`:
  `TestCloseDrainsAcceptedProcessingCompletion` deterministically blocks
  inside an accepted command router, calls `Server.Close` while the request
  is processing, verifies that the socket path is removed, releases the
  handler, requires the client to receive its exact associated completion,
  and requires `Serve` to join; and
- `cmd/expletives-test/main_test.go`:
  `TestHeadlessAutomationShutdownDeliversFinalCompletion` runs the real
  headless application loop, connects through the Unix socket, opens View and
  Actions using raw Alt key lifecycles, changes catalog screens, opens and
  dismisses a nested menu, verifies flat Menu state and checked command
  presentation, requires the shutdown client to receive the final completion,
  and requires `expletives-test` to return status 0.

The race run exercises the same assertions under Go's race detector. These
tests establish the orderly in-process path; they do not turn peer failure,
forced process termination, or an uncooperative application handler into a
delivery guarantee.

## Version And Extension Policy

Version 1 is intentionally strict:

- clients and servers accept only the exact protocol name and operational
  record version;
- unknown operation-request and operational-response fields are rejected;
- `hello.supported_versions` is a bounded unique inventory and must contain
  version 1 for this connection;
- unknown additive `hello` fields are ignored after required-field
  validation;
- unknown request types, outcomes, and query statuses are rejected;
- a new operational field therefore requires a new protocol version for
  interoperability with the reference client;
- changing raw-key identity or lifecycle semantics requires a new version;
  and
- changing snapshot shape requires an appropriate snapshot/protocol
  compatibility decision.

An application may change the command inventory and scenario identity without
a protocol version change because both are advertised data and command policy
already handles rejection. A client accepts positive advertised limits no
larger than the version 1 maxima; raising a maximum requires a compatibility
decision. The current server itself accepts exactly the fixed set above.

Clients should inspect `hello.operations` and `hello.commands` rather than
assuming that every future application exposes the fixture inventory.

## Version 1 Non-Goals

Version 1 does not provide:

- authentication or capability authorization;
- hostile multi-user, elevated, or remote operation;
- TCP, abstract-namespace sockets, endpoint discovery, or ambient enablement;
- multiple simultaneous controllers;
- request pipelining or more than one executing request per connection;
- a protocol `cancel` operation or preemptive cancellation of the executing
  sequential request;
- automatic retries or idempotent replay;
- arbitrary escape-sequence, byte-stream, signal, paste, mouse, or committed
  text injection;
- terminal capture, screenshot persistence, or capture retention;
- physical-terminal access from an automation worker; or
- a fairness guarantee stronger than serialized arrival at the App dispatch
  boundary.

These exclusions keep the first closed loop deterministic and bounded. Later
capabilities require explicit design, security review, versioning, and tests.
