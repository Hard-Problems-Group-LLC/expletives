# Imported ECR: Keep Disabled Button Labels Visible

- Source ID: `radioradio-ECR-2026-001`
- Revision: 1
- Target: Expletives
- Received: 2026-08-08 by direct operator instruction
- Source-file SHA-256: `4ca3a14e2c168a89577554cda17e7c9139dfabf02f96a8e377c88a09e832e9fe`
- Sensitivity: reusable defect report; no traffic content

## Received Request

The default `button.disabled` semantic style uses RGB `#808080` for both
foreground and background. The Button retains its label structurally, but the
equal colors make every grapheme invisible. radioradio observed this on its
disabled Chat Send button and requested practical Turbo Vision-consistent
contrast without changing labels, command state, focus, or activation.

Acceptance requires distinct resolved foreground/background colors, visible
retained label graphemes using `button.disabled`, preserved semantic disabled
evidence, rendering regressions for ordinary and standard-dialog Buttons, and
usable supported terminal-profile projection. radioradio locally mitigates by
overriding only the disabled foreground to black.
