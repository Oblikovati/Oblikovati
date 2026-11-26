# Atomic portable assignment edits — partial R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563),
under ADR-0019. The source behavior is `app/material_ops.go`:
`AssignMaterial`, `AssignAppearance`, `embedMaterial`, and `embedAppearance`.

The document model now prepares a complete assignment candidate from immutable
inputs. Physical targets are part or body; visual targets are part, body, or face.
Entity targets retain the existing occurrence-aware reference value. These values
express authored intent; they do not prove that a body or face exists.

Selection uses document, project, then builtin scope, as the existing resolver
does. A physical assignment copies a non-builtin material and its selected
appearance into the document. A builtin material can still refer to a project
appearance, which must be copied. A visual assignment copies only the appearance
and cannot change physical intent. Every property group remains API-owned.

Missing requested assets or appearance dependencies refuse the candidate before
adoption. The previous document stays unchanged. Builtin definitions remain
references. Empty IDs clear the selected intent without requiring its old asset
to exist, as the model assignment setters specify. Clearing does not delete
embedded definitions; other references can still need them. A material without
an appearance ID needs no visual copy and keeps the resolver's neutral behavior.

Model tests cover property preservation, all target kinds, dependency failure,
builtin references, document scope precedence, occurrence isolation, and clearing.
Integration tests use the ordinary document command owner and named memory
filesystem. They verify one undo step, redo, transaction abort, dirty state,
save/reopen, and typed queries after reopening without any project library.
Run `cargo test -p oblikovati-model --test asset_assignment_edits` and
`cargo test -p oblikovati-automation --test queries assignment_history`, or
`make gate` for the full tree.

The existing typed API assignment methods are now advertised by the file owner.
Part targets require an active part. Body and face targets additionally require
an injected `PartTopologyAccess`; a file-only host cannot accept raw entity
keys. The desktop supplies this capability from its current named part results.
It resolves the exact feature namespace and checks document, kind, generation,
and the complete key before mutation. Unsupported or failed topology is a named
decline, never an unverified assignment.

The request boundary validates hexadecimal keys and canonicalizes their envelope
and text case. The wire commands reject empty asset IDs, matching Go
`app/material_ops.go`; the internal model setters still support clearing intent.
Part appearance requests require the contract's empty key. All asset dependencies
are prepared before one ordinary history edit. Failure leaves assets, assignments,
and history unchanged. An explicit transaction can abort the complete edit.

Tests drive the API-owned clients, including portable project copies, save/reopen,
undo/redo, transaction abort, scope errors, and missing topology capability. Native
host tests bind actual body/face names, refuse wrong kinds and foreign namespaces,
quarantine failed features, preserve explicit appearance during material edits,
and retain a pending first scene after rejected asset lookup. Reference validation
and rendering share the same result cache; validation cannot consume a display
update that the window has not received.

Full appearance rendering, lost-reference health, remapping across general feature
edits, occurrence publication, and foreground MCPBridge acceptance remain open.
The current native verifier covers the retained sketch-prism capability and does
not claim unsupported feature families. No kernel geometry algorithm or public
wire definition is introduced.

## Material Browser

The library-row context menu assigns a material to the selected Part root
through the existing document assignment command. The Material selector
shows the assigned asset and the Appearance selector displays the
material's appearance until the user sets an explicit override.
See the [native Material Browser receipt](receipts/2026-11-26-material-browser-3563/README.md).

## Appearance Browser

Document and library appearance previews use authored colors and finishes.
Activating Assign to Selection in the library menu stores an explicit
appearance override on the selected Part. The Material selector continues
to display its physical assignment. See the
[native Appearance Browser receipt](receipts/2026-11-26-appearance-browser-3563/README.md).

## Generic Appearance Editor

The Appearance Browser creates document-owned Generic assets and opens
the selected asset in a separate editor. Creation and Apply record
document history edits. Pending color, name, glossiness, and highlight
changes are available in the live preview. The editor can reopen a
selected document appearance from the browser. See the
[native editor receipt](receipts/2026-11-26-appearance-editor-generic-3563/README.md).

The Information section edits a document appearance's description and
keywords. The optional `appearanceMetadata` recipe map is keyed by
appearance identity and travels with the Part. Older Parts read with
blank values. Saving an empty description and keywords removes the
metadata entry. See the
[native Information receipt](receipts/2026-11-26-appearance-information-3563/README.md).

Reflectivity has separate disclosure and enable controls. Direct and
Oblique affect the apparent reflection in the editor preview and, after
Apply, in the browser thumbnail. The document recipe keeps both
percentages when the effect is disabled. See the
[native Reflectivity receipt](receipts/2026-11-26-appearance-reflectivity-3563/README.md).

Transparency previews the material over a refracted checkerboard. Amount
sets the transmitted contribution and Translucency softens the contrast
of the transmitted pattern. The selected preset and custom index are
stored with the appearance. See the
[native Transparency receipt](receipts/2026-11-26-appearance-transparency-3563/README.md).

Cutouts, Self Illumination, Bump, and Tint have independent disclosure
and enable controls. Expanding a disabled section leaves the preview
and document unchanged. Enabled Tint and Self Illumination change the
preview, and Apply stores the authored settings with the Part. The
Cutouts and Bump image slots remain inactive pending texture editing.
See the [native optional-effects receipt](receipts/2026-11-26-optional-appearance-effects-3563/README.md).

The Generic Image menu opens the Checker Texture Editor. The editor
changes the draft colors, softness, sample size, transforms, and repeat
modes. Image Fade controls the blend between the base color and texture.
Apply stores the Checker in the Part, and the Asset Browser uses it in
the saved appearance thumbnail. See the [native Checker receipt](receipts/2026-11-26-appearance-checker-3563/README.md).

Preview Options switches the sample between Sphere and Cube and exposes
Grid Light, Plaza, Snowfield, and Warm Light. Quick is the active render
setting. Scene and lighting choices live in the open editor and do not
mark the appearance draft dirty. See the [native preview receipt](receipts/2026-11-26-appearance-preview-3563/README.md).

The Appearance Browser creates a document-owned Ceramic appearance with
a white glazed Porcelain finish. The shared editor exposes Ceramic and
Porcelain types, the base color, and Glazed, Satin, and Matte finishes.
Type and finish are stored in the Part recipe alongside the common
surface parameters. Ceramic also has independent Finish Bumps, Relief
Pattern, and Tint sections. See the [native Ceramic receipt](receipts/2026-11-26-appearance-ceramic-3563/README.md).

Ceramic Finish Bumps accepts Wavy or a custom height image. Relief Pattern
uses the same image chooser. The editor samples PNG, JPEG, or BMP images
into 32×32 grayscale maps for the preview and embeds the maps in the Part
when Apply commits the draft. Cancel preserves the previously selected
type and map. See the [native ceramic map receipt](receipts/2026-11-26-appearance-ceramic-maps-3563/README.md).
