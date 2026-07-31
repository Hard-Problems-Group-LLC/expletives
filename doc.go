// Package expletives provides a thread-safe character-cell UI toolkit.
//
// An App owns one immutable-parent control tree, semantic Theme, command
// registry, input state, renderer, and atomic Snapshot stream. Panel, Frame,
// and GroupBox are copy-safe handles over canonical toolkit nodes. Controls
// retain semantic StyleID values; the App Theme resolves them for intended
// frames and observation.
//
// BoxLayout and GridLayout form a separate nonvisual arrangement and stacking
// tree without changing immutable control parentage. Layout and Panel
// Raise/Lower operations change paint order without changing arrangement.
//
// Related changes should be grouped in a Transaction so observers receive one
// complete resulting Snapshot. Raw KeyEvent values and direct Command values
// share the same registered routing path. Application command callbacks run
// outside toolkit state locks on a bounded executor, which makes the package a
// practical view/controller boundary for MVC, MVVC, and similar application
// structures without requiring toolkit types in the model.
//
// Physical terminal ownership and byte decoding live in the terminal
// subpackage. Opt-in local drive-and-observe support lives in the automation
// subpackage.
package expletives
