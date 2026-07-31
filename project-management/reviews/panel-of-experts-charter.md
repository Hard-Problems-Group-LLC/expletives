# Panel Of Experts Charter

- Status: Adopted
- Effective: 2026-07-25
- Simplified: 2026-07-30 by direct operator instruction
- Process owner: project operator

## Purpose

The expletives Panel of Experts is an exceptional review mechanism for major,
cross-cutting risks. It is not the project's normal implementation, bug-fix,
or code-review workflow.

Ordinary work uses the smallest process that establishes confidence:

- implement against the current specifications and decisions;
- add proportionate unit, integration, automation, race, fuzz, PTY, or manual
  checks;
- use focused peer review when another perspective is useful; and
- record only durable decisions, defects, and verification evidence.

Using a sub-agent, asking one specialist a question, or performing a normal
code review does not convene the Panel. A review is a Panel review only when
it is explicitly declared as one.

The Panel is advisory. It does not approve scope, waive requirements, defer
work, accept risk, or overrule the operator. The operator retains those
decisions.

## Standing Expert Lenses

The standing board contains nine lenses:

| Seat | Lens |
| --- | --- |
| 1 | Maintainability |
| 2 | Orthogonality and design elegance |
| 3 | Extensibility |
| 4 | Customizability |
| 5 | Client ease of use |
| 6 | Testability |
| 7 | Security |
| 8 | Performance |
| 9 | Reliability, concurrency, and portability |

These are a roster of available perspectives, not a requirement to run nine
reviews for every change.

## Convening Threshold

Do not convene the Panel by default. Convene it only when the operator
requests it or when the project lead identifies a genuinely major unresolved
decision for which ordinary design work and focused review are insufficient.
Strong candidates include:

- a foundational control, event-loop, rendering, layout, or concurrency model
  that will constrain several future phases;
- a new externally reachable trust boundary or a material security model;
- a public compatibility break or migration with substantial downstream or
  data-loss risk;
- a product-wide performance or reliability decision that is expensive to
  reverse; or
- an unresolved dispute with multiple credible designs and material
  consequences.

The following do not justify a Panel by themselves:

- routine implementation within an accepted design;
- localized bug fixes, refactoring, tests, documentation, or diagnostics;
- compatible, bounded additions whose design follows existing patterns;
- mechanical resolution of an already-reviewed finding; or
- the desire for general reassurance.

If uncertain, use focused review and tests. Escalate only if that work exposes
a major unresolved issue.

## Proportional Composition

Every convened Panel uses an odd number of expert seats:

- three seats for a narrow major issue;
- five seats for a cross-cutting issue; or
- all nine seats only for a foundational or product-wide architecture review.

Select only the lenses material to the decision and name them in the
convening brief. The selected seats form the quorum for that review. There is
no majority risk-acceptance vote; an odd membership prevents an indecisive
board recommendation while supported minority concerns remain visible.

A facilitator or evidence chair may coordinate the review. A separate
independent chair is required only when the operator requests one or when the
review concerns a contested security, safety, or evidence-integrity claim.

## Lean Review Contract

A normal Panel review uses one concise, immutable brief containing:

- the decision or change under review;
- scope and explicit non-goals;
- the important alternatives and risks;
- acceptance criteria;
- links to the exact commit, diff, specification, and verification evidence;
  and
- selected seats, facilitator, and response deadline.

Prefer a committed revision or an exact diff. A separate file-by-file
SHA-256 manifest is needed only when an uncommitted mutable snapshot must be
reviewed and cannot be identified more simply.

Each seat returns at most three risk-ranked findings, plus any additional
blocker. Findings use a compact table or structured record with claim,
evidence, consequence, and recommendation. Reviewers are time-boxed and read
only material relevant to their assigned lens.

Use one findings record and one outcome/addendum for the review. Do not create
one administrative file per finding unless a finding independently becomes a
tracked bug, proposal, or decision.

## Rounds And Closure

One review round is the default. The sponsor responds with fixes, evidence,
or a concise rebuttal. A targeted follow-up goes only to the seat whose major
or blocker finding remains unresolved. Do not reconvene unaffected seats.

A second full round or separate delivery review requires an explicit reason:
material redesign, disputed evidence, or a new major risk. There is no
automatic design-plus-delivery pair.

Before closure, every confirmed blocker or major finding must be fixed and
verified, explicitly deferred, rejected with the design, or accepted as risk
by the operator. Minor suggestions use the normal backlog only when their
value exceeds their tracking cost.

## Records

Panel records live under `project-management/reviews/`. The minimum record is:

- the convening brief;
- one combined findings record; and
- one closure record linking fixes and verification or operator decisions.

Durable behavior belongs in `docs/specifications/`; durable choices belong in
the decision log. Panel artifacts are evidence, not a second source of truth.

## Existing Foundation Review

`EXPL-REV-001` was intentionally a comprehensive nine-seat initial
architecture review. Its frozen packet, raw findings, and chair adjudication
remain valid historical evidence.

Its correction cycle closes under the operator's 2026-07-30 simplification:
one shared correction proposal, implementation and verification evidence, and
a concise chair addendum. It does not require another nine-seat round or an
automatic full delivery review unless the operator explicitly requests one.
