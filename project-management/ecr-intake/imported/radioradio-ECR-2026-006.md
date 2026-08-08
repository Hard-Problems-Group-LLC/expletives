# Imported ECR: Prevent Negative ListBox Viewport Offsets

- Source ID: `radioradio-ECR-2026-006`
- Revision: 1
- Target: Expletives
- Received: 2026-08-08 by direct operator instruction
- Source-file SHA-256: `ae6eeb2e28f60a278bde9bfb3b48e05536fd68806692adc0ce4b04f2d08c42ab`
- Sensitivity: reusable control-state defect; no traffic content

## Received Request

ListBox Left Arrow subtracts the configured horizontal step at origin, while
the shared viewport clamp enforces only the maximum and allows the negative
candidate to survive. radioradio observed repeated Left Arrow shifting a
wrapped Monitor Columns list into invalid blank space even with no horizontal
overflow and a `Never` horizontal-bar policy.

Acceptance requires every committed offset to remain between zero and its
calculated maximum; handled no-ops at both boundaries; stable zero horizontal
offset for fitting wrapped lists; clamping after content and geometry changes;
unchanged logical navigation/selection/activation; nonnegative core and
automation evidence; and focused unit plus consumer-equivalent regression
coverage.
