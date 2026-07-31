# Layouts And Overflow

Status: Implemented directed behavior; exact public types are fixed in
[`layout-api-v0.md`](layout-api-v0.md)
Authority: Direct operator decisions on 2026-07-24
Related decisions: `EXPL-DEC-004` and `EXPL-DEC-007` in
[`project-management/decision-log.md`](../../project-management/decision-log.md)
Related proposal:
[`EXPL-PROP-001`](../../project-management/proposals/under-review/expl-prop-001-foundational-window-presentation-automation-layouts.md)

## Purpose

This specification defines the initial attachment, undersized-geometry, and
notification contracts for `BoxLayout` and `GridLayout`. It keeps Layout
calculation deterministic, makes insufficient space inspectable, and lets an
application respond without introducing callback reentrancy into measure or
arrange.

## Layout Attachment

A Layout is instantiated independently of a Panel. It becomes active only
through an atomic:

```text
Panel.SetLayout(layout)
```

The operation validates the entire attachment before mutating either object.
At minimum, it rejects a Layout already owned by another Panel, an invalid
cross-application relationship, and an attachment that would violate the
direct-child-Panel contract. Failure leaves the Panel, Layout, control tree,
logical geometry, effective clips, and current frame unchanged.

An attached Layout is a non-control object owned by exactly one Panel. It does
not become a second parent for its items. Its Panel items are direct child
Panels of that owner. A Panel has one primary Layout and may add further
top-level Layout stacking contexts. A Layout may contain another Layout; all
Panel items in that tree remain direct control children of the top-level
owner. Child Panels may independently own their own Layout trees.

Replacement and detachment remain deferred. `SetLayout` installs only the
first top-level Layout, while `AddLayout` requires that primary and adds a
sibling stacking context. Neither operation reparents controls.

## Logical Geometry And Effective Clipping

Every arranged child has two related geometric facts:

- its **logical Layout rectangle**, produced deterministically from minimum
  sizes, order, gaps, insets, grow weights, and alignment; and
- its **effective clip**, formed by intersecting that logical rectangle with
  every ancestor clip and the application surface.

Ordinary paint, hit testing, cursor placement, and physical presentation cannot
escape the effective clip. Automation exposes both facts so a clipped control
is not misreported as having a smaller logical minimum.

## Below-Minimum Behavior

Overflow exists when the owning Panel's available content rectangle is smaller
than the Layout's combined required minimum in either dimension.

The initial policy is:

1. preserve each item's logical minimum and deterministic ordering;
2. retain the resulting logical Layout rectangles even where they extend past
   the available content rectangle;
3. clip painting, hit testing, cursor presentation, and descendant output to
   the effective clip;
4. publish structured overflow state; and
5. dispatch the configured application notification policy.

The Layout does not silently compress a Panel below its declared minimum,
overlap unrelated items as an accidental substitute for space, mutate the
parent tree, or create negative geometry.

Overflow state is recalculated on every relevant Layout pass. Recovery clears
the active state when the required minimum again fits.

## Structured Overflow State

The bounded typed semantic view must identify, at minimum:

- the owning App, Panel, and Layout;
- the available content size and combined required minimum;
- horizontal and vertical deficit;
- the affected direct-child Panel identities and their logical rectangles and
  effective clips;
- whether the episode entered, remains active, or cleared; and
- the bounded notification state, such as pending, application-handled,
  default-active, acknowledged, failed, or suppressed; and
- the associated immutable frame sequence.

The exact Go and wire field names remain under design. The information cannot
be hidden solely in a log message or callback argument because headless tests
and attached automation must inspect it without installing application code.

## Overflow Callback

An application may register an `Overflow` callback or handler. The exact
function and disposition types remain under design, but the behavior is fixed:

- the Layout pass records overflow; it never invokes application code inline;
- the toolkit queues notification only after measure and arrange have
  completed and all Layout/internal locks have been released;
- the callback runs on a bounded application-callback dispatcher, not the
  Layout, render, presentation, or UI owner, and may select handled or
  default-fallback disposition;
- the handler receives a documented cancellation/deadline signal; a result
  returned after the delivery bound is ignored and the fallback is selected;
- long-running work must be posted elsewhere rather than blocking the callback;
- callback-driven mutations are scheduled as later transitions, not nested
  Layout passes; and
