# Assembly Finish and Simplify evidence

Issue #3963 remains open. The document-opening scenario records that Finish and
Simplify are available after Load Full, but does not identify their ribbon
panels, tooltips, inputs, results, or behavior when clicked. Its live source
`features/application/document-opening/document-opening.feature` has SHA-256
`d7965f270ecc21d99f2ed9a0837f17a1996203fccf4d4d1966e7876ebb01464c`, which
matches `bdd-source-inventory.json`. The scenario folder's `BLOCKERS.md` asks
its owner to specify both commands. The related `FUNCTIONAL.md` describes
visible state only. No command behavior is verified by that observation.

The Go reference at `aa8297d1b0ac33c024f19e54821536d76e8a447a` does not resolve
this ambiguity. `app/commands_standard.go` registers `Manage.Simplify` only
when a part is active. Its `SimplifyTool` removes picked faces or fills voids
in a part; it is not an assembly tool. Go has a cosmetic `FinishFeature` for
part faces, but the command inventory found here has Finish Sketch and Finish
3D Sketch rather than an assembly Finish command. The separate in-place-edit
Return work is tracked by #3960; no evidence links it to this scenario's label.

The Inventor-derived C# reference at `d4958b722c2c943c380b7b09b84dacc7f3d58029`
contains several distinct operations:

- `AssemblyComponentDefinition.Features` exposes `Features.FinishFeatures`.
  `FinishFeatures.CreateDefinition` takes included faces, bodies, and proxies,
  optional excluded faces, a finish type, process, and appearance. The general
  assembly `Features` documentation also says its API does not yet support
  adding assembly features. These declarations alone do not establish which
  ribbon control the scenario observed or its supported interactive behavior.
- `PartFeatures` exposes both `FinishFeatures` and `SimplifyFeatures`.
  `SimplifyDefinition` describes a part feature, with body exclusion, envelope,
  hole, fillet, chamfer, and other removal settings. The source `solid-model`
  Finish and Simplify scenarios describe part workflows. They do not specify
  an assembly workflow after Load Full.
- `DerivedAssemblyDefinition` describes deriving an assembly into a part and
  includes simplification options. Issue #631 describes this distinct derived
  assembly and shrinkwrap path. It does not identify the observed ribbon control.

The Rust assembly picker currently returns occurrence paths and discards the
leaf face hit. Stable assembly-face candidates and their named resolution are
still part of #3890. That is a host dependency for any confirmed workflow that
selects component faces. It is not evidence that every possible Finish or
Simplify command requires that selection. No specific Martensite capability gap
has been established for the unidentified assembly commands.

Before enabling either command, obtain its observed ribbon location and identity,
its target document and selection, the action and resulting model or view state,
and its cancel, edit, undo, and save behavior. Then implement the confirmed
operation through the shared host and public contract, with native mouse and
keyboard tests. Keep the scenario assertion and #3963 open in the meantime; an
enabled control with no defined action is not acceptance evidence.
