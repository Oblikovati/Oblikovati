# Inventor Ribbon Structure — canonical reference

**This is the source of truth for the ribbon layout** (tabs → panels → buttons) Oblikovati
targets for parity. Every ribbon command's **tab** and **panel** placement (the `.WithTab`
+ panel arg in `app/commands_standard.go`) must match this tree. When adding or moving a
command, place it in the panel named here — do not invent panels.

Captured from Autodesk Inventor 2026.1 (Default). Internal names (`id_Tab…`) are Inventor's
command-IDs, kept for traceability. Oblikovati uses the same tab/panel **names**; button
availability is incremental (see PROGRESS.md), but placement must follow this structure.

> See also: [core/09-ui-imgui.md](../core/09-ui-imgui.md) (how the ribbon is built from the
> command registry) and the per-feature DoD in
> [implementation-plan/CONVENTIONS.md](../../implementation-plan/CONVENTIONS.md).

## Canonical structure

```yaml
autodesk_inventor_ribbon:
  version: "2026.1-Default"
  environments:

    - name: "ZeroDoc Environment"
      internal_name: "ZeroDoc"
      description: "Ribbon layout active when no files are open (My Home / Startup state)"
      tabs:
        - name: "Get Started"
          internal_name: "id_TabGetStarted"
          panels:
            - name: "Launch"
              buttons: [New, Open, Projects, Configure Default Templates]
            - name: "New File"
              buttons: [Part, Assembly, Drawing, Presentation]
            - name: "My Home"
              buttons: [Home, Reset Layout]
            - name: "Support"
              buttons: [What's New, Help, Tutorials, Community, Guided Tutorials]

        - name: "Tools"
          internal_name: "id_TabTools"
          panels:
            - name: "Options"
              buttons: [Application Options]
            - name: "Customize"
              buttons: [Customize User Commands]
            - name: "iLogic"
              buttons: [iLogic Browser, iLogic Event Triggers]

    - name: "Part Environment (.ipt)"
      internal_name: "Part"
      description: "Standard 3D solid part design workspace"
      tabs:
        - name: "3D Model"
          internal_name: "id_TabModel"
          panels:
            - name: "Sketch"
              buttons: [Start 2D Sketch, Start 3D Sketch]
            - name: "Create"
              buttons: [Extrude, Revolve, Sweep, Loft, Coil, Rib, Emboss, Decal]
            - name: "Modify"
              buttons: [Hole, Fillet, Chamfer, Shell, Draft, Thread, Split, Direct, Combine, Thicken/Offset]
            - name: "Work Features"
              buttons: [Plane, Axis, Point, Coordinate System]
            - name: "Pattern"
              buttons: [Rectangular, Circular, Sketch Driven, Mirror]
            - name: "Freeform"
              buttons: [Box, Cylinder, Sphere, Torus, Quadball, Plane, Face]
            - name: "Surface"
              buttons: [Patch, Stitch, Sculpt, Extend, Trim, Rule Fillet]

        - name: "Sketch"
          internal_name: "id_TabSketch"
          panels:
            - name: "Create"
              buttons: [Line, Circle, Arc, Rectangle, Slot, Spline, Fillet, Chamfer, Text, Point]
            - name: "Modify"
              buttons: [Move, Copy, Rotate, Scale, Stretch, Trim, Extend, Split, Offset]
            - name: "Pattern"
              buttons: [Rectangular, Circular, Mirror]
            - name: "Constrain"
              buttons: [Dimension, Auto Dimension, Coincident, Collinear, Concentric, Fix, Parallel, Perpendicular, Horizontal, Vertical, Tangent, Smooth, Symmetric, Equal]
            - name: "Format"
              buttons: [Construction, Centerline]
            - name: "Exit"
              buttons: [Finish Sketch]

    - name: "Sheet Metal Environment (.ipt)"
      internal_name: "SheetMetal"
      description: "Specialized part environment for uniform thickness manufacturing"
      tabs:
        - name: "Sheet Metal"
          internal_name: "id_TabSheetMetal"
          panels:
            - name: "Setup"
              buttons: [Sheet Metal Defaults]
            - name: "Create"
              buttons: [Face, Flange, Contour Flange, Hem, Bend, Fold]
            - name: "Modify"
              buttons: [Cut, Corner Chamfer, Corner Round, Corner Seam, Punch Tool]
            - name: "Flat Pattern"
              buttons: [Go to Flat Pattern, Create Flat Pattern]

    - name: "Assembly Environment (.iam)"
      internal_name: "Assembly"
      description: "Multi-component assembly workspace"
      tabs:
        - name: "Assemble"
          internal_name: "id_TabAssemble"
          panels:
            - name: "Component"
              buttons: [Place from Content Center, Place, Create Component, Replace, Replace All]
            - name: "Position"
              buttons: [Move Component, Rotate Component, Grip Snap, Free Move, Free Rotate]
            - name: "Relationships"
              buttons: [Constrain, Joint, Assemble, Show, Hide All, Unground and Root]
            - name: "Pattern"
              buttons: [Pattern Component, Mirror Component, Copy Component]

        - name: "Design"
          internal_name: "id_TabDesign"
          panels:
            - name: "Fasten"
              buttons: [Bolted Connection, Pin, Joint, Clevis Pin]
            - name: "Frame"
              buttons: [Insert Frame, Change Frame, Miter, Trim/Extend, Notch, Lengthen/Shorten]
            - name: "Power Transmission"
              buttons: [Shaft, Gear, Bearing, Belt, Chain, O-Ring, Cam]

    - name: "Drawing Environment (.idw / .dwg)"
      internal_name: "Drawing"
      description: "2D manufacturing documentation workspace"
      tabs:
        - name: "Place Views"
          internal_name: "id_TabPlaceViews"
          panels:
            - name: "Create"
              buttons: [Base, Projected, Auxiliary, Section, Detail, Overlay, Draft View]
            - name: "Modify View"
              buttons: [Break, Break Out, Crop, Slice]

        - name: "Annotate"
          internal_name: "id_TabAnnotate"
          panels:
            - name: "Dimension"
              buttons: [Dimension, Baseline, Chain, Ordinate]
            - name: "Feature Notes"
              buttons: [Hole and Thread, Chamfer, Thread, Surface Texture, Welding Symbol]
            - name: "Table"
              buttons: [Parts List, General Table, Hole Table, Revision Table]

    - name: "Presentation Environment (.ipn)"
      internal_name: "Presentation"
      description: "Exploded views and animation tracking"
      tabs:
        - name: "Presentation"
          internal_name: "id_TabPresentation"
          panels:
            - name: "Create View"
              buttons: [Create View]
            - name: "Component"
              buttons: [Tweak Components, Clear Tweaks]
            - name: "Camera"
              buttons: [Capture Camera, Restore Camera]
            - name: "Publish"
              buttons: [Video, Raster Image]

    - name: "Global Active Context Utilities"
      internal_name: "GlobalUtilities"
      description: "Diagnostic and visibility tabs persistent across all open files"
      tabs:
        - name: "Tools"
          internal_name: "id_TabToolsGlobal"
          panels:
            - name: "Options"
              buttons: [Application Options, Document Settings]
            - name: "Content Center"
              buttons: [Editor, Favorites, Batch Publisher]
            - name: "Material and Appearance"
              buttons: [Materials, Appearances]
            - name: "Options 2"
              buttons: [Add-Ins, VBA Editor, Run Macro]

        - name: "Inspect"
          internal_name: "id_TabInspect"
          panels:
            - name: "Measure"
              buttons: [Measure, Region Properties]
            - name: "Interference"
              buttons: [Analyze Interference]

        - name: "View"
          internal_name: "id_TabView"
          panels:
            - name: "Appearance"
              buttons: [Visual Style, Shadow, Reflection, Orthographic, Perspective]
            - name: "Navigate"
              buttons: [Zoom, Pan, Orbit, Look At, ViewCube]
            - name: "Windows"
              buttons: [Switch Windows, Cascade, Tile Horizontally, Tile Vertically]
```

