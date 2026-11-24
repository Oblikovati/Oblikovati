# Extrusion distance draft validation

Issue #3929 covers the desktop Extrude distance draft. The dialog asks the
active document owner to validate the exact expression. The owner uses the
same resolver that prepares a committed extrusion, including document units,
named parameters, dimensional checks, and positive finite length admission.
The result is valid only for its document, expression, and document generation.
A missing or stale result keeps OK disabled until the owner checks the draft.

Invalid input produces red diagnostic text in the panel. Changing the input
clears the old preview without creating a feature. A rejected feature command
returns the draft to the dialog with its reason, and Cancel removes a preview
without changing history. The commit control is named OK, matching the Go
reference. This change does not relax sketch readiness or certify kernel results.

The native regression covers zero, negative and wrong-unit values, an unknown
name, reference-free arithmetic, a named parameter, parameter changes while the
panel is open, cancellation, failed-profile retry, and one successful commit.
The host resolver test checks the same expression rules without a window.

## Automatic preview refresh (#3930)

A valid draft requests a transient preview from the owner. The request includes the
document generation used for validation. The owner declines an older generation
without replacing a newer preview. An invalid draft clears the displayed preview.
Parameter edits invalidate the previous validation and cause a new preview.

The desktop frame dispatches GUI actions in order and computes previews inline.
There is no background preview completion that can arrive after an edit or Cancel.
Cancel clears the transient scene without creating a feature or changing history.

## Flipped direction (#3893)

Desktop creation and preview pass the same direction through the typed Extrude
request. Flipped uses the existing negative direction value and keeps the
Distance A expression positive. The prism recipe records the reversal
separately and applies it to the sketch-normal displacement after resolving
the distance. Existing recipes default to their original signed displacement.
An accepted extrusion keeps its direction during a distance edit.

See the [Flipped direction receipt](receipts/2026-11-24-extrusion-flipped-3893/README.md).

## Symmetric direction (#3893)

The symmetric direction keeps Distance A as the total depth. Replay places
the sweep start half that distance behind the resolved sketch plane and
uses the full positive displacement. The sketch's own placement stays
available to the document owner. The certified adapter builds one prism
from the translated profile and uses the same source curve and vertex
identities. The conditional sweep report requires a sketch-plane start and
declines centered spans until it can bound the shifted start.

See the [Symmetric direction receipt](receipts/2026-11-24-extrusion-symmetric-3893/README.md).

## Asymmetric direction (#3893)

Asymmetric extrusion uses a positive Distance A and a separate positive
Distance B. Replay starts at `-B` along the sketch normal and sweeps by
`A + B` through the same certified prism operation. The typed request uses
`secondDistance` to select this mode. Both lengths have independent
parameter reads, and both are available through the feature's scalar slots.
Edit Feature can change either length or swap the pair in one history step.

The desktop exposes Taper B as a disabled zero-angle field pending a
validated tapered sweep. See the
[Asymmetric direction receipt](receipts/2026-11-24-extrusion-asymmetric-3893/README.md)
and [Martensite #463](https://github.com/Oblikovati/Martensite/issues/463).

## Direction selector style (#3893)

The Direction menu switches between four labeled icons and a list of the
same four directions. The dialog starts in Icons style and remembers the
style for the desktop session, including after Cancel. Changing
presentation preserves the current direction and preview. Selecting a
direction from either style sends the same draft through validation.

See the [Direction selector receipt](receipts/2026-11-24-extrusion-direction-selector-3893/README.md).

## Reference extents (#3893)

The typed Extrude command accepts To Face and From–To extents for new solids.
Each target can be a named planar face, an origin plane, or an indexed work
plane. An index resolves to the work plane's stable identity when the command
is authored. Replay resolves the current planes against the feature prefix,
validates that they are parallel to the profile, and passes the ordered normal
span to the same certified prism operation as a numeric extrusion. The recipe
stores the references and reads their producing features. A reference span
has no distance parameter.

See the [Reference extent receipt](receipts/2026-11-24-extrusion-reference-extents-3893/README.md).

The desktop From and Termination controls can switch the draft to From–To
or To Face. The selector arms a specific endpoint for the next viewport
face pick. Missing targets clear the preview and disable OK. When all targets
are selected, the owner validates the exact draft at the active document
generation by previewing it through the same replay as acceptance. Switching
back to Distance restores the numeric draft.

See the [Desktop reference controls receipt](receipts/2026-11-24-extrusion-reference-controls-3893/README.md).

Geometric face selectors in the typed API, desktop work-plane picking,
taper, Through All, and To Next remain open.

## Distance A editor menu (#3893)

The Distance A editor menu opens Measure, the active part's Parameters table,
or a list of its parameters. Choosing a length parameter adds its name to
the draft; the document owner evaluates that expression through the same
preview and confirmation path as typed input. Select Feature Dimension
prompts for a feature and offers length parameters read by that feature.
The Parameters table displays consuming features from the model read graph.
Done closes the table while the Extrude draft stays open.

For an editable parameter in Distance A, Tolerance opens the existing
parameter tolerance editor. The full format and controls from pinned
`extrusion-12` remain open.

See the [Distance A editor receipt](receipts/2026-11-24-extrusion-distance-editor-3893/README.md).