- registration, replacement, shutdown, and concurrent use follow the toolkit's
  public thread-safety and owner-marshaling contract.

An overflow-producing request may complete after the structured fact is
published and bounded notification delivery is queued. Its associated snapshot
exposes the active episode and pending notification. It does not wait for
application callback execution or human dismissal. Callback disposition and
default-notification changes publish later sequenced snapshots so their
outcomes remain observable without letting a blocking application callback
hang closed-loop request completion.

Go cannot forcibly stop application code that ignores cancellation. The
dispatcher therefore bounds in-flight callbacks and retained goroutines and
does not create a replacement goroutine per episode. When the dispatcher is
unavailable, saturated, cancelled, panicked, shut down, or misses the
disposition deadline, the notification records that delivery failure and
selects the same default fallback. A stuck callback may consume only its
bounded dispatcher slot; it cannot block Layout, painting, input, automation
request completion, or an unbounded sequence of new workers.

## Episodes And Coalescing

An overflow episode begins when one owning Panel/Layout pair transitions from
fitting to overflowing and ends when it fits again or is destroyed.

When an application handler applies, exactly one callback delivery attempt is
queued for that pair during one episode. Without a handler, exactly one
default-notification attempt is queued. Further resizes update the structured
deficit and geometry but do not generate another callback or popup. Recovery
followed by a later overflow begins a new episode and notifies again.

Multiple simultaneous Layout overflows remain individually observable. The
default human notification may summarize a bounded set of them rather than
stacking one warning per Layout.

## Default Fallback

An absent handler, or a handler that requests the default disposition, cannot
make overflow silent. The default behavior is:

1. retain the structured overflow state in every applicable snapshot;
2. present one compact, dismissible application-level warning with an `OK`
   action when interactive geometry permits;
3. adapt the warning down to a bounded high-visibility indicator when the
   surface is too small for the full message; and
4. retain semantic and diagnostic evidence even when a zero-sized surface
   makes physical notification impossible.

The foundational fallback may use a small private notification overlay; it
does not require the later public `Dialog` or `MessageBox` phase. That overlay:

- is outside ordinary Layout minimum calculation;
- cannot itself generate another Layout-overflow episode;
- never changes the underlying overflow state;
- appears at most once per active episode or bounded group of episodes;
- has keyboard and semantic-command dismissal paths; and
- may later be implemented with a public modal control only if the observable
  contract remains the same.

Dismissing the warning acknowledges the notification; it does not claim that
geometry now fits.

## Headless And Attached Automation

Default notification never pauses the event loop waiting for `OK`. A headless
or attached request completes after the overflow fact and queued notification
state are published. Automation can then:

- inspect the active overflow records;
- inspect whether the application handled the event or the default warning is
  active;
- dismiss the warning through the same key or semantic command path available
  to a human;
- resize and observe updated deficits without repeated notification; and
- resize back above minimum and observe the cleared episode.

Human and automation input remain fairly serviced while the warning is active.
Timeout, shutdown, and exit behavior remain explicit.

## Verification

Ordinary Go, race, headless, `expletives-test`, and attached-automation tests
must cover:

- successful atomic `Panel.SetLayout` attachment and every rejected
  attachment leaving both objects unchanged;
- exact logical rectangles and effective clips below minimum;
- structured horizontal, vertical, and two-axis deficits;
- a callback invoked after Layout and outside internal locks;
- dispatcher saturation, panic, cancellation, deadline expiry, and a handler
  that ignores cancellation, all with bounded retained workers and fallback;
- callback mutation being deferred without nested Layout or deadlock;
- once-per-episode coalescing, recovery, and recurrence;
- multiple simultaneous overflows with bounded default notification;
- handled and default dispositions;
- default warning display, acknowledgement, and continued underlying overflow;
- tiny and zero geometry without recursive warning overflow;
- request completion associated with the overflow-and-notification-queued
  snapshot, followed by sequenced callback/default-notification snapshots;
- headless and attached dismissal without waiting on a human; and
- shutdown, destruction, resize races, and race-detector coverage.

## Deferred Details

The following require later API decisions without reopening the directed
behavior:

- Layout replacement or detachment through `Panel.SetLayout`;
- exact warning text, style, and smallest visible indicator;
- aggregation limits for simultaneous episodes; and
- optional suppression or rate-policy configuration beyond once-per-episode
  coalescing.