## Current deviations (to align — tracked 2026-06-04)

Audit of `app/commands_standard.go` against the tree above. These are placement bugs to
fix as the affected features are touched (the commands work; they're in the wrong panel):

| Command | Was | Correct panel (per tree) | Status |
|---------|-----|--------------------------|:------:|
| Sketch `Dimension` | `Dimension` | **`Constrain`** | ✅ fixed 2026-06-04 |
| Sketch `Auto Dimension` | `Dimension` | **`Constrain`** | ✅ fixed 2026-06-04 |
| Sketch `Mirror` | `Modify` | **`Pattern`** | ✅ fixed 2026-06-04 |
| Sketch `Fillet` | `Modify` | **`Create`** | ✅ fixed 2026-06-04 |
| Sketch `Project Geometry` | `Draw` | (not in this tree's Sketch tab; verify) | review |
| Part `Offset Face`/`Thicken` | `Modify` | `Modify` has **`Thicken/Offset`** (combined) | ok-ish |

The Sketch tab now matches the tree (panels: Create, Modify, Pattern, Constrain, Exit;
plus a non-canonical `Draw` panel holding Project Geometry — under review). Locked by
`app.TestSketchTabPanelsMatchInventor`. **Modify** = Move/Copy/Rotate/Scale/Stretch/Trim/
Extend/Split/Offset; **Pattern** = Rectangular/Circular/Mirror; **Create** = the full
canonical set incl. Slot/Fillet/Chamfer/Text (plus Ellipse/Polygon extras); **Constrain** =
Dimension/Auto Dimension/constraints; **Exit** = Finish (all built 2026-06-04). **The
Sketch 2D tab is at full button parity with the tree.** Parameterized tools share one
generic property dialog (`app.ParameterizedTool` → `head/ui/tool_params_dialog.go`).
Follow-up: Stretch's head interaction is vertex-window selection (the model/tool work on an
explicit point set today).

Notes:
- The **`Dimension` panel does not exist** in the Inventor Sketch tab — dimensions live in
  **`Constrain`**. The standalone `Dimension` panel we use must be merged into `Constrain`.
- Sketch **`Modify`** should be: Move, Copy, Rotate, Scale, Stretch, Trim, Extend, Split,
  Offset. We currently also have Mirror (→ Pattern) and Fillet (→ Create) there.
- Part **`Surface`** panel buttons are Patch, Stitch, Sculpt, Extend, Trim, Rule Fillet —
  when the M10 surfacing features get UI (PARTDOC-PLAN Phase 5), place them here, not in a
  new panel.
- Part **`Pattern`** panel (Rectangular, Circular, Sketch Driven, Mirror) is where the
  flagship pattern/mirror UI (PARTDOC-PLAN Phase 3, finding U-01) must land.
- Out-of-scope environments (SheetMetal/Assembly/Drawing/Presentation) are included for
  completeness; do not build them until PartDocument is complete (see PARTDOC-PLAN.md at
  the repo root).
