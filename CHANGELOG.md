# Changelog

## [Unreleased]

### Added

- Added native API execution for Home, Zoom All, Look At, and sketch commands shared with the marking menu. (#255)
- Added single and batch assembly placement by definition to the desktop API. A batch places copies of one component as one undoable edit. Added API commands to transform, ground, release, remove, and replace existing occurrences. (#728)
- Added separate document and library regions to the Material and Appearance Browsers, with search, category browsing, and independent display options. Added context-menu assignment of library materials to a selected part and displayed its inherited appearance in the ribbon. Added shaded appearance thumbnails, independent list and thumbnail views, search, and context-menu assignment of appearances. Added document-owned Generic appearances with a live editor preview, palette, Apply, and a Save Edits prompt. Added editable appearance descriptions and keywords that persist with the Part. Added independent Reflectivity disclosure and enable controls with editable Direct and Oblique values and a live preview. Added Generic Transparency with live transmission, translucency, and refraction previews and a saved custom index. Added independent Cutouts, Self Illumination, Bump, and Tint controls to the Generic editor. Added the Generic Image menu and a separate Checker texture editor with live previews and Part persistence. Added Sphere and Cube preview scenes and four lighting presets to the Appearance Editor. Added document-owned Ceramic appearances with editable type, finish, color, Wavy Finish Bumps, and Tint. Added PNG, JPEG, and BMP import for custom Ceramic Finish Bumps and Relief Pattern. Added Duplicate and Duplicate as Generic for applied document appearances. Added Concrete appearances with editable sealant, finish, weathering, and Tint. Added Metal appearances with alloy presets, finishes, relief, cutouts, and Tint. (#3563)
- Added the View ribbon Object Visibility checklist with independent category switches and the All Workfeatures control. Connected origin, user work plane, 2D sketch, and sketch dimension overlays to the checklist. (#3897)
- Enabled Open File Location for a saved Model Browser document, opening its containing folder in the system file manager. (#4116)
- Enabled Make New Folder in the project wizard and selected the created folder for workspace confirmation. Finish saved the new project with the chosen reference libraries. Enabled Browse to load a project, Apply to activate it and refresh the template chooser, and Save to commit edited project settings. (#3895)
- Added Home and End navigation to the staged Ribbon Customization list. Advanced the available command selection after a single transfer and kept Space on Add after a mouse edit to Large or Text. (#3856)
- Added named native templates from the active project's root, Metric, and English folders to the desktop chooser. The chooser shows Part materials, previews drawing sheets, and restores the selected template after restart. Populated drawing layouts accept a Part, warn when its previous evaluated geometry is used, and show its authored parts list. (#3899)
- Added Flipped, Symmetric, and Asymmetric directions and To Face and From–To extents for new solid extrusions in the desktop and typed API. Added typed API Cut and Join for a single current solid with persistent body identity, including associative Through All Cut and stable names for split topology. Added typed rectangular patterns, circular patterns, and feature mirrors of Join and Cut extrusions. Enabled typed extrusion from sketches on work planes offset from named faces. Enabled typed sketch rectangle creation and profile queries with analytic area and hole counts for extrusion selection. Added typed offsets for sketch lines, circles, arcs, connected line chains, and closed profile boundaries. Added desktop Icons and Dropdown direction selectors and a Distance A editor menu. (#3893)
- Expanded the desktop parameter tolerance editor with tolerance formats, precision, evaluated size, and ISO fit classes. (#3893)
- Added analytic solid-edge lengths to the Part Measure panel. (#3967)
- Added Zoom, Zoom All, Zoom Selected, and Zoom Window to ribbon customization. (#3856)
- Added Show Panels instructions, F1 help, and visibility toggles for Primitives, Simulation, and 3D Print on the 3D Model ribbon. Added the ten Part Manage panels and connected Update and iProperties to their existing actions. Added an Explore panel with Measure to the 3D Model ribbon and completed the side-docked 3D Model panel list and default check states. (#3856)
- Added the Standard overview and Units tab to Document Settings. Added the Exit 2D Sketch Tab category and source paging. Transfers after Ctrl+A put the focused command first. Up and Down select staged rows without reordering them. (#3856)
- Added a Tools Options expansion with a ribbon layout reset confirmation. Kept Shell and the solid visible when opening Customize. Applying an empty User Commands group now removes it immediately in ordinary ribbon editing. During Shell, the empty group stays visible until Customize closes. (#3856)
- Added a Highlight New drop-down that can open over Tools Options additional commands and close independently. (#3856)
- Added Assembly Design customization for the Assemble ribbon. Listed unavailable targets and kept a clean draft open when Escape dismissed its target selector. (#3856)
- Added a curvature comb and live density and scale settings for sketch ellipses. (#3995)
- Added a saved, opt-in set of default multi-character command aliases to Keyboard customization. Reset its search and filters when reopening Customize and copied the full shortcut table in display order. (#3856)
- Added Italic to the Sketch Text editor and kept Bold when applying it. (#3907)
- Included applied ribbon, Quick Access, keyboard, marking menu, and mini-toolbar state in Customize exports. (#3856)
- Opened the editor for existing Sketch Text on a double-click of the glyph. (#3907)
- Listed 2-D Drawing, 3D PDF, 3D PDF Export, and Z Parallel as unavailable commands that can be arranged in custom ribbon panels. (#3856)
- Added Bold preview and editing for a full Sketch Text selection. (#3907)
- Added stacked fractions to Format Text and preserved them in native sketches and named presets. (#3907)
- Added five line-spacing modes to Format Text and retained the selected spacing in native sketch files. (#3907)
- Added typed Sketch Text creation, editing, and queries for automation. (#3907)
- Added paragraph bullets, numbering, and diameter-symbol insertion to Format Text. (#3907)
- Preserved valid character-format ranges in native sketch files and sketch history. (#3907)
- Added a saved Mini-Toolbar visibility option to the View ribbon. (#3897)
- Add an exact best-fit certificate for explicit affine equations; sketch drag integration remains open. (#4171)
- Draw midpoint lines with pointer or typed length and angle, including explicit point selection. (#4178)
- Choose and select one relation when sketch constraint glyphs overlap. (#4169)
- Added individual sketch constraint glyph hiding and Show All through the viewport and typed API. (#4166)

- Enabled the Rust desktop application on Windows through a native Win32 window and Martensite Vulkan rendering.
- Enabled isolated Lua scripting on Windows in the desktop and CLI.

- Added analytic face properties from the selected-face context menu. (#4053)

- Show the type and highlight of a sketch entity under the pointer in Ready. (#4043)

- Show the picked sketch entity type and sketch name near the pointer. (#4042)

- Added Move and Measure pointer feedback for sketch curves, points, and midpoints. (#4034)

- Added transient candidate feedback for sketch constraint and dimension tools. (#4034)

- Added an experimental offline sandbox build probe for native BDD investigations. (#3978)

- Added an experimental committed-source exporter for native BDD build investigations. (#3978)

- Added a native scenario receipt command that checks its artifacts before publication. (#3865)

- Added a diagnostic mode to report native BDD step gaps after failures. (#3863)

- Added browser rows and an editor for saved rectangular sketch patterns.

- Added a rectangular sketch-pattern dialog with live preview and cancel.

- Added rectangular sketch-pattern creation through the typed API.

- Added previewable and undoable rectangular pattern edits that retain existing member identities.

- Added saved rectangular sketch-pattern records with preview and undoable creation.

- Added parameter-driven spacing for linked sketch copies and recompute.

- Linked translated sketch copies to their seed points and circle radii through solve and history.

- Added saved structural point links for associative sketch-pattern solving.

- Added rectangular sketch-pattern layouts with fitted spans and suppressed instances.

- Added atomic sketch-copy batches and previews as a foundation for patterns.

- Added rotated sketch-copy previews and undoable copy operations.

- Kept Finish Sketch available on document ribbon tabs while a sketch is being edited.

- Added context menus for unavailable commands in expanded ribbon panels.

- Added the optional Parameters panel to the Sketch ribbon.

- Added document unit labels to Precise Input coordinates.

- Added Copy mode to sketch Move, with preview and undo/redo.

- Added precise coordinates and triad controls to sketch Move.

- Added multi-entity selection and single-selection optimization to sketch Move.

- Added single-entity sketch Move with a solved preview, canvas points, and undo/redo.

- Added undoable geometric relation removal from sketch curve context menus.

- Added repeat of sketch constraints to the sketch context menu. (#3856)

- Added selection-aware Delete to the sketch context menu. (#3856)

- Added marking-menu command search and assignment to radial slots or overflow. (#3856)

- Added saved sketch marking-menu layouts and slot customization. (#3856)

- Added sketch-ready aliases for Undo, Redo, Save, Home, and Previous View. (#3856)

- Added saved keyboard shortcuts and sketch command aliases to the Customize dialog. (#3856)

- Added ribbon location information for the sketch visibility command. (#3856)

- Added the View Tab command category to ribbon customization. (#3856)

- Added keyboard switching between ribbon customization targets. (#3856)

- Added keyboard navigation and Escape handling to the ribbon command-category list. (#3856)

- Added the Macros category to ribbon customization for sessions with no macros. (#3856)

- Added keyboard navigation and scrolling for available ribbon commands. Page Up and Page Down select the visible edge before advancing to the next page. (#3856)

- Added name sorting to the available ribbon commands while preserving selection. (#3856)

- Added expanded-panel placement for custom ribbon commands. (#3856)

- Saved applied part ribbon customization in user preferences across restarts. (#3856)

- Added XML export for applied part ribbon customization and staged XML import. (#3856)

- Added Ctrl+A selection and a title-bar close control to ribbon customization. (#3856)

- Added keyboard selection and transfer of multiple ribbon commands. Secondary-click selection gives its list keyboard focus. Selected the next surviving staged row after removing a range. (#3856)

- Added sketch tools and constraints to the ribbon customization command list. (#3856)

- Added double-click transfer of ribbon user commands. (#3856)

- Added session-local customization of part ribbon user commands. (#3856)

- Added typed API placement of existing sketch block definitions with checked
  position, rotation, scale, and undo/redo. (#3856)

- Added viewport display for placed sketch block definitions, including their
  rotation, scale, and instance selection color. (#3856)

- Added saved sketch block instance identities and placements, with typed
  instance listing and protection against deleting a definition in use. (#3856)

- Added saved local geometry for sketch block definitions, including captured
  shared points and curve roles. (#3856)

- Added typed API operations to create, list, and delete empty sketch block
  definitions, with undo and save/reopen support. (#3856)

- Added the Sketch Create expansion with access to Precise Input while a sketch
  command accepts points. (#3856)

- Added the Sketch tab for part documents and its Constrain expansion. (#3856)

- Added immediate placement and value editing for sketch dimensions. Cancel keeps
  the measured dimension, and Undo treats placement and value edits as separate
  steps. (#3855)

- Added Left and Right navigation for feature branches in the model browser. (#3856)

- Added Up and Down navigation for visible rows in the model browser’s Features branch. (#3856)

- Added a Zoom Selected toolbar for selected model faces. (#3856)

- Added arrow-key view rotation while ribbon key tips are visible. (#3856)

- Added keyboard access to compact and side-docked ribbon groups for part documents. (#3856)

### Changed

- Made the automatic X11 UI size more compact on high-DPI displays and added an Automatic text-size choice. Added a native Wayland desktop host with X11 fallback on Linux. (#3850)
- Reduced desktop GUI render preparation costs and raised the ViewCube animation cadence. Linux release builds made Vulkan validation opt-in and added an X11 VSync override. (#3630)
- Updated the geometry kernel and Vulkan renderer to Martensite v0.14.2. (#3563)
- Updated the host and generated script help to Oblikovati.API v0.166.0. (#3965)
- Adopted the new desktop visual identity across Dark, Light, and High Contrast themes, including studio viewport palettes, bundled fonts, vector navigation and ribbon icons, and the Extrude panel. Added active asset dock indicators and swatches, increased ViewCube contrast, and applied neutral drawing paper. Added document tab artwork, descender clearance, a centered Documents control, and live viewport status readouts. Updated Home wordmarks, ribbon captions, density controls, and command artwork. Home opens at full width with its document docks suspended. Standardized Normal ribbon command sizes and aligned the Material and Appearance selectors. Added and recolored command glyphs in Sketch, Tools, View, and the Modify flyout. Extrude now fits its contents, toggles preview, and offers Apply & Create New. Updated Model Browser rows, moved options and search into its dock title, and embedded the clear control in the outlined search field. Added compact DOF badges, a stronger dashed rollback rule, Studio selection colors, and dimension units and extension lines. (#4063)
- Reused part checkpoints during Undo and Redo when a clean feature prefix remains. (#4079)
- Reused part checkpoints when deleting bodies through the typed API. (#4079)
- Reused part checkpoints when renaming features through the typed API. (#4079)
- Reused part checkpoints when the typed suppression command reports feature health. (#4079)
- Reused current part checkpoints when applying Feature Properties. (#4079)
- Avoided replaying a part when inspecting an independent sketch. (#4079)

- Reused current part results for the sketch dimension count. (#4079)

- Listed Rule Log and Rule Automation as disabled in the View menu. (#4185)

- Enabled existing sketch placement tools from the Part ribbon when the part has one planar sketch. (#3956)

- Used native ImGui dock tabs for workspace, material, and appearance panels and added tab bar controls for panel visibility and Model Browser filtering. Restored each document's selected tab in the Model Browser dock, including after leaving Home. (#4133)

Fillet and Chamfer validation reuse a current matching part prefix before commit. (#4079)

Stored prism diagnostics now use document-scoped complete replay. (#4079)

Model tree reads reuse current part results and refresh them after edits. (#4079)

- Preserved explicitly selected sketch points during placement previews and commits. (#4179)

- Reduced repeated constraint analysis during sketch dimension placement. (#4179)

- Kept Point placement at the cursor when sketch inference or auto-apply was disabled. (#4177)

- Kept inferred line directions temporary when constraint persistence is off. (#4160)

- Kept inferred placement snaps without saving their constraints when persistence is off. (#4160)

- Reduced sketch solving time and memory when many coordinates have no constraints. (#4165)

- Removed the partial certified-prism whole-part evaluator. (#4079)

- Removed whole-part replay from the partial named-prism evaluators. (#4079)

- Required part evaluators to explicitly resolve or decline named planar faces. (#4079)

- Made Fillet and Chamfer dispatch explicit in the part evaluator. (#4079)

- Reused current source part results when saving drawing view snapshots. (#4079)

- Reused current part results for drawing view previews and display when their modelling policy matches. (#4079)

- Reused the current source part result when creating a drawing base view. Enabled typed API listing, rotation, caption editing, and deletion of base views, sheet lifecycle, title-block fields and corner, sheet revision, zoned borders, reusable sheet formats, ISO/ANSI drafting styles, drawing notes with optional leaders, custom tables, revision annotations, and datum feature symbols. (#4079)

- Reused current part results when typed Fillet and Chamfer commands resolve selected edges. (#4079)

- Retained current part replay results per open document in the desktop. (#4079)

- Reused unchanged sketch replay results during part updates to avoid a second full replay. (#4079)

- Separated selected-face context commands from the radial marking menu. (#4055)

- Enabled analytic area measurement from a selected face context menu. (#4052)

- Enabled Chamfer from the default Part marking menu. (#4046)

- Kept selected input edges visible during a Fillet preview. (#4045)

- Allowed the Fillet tool to start before an edge is selected. (#4045)

- Enabled Fillet radius editing from the model browser with preview and cancel. (#4045)

- Enabled certified Fillet radius edits through the feature API, including named parameters and undo/redo. (#4045)

- Reduced repeated projection of constraint glyphs in an active sketch. (#4035)

- Enabled named-edge Fillet preview and creation in the Part viewport, with undo, redo, and save/reopen support. (#4045)

- Refresh camera depth when an edited model would be clipped. (#4044)

- Preserve body-relative face orientation in prism presentation. Added typed viewport controls for inspecting triangle winding and coloring indexed faces or triangles. (#4044)

- Reduced repeated sketch point projection work during pointer interaction and context picking. (#4035)

- Discarded closed document-window preview images and stale pending updates. (#3987)

- Added type-preserving integer, number, boolean, and byte property editing to iProperties. (#3958)

- Added an optional locked-registry comparison to the offline BDD build probe. (#3978)

- Added an optional committed-source comparison to the offline BDD build probe. (#3978)

- Enabled viewport selection of visible work planes for sketch and offset commands. (#3984)

- Displayed visible authored work planes with a translucent fill and themed border in part views. (#3981)

- Enabled creation of offset work planes from the 3D Model ribbon, with document-unit validation and undo/redo. (#3977)
- Enabled analytic length measurements for named solid edges through the Analysis API. (#3967)

- Saved assemblies can include a qualified display cache; save warnings explain when it cannot be created. Express loading remains unavailable. (#3949)

- Exposed a shared assembly presentation traversal with explicit selection styling for the live scene; Express loading remains unavailable. (#3949)

- Extended staged legacy part replay to face-offset planes with independently witnessed side lineage; production precision qualification remains required. (#3882, #3928)

- Added a shared resolver that matches witnessed Go profile boundaries to current certified regions, with precision qualification required. Automatic Go profile witnessing remains open. (#3882, #3928)

- Added conservative physical-unit labels to raw sketch residual rows, with unknown units retained where the equation is ambiguous. These labels do not certify spatial or angular bounds. (#3928)

- Exposed source-attributed raw sketch residual rows and their convergence domains in solver diagnostics; these values are not certified spatial bounds. (#3928)

- Split sketch frame operations into `HeadSketchWork`; custom `HeadWork` owners must implement that interface. (#3860)

### Fixed

- Rendered Windows windows at the monitor DPI and scaled layout, input, and fonts when display density changes. Linux X11 uses desktop DPI from X resources or XSETTINGS for the same layout, input, and font scaling. (#3850)
- Made File Properties dockable from Open and restored the picker and its selection on close. (#4073) Fixed a thumbnail crash in large file lists, kept Details headings visible while scrolling, and kept long filenames out of the date column. (#3889)
- Matched the default viewport and status bar to the desktop theme while preserving explicit colour schemes. Gave status telemetry compact monospace text and highlighted Fully Constrained in the active theme’s success color. Projected ViewCube labels onto their faces, added directional shading and hover highlights, and added fading controls, roll arcs, and an options chevron. Strengthened foreshortened face lettering in both desktop themes. Centered the navigation bar under the cube and moved it beside the cube when the viewport is too short to stack them. Showed the viewport controls in Assembly. Kept the Orientation Options labels readable and its actions visible while scrolling at large text sizes. Restored padding around Quick Access selection commands, used the strong theme outline for asset swatches, and kept asset captions left-aligned and values clear of dropdown arrows during resize. Kept inactive dock tab titles legible in Dark and Light. (#4063)
- Kept Extrude actions visible in short panels and made feature drafts exclusive. Fitted origin picker planes to the model bounds, projected their labels onto the planes, faded edge-on labels, and kept the plane overlay behind the viewport controls. Fitted and labelled the Sketch ribbon groups in a stable 88 px Normal ribbon row and kept Light ViewCube labels readable when inactive. Kept origin plane labels visible when zoom crops every plane corner and gave them a theme-colored backing over model faces. Calibrated the picker’s translucent fill for the viewport renderer. (#4063)
- Updated the active ribbon underline in the same frame as its commands. A title-bar drag now sends one native window move request per press. The first click after a Wayland resize activates the frame control. (#4063)
- Aligned and outlined Fillet, Chamfer, Hole, Offset Plane, Move Bodies, and Shell inputs and fitted their panels to the property rows and footer. Kept the Hole depth label visible and showed Through all in the empty field. Fitted Offset Plane to its controls and wrapped validation messages. Gave Extrude a 10.00 mm default, outlined its numeric inputs, and gave its four direction buttons equal width. Made From and Termination dropdowns and aligned their reference selectors. Kept floating modeling properties inside the viewport during resize and improved Light theme contrast for disabled property actions and read-only Theme Editor values. (#4063)
- Centered the active document and model state in the titlebar, separated the brand mark from Quick Access, and restyled the window controls. Labelled compact ribbon groups across all tabs, aligned the 28 px tab strip, and fitted expanded command rows to the flyout width. Matched the Tools Options and Sketch Create flyouts to the icon and row spacing of the other panels. (#4063)
- Restyled the File and Home recent lists with type icons, dedicated pin space, and more room for filenames. Gave Home Recent aligned 14 px checkboxes, a shared Search and Clear field, scroll-end padding above the document tabs, and one date format for both timestamp columns at each width. Gave the File menu matching footer rows, an accent on the open tab and current Recent controls, and a popup shadow. Gave the default Model browser about a quarter of the workspace when space allowed. Preserved viewport space, active document tabs, and side dock minimums during resize. Enforced the viewport width limit during splitter drags, recentered the projection in the resized document area, and kept the chosen dock widths when reopening the app. Labelled compact 3D Model groups with icons and fitted all default groups across the minimum-width ribbon. Kept Get Started and Tools visible while Home covered an open Part, suspended its docks, and cleared its title and viewport telemetry until the Part returned. (#4063)
- Kept Work Plane and Sketch dropdown arrows inside their command cells, filled the ribbon height with single-command buttons, and placed selector labels to the left of their controls. Aligned the Large and Text headings with their Ribbon Customization checkboxes. The full-width Name header sorts available commands, returns to the top, and keeps the selected command. Fitted small icon-only custom commands and their separators into the ribbon without reserving space for hidden labels. (#3856)
- Kept long tab labels readable when the ribbon is docked on either side. Used the full height of the floating ribbon for its scrollable command groups. (#3860)
- Allowed the only attached drawing to open in its own window, with Home in the main window. (#3900)
- Wrapped long project names in the Projects dialog. (#3899)
- Restored the template gallery's left edge when keyboard navigation reaches the first tile. Kept the saved template selection when the startup project loads after Home. (#3899)
- Enabled Page Up and Page Down in the staged ribbon command list. (#3856)
- Moved keyboard focus to Add when pressing Tab from a selected available ribbon command. (#3856)
- Drew the 3D Annotations icon at its configured size in custom ribbon panels. (#3856)
- Enabled Offset from Plane in the Plane menu and as a separate ungrouped command. Added menu icons and kept Work Features compact after regrouping until the ribbon width changes. (#3856)
- Resized both Customize command lists with the dialog and restored its default size on reopening. (#3856)
- Added a saved Standard or Large button size for Shell in the Modify panel. (#3856)
- Added Shell to the commands that can be pinned to Quick Access. (#3856)
- Added Part, 2D Sketch, and 3D Sketch profiles plus staged style, overflow, and classic context preferences to Marking Menu customization. Matched the default Part slots to the published profile and kept Chamfer in its overflow. (#3856)
- Kept the entire Add Separator label visible in the Quick Access context menu and displayed icons for the Extrude and Print shortcuts. (#3955)
- Preserved available ribbon command selections when switching targets and selected the separator when a source filter hid the selected command. (#3856)
- Returned to Ribbon Customization with an error when importing invalid XML, preserving staged edits. (#3856)
- Selected command rows on secondary click in both Ribbon Customization lists. (#3856)
- Stored literal Extrude distances as built-in feature parameters so the model-state table could edit them per member. (#4119)
- Selected the clicked face when a viewport right-click dismisses the ribbon panel selector, and showed its mini toolbar. (#3856)
- Kept the unpinned Modify title strip available while its panel is hidden in the direct selector, and fixed closing the selector with its own button. (#3856)

- Kept the active ribbon tab selected when opening the context menu on an inactive tab. (#3856)

- Kept clicks on Home, Customize, Options, and the ribbon display menu working while the panel selector is open. Made the ribbon display control switch between full and tabs-only display in one click while the selector is open. (#3856)

- Kept the Sketch Create title's Ribbon Appearance submenu open when Escape is pressed. (#3856)

- Completed the floating Part ribbon group list and closed the Notes pane when switching tabs. (#3856)

- Kept Finish Sketch directly visible in the Exit group when the ribbon is docked at a side. (#3856)

- Highlighted active groups on side and floating ribbons and listed Slot, Draft, and Thread as disabled in Modify. (#3856)

- Kept the desktop responsive when a resize clips a dock tab bar. (#3860)
- Kept the Cargo workspace and repository checks running when the BDD ledger failed, and reported all failed stages. (#3860)
- Canceled superseded Rust CI runs on the same branch or pull request so the latest checks can start sooner. (#3860)

- Kept long names and paths inside their columns and reserved space for row actions in Home Recent Documents. (#3898)

- Opened Home with a New Part action on no-document startup and docked the asset browsers for new profiles. (#3898)

- Wrapped recent-document cards to fit in Home when the asset dock reduces the workspace width. (#3898)

- Kept Home Recent sort headings and direction marks visible beside the asset dock. (#3898)

- Showed complete calendar dates in compact Home Recent columns and full timestamps on hover. (#3898)

- Restored Ready after closing the Rotate at Angle dialog. (#3872)

- Restored installed Tahoma and the inherited color swatch when starting Sketch Text without a preset. (#3907)

- Kept a modified Sketch Text linked to its named preset when reopening the editor or changing the preset name. (#3907)

- Restored the panel-add control after the last browser tab closes and activated the tab selected from its menu. (#3897)
- Kept Add to Custom Colors above the scrollbar when the expanded Geometry Properties color picker fits the display. (#3896)
- Animated ViewCube turns when the animated transitions option is enabled. Kept a single hover highlight across every visible side of a shared edge or corner. Added typed camera reads and writes for the active or addressed document window. Fixed argument-free camera reads from the typed Rust client. (#3961)

- Restored Ready after closing the navigation wheel. (#3881)

- Refreshed dock tabs when workspace panels open or close. (#4133)

- Keep sketch dimension labels readable and select the dimension under the pointer. (#4180)

- Asked before creating a redundant sketch dimension when the warning preference is selected. (#4163)

- Honored the edit-on-creation preference for new sketch dimensions. (#4167)

- Refused named-prism queries when a later blend replaces the requested body. (#4086)

- Preserved exact curve bounds when sampling saved drawing views. (#4085)

- Resolve Fillet and Chamfer inputs with the certified policy when the retained part uses different bounds. (#4084)

- Avoided computing the assembly producer identity for ordinary document commands. (#4081)

- Allow Fillet and Chamfer creation after model-state changes and reopening a part. (#4080)

- Preserved dock tabs when tool windows close, suspend, or reopen. Restored input for docked Projects and returned to the selected template on close. Fixed a docking crash when switching rapidly between Part and Drawing. (#4073)

- Replayed fillets and chamfers consistently in drawing, feature health, and extrusion preview. (#4074)

- Enabled Link Parameters cancellation without a source and restored the next dock tab on close. (#4073)

- Kept the remaining dock tab active when Precise Input closes from its menu. (#4073)

- Kept the remaining dock tab active when Create Block closes or completes. (#4073)

- Kept the remaining dock tab active when Navigation Wheel Options closes. (#4073)

- Rejected a selected-face Measure action when its face or document changed. (#4052)

- Started an associative sketch on the selected face from the Part marking menu. (#4050)

- Keep hover feedback in sync with a sketch changed before the next frame. (#4051)

- Restored Extrude from the Part marking menu for empty parts and finished sketches. (#4049)

- Listed and selected overlapping sketch point and curve candidates in Select Other. (#4041)

- Refused preference saves when YAML keys have ambiguous merge meaning, preserving the existing file. (#4007)

- Preserved document and Home tab order when scrolling and closing windows. (#3993)

- Kept off-screen document selection from highlighting a different visible tab. (#3992)

- Kept document tab previews clear of the tab at large UI scales and near window edges. Used a higher-resolution filtered thumbnail and displayed Windows drive and network paths without internal prefixes. (#3988, #3900)

- Cleared selected body and face appearance overrides together in one undoable edit. (#3962)

- Cleared multiple selected named face appearances in one undoable edit. (#3962)

- Cleared a selected named face appearance with Cancel and undo support, while keeping other overrides. (#3962)

- Kept Ctrl and Shift viewport body selections for Clear Appearance. (#3962)

- Kept the rectangle placement prompt while a zero-size preview is incomplete. (#3975)

- Excluded isolated task worktrees from source and documentation checks while retaining active nested modules. (#3971)

- Reopened existing extrusions for distance editing from the browser, with a cancellable candidate preview. (#3968)

- Edited extrusion distances without changing shared sketch parameters; literal distances can also be edited. (#3968)

- Kept sketch ribbon controls and the previous part tab available across sketch editing. (#3956)

- Refused legacy cap-program staging when qualified sketch precision evidence is unavailable. (#3928)

- Rejected ambiguous or forward legacy sketch and extrusion creation order before preparing a part candidate. (#3936)

- Decoded legacy work-plane face references with Go's unpadded URL-safe base64 format. (#3935)

- Showed the actual Part template material state, including unassigned and unresolved identities, in the chooser. (#3932)

- Reopened the Extrude tool with its configurable shortcut after cancelling a draft. (#3931)

- Refreshed extrusion previews after valid draft or parameter changes without creating features. (#3930)

- Validated extrusion distance drafts with the document owner, and retained rejected drafts for correction. (#3929)

- Resumed Extrude after its requested sketch finishes, with a 10 mm preview for a converged closed sketch, including a sketch with free variables. Sketch solve precision qualification remains open. (#3927, #3928)

- Cleared the save-path request after opening Save As, so cancellation leaves no stale error message. (#3888)

- Preserved the conversion guard of built-in model parameters when saving and reopening. (#3883)

- Made sketch creation and Finish Sketch separate undo and redo steps. (#3880)

- Started a sketch directly on a preselected origin plane. (#3879)

- Kept the ViewCube visible and interactive while editing a 2D sketch. (#3877)

- Enabled analytic area measurements for named solid faces through the Analysis API. (#3876)

- Kept toolbar arrow-key navigation from rotating the model on its first key press. (#3874)

- Lit model faces in the viewport and isometric preview so their orientations are visible. (#3873)

- Kept the model inside the document area when using Home or Fit with docked panels. (#3872)

- Selected a ribbon tab on the click that dismisses its context menu.

- Closed unpinned ribbon expansions when Escape dismisses their title menu.

- Kept expanded ribbon commands open while their docking menu is in use. Closed unpinned expansions after panel visibility changes from their title menu. (#3856)

- Kept the ribbon docking submenu closed after Escape until it is opened again.
- Made the default ribbon display control toggle between full and tabs-only modes.

- Made Precise Input show coordinates and the triad for the selected reference mode.

- Resolved API dimension Drive expressions once, as in the Go host, while keeping desktop formulas live.

- Kept selected sketch geometry at the requested transform, and refused moves blocked by constraints.

- Kept geometric constraint tools active for repeated selections and restored Coincident prompts.

- Preserved sketch selection when opening a point context menu. (#3856)

- Disabled sketch marking-menu commands while a transaction is open. (#3856)

- Made Escape from the ribbon target selector discard the pending customization. (#3856)

- Kept origin-plane labels inside the clickable workspace when starting a sketch. (#3857)

- Included User Commands in Show Panels in the ribbon context menu and limited the Part Tools checklist to built-in panels. (#3856)

- Added a missing-file warning and filename recovery to ribbon settings import. (#3856)

- Fixed ribbon customization to keep display options per command and restore them after a staged removal. (#3856)

- Kept reversed ribbon customization edits pending until Apply or Cancel. (#3856)

- Fixed Escape in ribbon customization to discard pending edits without cancelling the active sketch. (#3856)

- Show the model's reason when a sketch dimension value is refused. (#3855)
- Reject driving dimension values that fail the sketch constraint's target checks, including a zero radius, before saving the edit. (#3855)

- Resolved viewport body colors from document and catalog appearances. (#3859)
