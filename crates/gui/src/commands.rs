// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::wire::MarkingMenuView;
use oblikovati_model::marking_menu::MarkingMenuDisplay;
#[derive(Clone, Debug, PartialEq)]
pub enum GuiShellAction {
    OpenTeamWebInBrowser(&'static str),
    OpenDocumentFolder(u64),
    SetMarkingMenu(oblikovati_api::wire::SetMarkingMenuArgs),
    SetMarkingMenuPreferences(Vec<MarkingMenuView>, MarkingMenuDisplay),
    SetKeymap(oblikovati_model::keymap::KeymapPreferences),
    PlaceSketchBlock(String),
    /// Enter or leave the Create Block geometry selection mode.
    SelectBlockGeometry(bool),
    /// Open the active sketch's scope draft.
    OpenInferenceScope,
    /// Restore or clear the scope draft's visible selection.
    SelectInferenceGeometry(Option<Vec<u64>>),
    /// Apply the accepted scope through the typed host API.
    SetInferenceScope(oblikovati_api::wire::SketchInferenceScopeView),
    /// Group the editor's current checked selection under this name.
    CreateSketchBlock(String),
    /// Replace one placed block with its independent sketch members.
    ExplodeSketchBlock {
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        instance: oblikovati_model::sketch::SketchEntityId,
    },
    /// Explode the placed block hit by the current sketch context menu.
    ExplodeSketchContext {
        token: u64,
        instance: oblikovati_model::sketch::SketchEntityId,
    },
    SetDocumentUnits {
        document: u64,
        request: oblikovati_api::wire::SetDocumentUnitsArgs,
    },
    CreateDrawingBaseView(GuiBaseViewRequest),
    /// Place an open component in the active assembly.
    PlaceComponent {
        assembly: u64,
        request: oblikovati_api::wire::PlaceOccurrenceArgs,
    },
    /// Open a component file in the background and place it.
    PlaceComponentFile {
        assembly: u64,
        path: String,
        name: String,
    },
    /// Show or clear a transient open-component placement preview.
    PreviewComponent(Option<(u64, u64)>),
    /// Show a transient preview of a component file.
    PreviewComponentFile {
        assembly: u64,
        path: String,
    },
    PreviewDrawingBaseView(Option<GuiBaseViewRequest>),
    SetUiScale(oblikovati_model::ui_scale::UiScaleOptions),
    SetPromptFrequency {
        prompt: oblikovati_model::application_prompts::ApplicationPrompt,
        frequency: oblikovati_model::application_prompts::PromptFrequency,
    },
    SetSketchOptions(oblikovati_model::application_options::SketchApplicationOptions),
    /// Accept the edited constraint settings for a specific document.
    SetDocumentSketchSettings {
        request_id: u64,
        document: u64,
        expected: oblikovati_api::types::SketchSettings,
        settings: oblikovati_api::types::SketchSettings,
    },
    /// Hide or restore surrounding model bodies while a sketch is edited.
    SetSketchIsolation(bool),
    SetOverlayVisibility(crate::GuiOverlayCategory, bool),
    SetPartOptions(oblikovati_model::application_options::PartApplicationOptions),
    SketchMove(crate::GuiSketchMoveAction),
    SketchCurvature(crate::GuiCurvatureAction),
    SketchText(crate::GuiSketchTextAction),
    PreviewSketchPattern(Option<oblikovati_model::history::SketchRectangularPatternRequest>),
    CreateSketchPattern(oblikovati_model::history::SketchRectangularPatternRequest),
    OpenSketchPattern {
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        pattern: oblikovati_model::sketch::SketchEntityId,
    },
    PreviewSketchPatternEdit {
        pattern: oblikovati_model::sketch::SketchEntityId,
        request: oblikovati_model::history::SketchRectangularPatternRequest,
    },
    EditSketchPattern {
        pattern: oblikovati_model::sketch::SketchEntityId,
        request: oblikovati_model::history::SketchRectangularPatternRequest,
    },
    LineCommand(crate::GuiLineCommand),
    PrecisePoint(crate::GuiPrecisePoint),
    PreciseTriad(crate::GuiTriadCommand),
    /// Turn new points into center points, or back.
    ToggleCenterPointFormat,
    /// Change the role of the point the canvas menu was opened on.
    SetCenterPoint {
        point: u64,
        center: bool,
    },
    /// Delete the stable relation selected in the sketch context menu.
    DeleteSketchRelation(u64),
    /// Select one checked entity from the current sketch context menu.
    SelectSketchContextCandidate {
        token: u64,
        entity: oblikovati_model::sketch::SketchEntityId,
    },
    /// Select one relation offered by the current glyph context.
    SelectContextConstraintGlyph {
        token: u64,
        glyph: oblikovati_model::sketch::SketchConstraintId,
    },
    /// Hide the constraint glyph captured by this context menu.
    HideContextConstraintGlyph {
        token: u64,
        glyph: oblikovati_model::sketch::SketchConstraintId,
    },
    /// Set the global constraint glyph switch from this context menu.
    SetContextConstraintGlyphs {
        token: u64,
        shown: bool,
    },
    /// Retire a dismissed sketch context menu.
    DismissSketchContext {
        token: u64,
    },
    Inspect(crate::GuiInspectAction),
    EditDimensionValue(crate::GuiDimensionValue),
    SetTheme(String),
    EditThemeColor {
        name: String,
        token: oblikovati_api::types::ThemeToken,
        value: oblikovati_api::types::Rgba,
    },
    DuplicateTheme {
        base: String,
        name: String,
    },
    DeleteTheme {
        name: String,
    },
    SaveTheme {
        name: String,
    },
    SetColorScheme(oblikovati_api::wire::SetColorSchemeArgs),
    ActivateDocument(oblikovati_api::wire::ActivateDocumentArgs),
    CloseDocument(oblikovati_api::wire::CloseDocumentArgs),
    OpenDocument(oblikovati_api::wire::OpenDocumentArgs),
    /// Load the components of an express assembly.
    LoadFullDocument(u64),
    /// Open a drawing with its saved views and deferred updates.
    OpenDeferredDrawing(String),
    SaveDocumentAs(oblikovati_api::wire::SaveDocumentAsArgs),
    SelectBrowserFeature {
        document: u64,
        feature: oblikovati_model::feature::FeatureId,
    },
    /// Change a feature's name and when it is suppressed; a saved, undoable change.
    EditFeature {
        document: u64,
        edit: crate::GuiFeatureEdit,
    },
    /// Change how a sketch's lines are drawn; a saved, undoable change.
    SetSketchLineStyle {
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        style: GuiSketchLineStyle,
    },
    /// Choose a whole row of the browser, or change what it shows.
    BrowserChoice {
        document: u64,
        choice: GuiBrowserChoice,
    },
    UpdateBodyProperties {
        document: u64,
        body: i64,
    },
    /// Assign a catalog asset to the active part.
    AssignPartAsset {
        document: u64,
        kind: GuiPartAssetKind,
        id: String,
    },
    EditDocumentAppearance(crate::GuiAppearanceEdit),
    /// Open the clear tool for the active document.
    OpenClearBodyAppearance {
        document: u64,
    },
    /// Clear the exact currently selected face overrides.
    ClearSelectedFaceAppearance {
        document: u64,
        keys: Vec<String>,
    },
    /// Clear selected body and face overrides together.
    ClearSelectedMixedAppearance {
        document: u64,
        body_keys: Vec<String>,
        face_keys: Vec<String>,
    },
    /// Clear the selected solids' own appearance overrides.
    ClearSelectedBodyAppearance {
        document: u64,
        keys: Vec<String>,
    },
    SetBodyAppearance {
        document: u64,
        key: String,
        appearance: String,
    },
    /// Turn the view as the orientation control asks.
    OrientView {
        command: crate::GuiOrientCommand,
        /// The options ask to fit the model to the new view.
        fit: bool,
        /// The options ask the desktop to display intermediate camera frames.
        animate: bool,
    },
    SetViewProjection(crate::GuiViewProjection),
    /// Persist the orientation control selection pivot preference.
    SetOrientationSelectionLock(bool),
    /// The navigation bar's commands on the view and the status area.
    ViewCommand(crate::GuiViewCommand),
    /// Read a closed file's stored properties, for the Open dialog's panes.
    InspectFile(String),
    /// What this document's next save stores as its picture; `None` stores none.
    KeepDocumentPicture {
        document: u64,
        picture: Option<GuiPreviewImage>,
    },
    /// A click on the view, off every panel: it chooses the face there, or nothing.
    ClickModel {
        document: u64,
        pixel: [i32; 2],
        toggle: bool,
        extrude_profile: Option<GuiExtrudeProfilePick>,
    },
    /// Select the target of a context menu without clearing selection on empty space.
    ContextClickModel {
        document: u64,
        pixel: [i32; 2],
    },
    SelectBrowserSketch {
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        toggle: bool,
    },
    SelectBrowserBlock {
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        instance: oblikovati_model::sketch::SketchEntityId,
        toggle: bool,
    },
    SetSketchVisibility {
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        visible: bool,
    },
    EditSketch {
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
    },
    Parameter {
        mutation: GuiParameterMutation,
        request: oblikovati_api::wire::ParameterSetArgs,
    },
    /// Set a typed document property through the document owner.
    SetDocumentProperty(oblikovati_api::wire::SetPropertyArgs),
    /// Open the metadata editor of the active document.
    OpenIProperties,
    /// Open the active part model-state property table.
    OpenModelStateTable,
    /// Check a model-state table draft without changing the document.
    VerifyModelStateTable {
        document: u64,
        request: oblikovati_api::wire::SetModelStateTableArgs,
    },
    /// Apply one model-state table draft through the document owner.
    SetModelStateTable {
        document: u64,
        request_id: u64,
        request: oblikovati_api::wire::SetModelStateTableArgs,
    },
    NewPart,
    CreateDocument(crate::GuiDocumentTemplate),
    CreateProjectTemplate(crate::GuiProjectTemplate),
    /// Load or create a project from the Projects dialog.
    ProjectFile(GuiProjectAction),
    DrawingLayout(GuiDrawingLayoutAction),
    /// Picture the active window for the Save As preview; the file is not changed.
    CaptureWindowPreview,
    /// Render the active part from an isometric view for Save As.
    CaptureIsometricPreview {
        document: u64,
    },
    /// Close the application through the window's own close policy.
    Exit,
    /// Show the Home workspace in front of the active document.
    ShowHome,
    Recent(crate::GuiRecentAction),
    ActivateWindow(u64),
    NewDocumentWindow,
    CloseDocumentWindow(u64),
    /// Show the origin planes; the sketch starts when one is chosen.
    ChooseSketchPlane,
    /// Choose a sketch plane for a waiting extrusion.
    ChooseSketchPlaneForExtrude,
    /// Measure the exact selected named face.
    MeasureSelectedFace {
        document: u64,
        key: String,
    },
    /// Open analytic properties for the exact selected named face.
    OpenFaceProperties {
        document: u64,
        key: String,
    },
    /// Close the face properties window.
    CloseFaceProperties,
    /// Start an associative sketch on the one currently selected named face.
    StartSketchOnSelectedFace {
        document: u64,
        key: String,
    },
    StartSketchOn(crate::GuiOriginPlaneKind),
    /// Start on a selected authored work plane.
    StartSketchOnWorkPlane(oblikovati_model::feature::FeatureId),
    /// Open the offset-plane editor for the active part.
    OpenOffsetPlane,
    /// Start or advance the interactive Work Axis command.
    WorkAxis(crate::GuiWorkAxisAction),
    /// Validate an offset expression against the active document.
    ValidateOffsetPlane {
        document: u64,
        offset: String,
    },
    /// Create an offset plane through the document owner.
    CreateOffsetPlane {
        document: u64,
        base: GuiOffsetPlaneBase,
        offset: String,
    },
    /// Choose a solid face under the pointer, with an origin-plane fallback.
    StartSketchAt {
        document: u64,
        pixel: [i32; 2],
        origin_plane: Option<crate::GuiOriginPlaneKind>,
    },
    FinishSketch,
    OpenExtrude,
    Revolve(crate::GuiRevolveAction),
    Sweep(crate::GuiSweepAction),
    Loft(crate::GuiLoftAction),
    Coil(crate::GuiCoilAction),
    Emboss(crate::GuiEmbossAction),
    Derive(crate::GuiDeriveAction),
    Rib(crate::GuiRibAction),
    Decal(crate::GuiDecalAction),
    Import(crate::GuiImportAction),
    Unwrap(crate::GuiUnwrapAction),
    OpenFillet,
    OpenMoveBody,
    PreviewMoveBody(Option<GuiMoveBodyRequest>),
    CreateMoveBody(GuiMoveBodyRequest),
    /// Open, pick, preview, or commit Part Modify features (Hole, Slot, Combine, Draft, Thread, Delete Face, Thicken/Offset, Direct Edit, Split, Mark).
    Hole(crate::GuiHoleAction),
    Slot(crate::GuiSlotAction),
    Combine(crate::GuiCombineAction),
    Draft(crate::GuiDraftAction),
    Thread(crate::GuiThreadAction),
    DeleteFace(crate::GuiDeleteFaceAction),
    ThickenOffset(crate::GuiThickenOffsetAction),
    DirectEdit(crate::GuiDirectEditAction),
    Split(crate::GuiSplitAction),
    Mark(crate::GuiMarkAction),
    Finish(crate::GuiFinishAction),
    Ucs(crate::GuiUcsAction),
    WorkPoint(crate::GuiWorkPointAction),
    /// Selection mode, preview, or commit for Part Modify Fillet v2.
    FilletV2(crate::GuiFilletV2Action),
    ChamferV2(crate::GuiChamferV2Action),
    ShellV2(crate::GuiShellV2Action),
    /// Open the Shell properties draft.
    OpenShellFeature,
    /// Reopen an existing Shell feature in the Shell Properties panel.
    OpenShellEdit(oblikovati_model::feature::FeatureId),
    /// Show or clear the exact Shell preview.
    PreviewShellFeature(Option<GuiShellFeatureRequest>),
    /// Commit the Shell draft.
    CreateShellFeature(GuiShellFeatureRequest),
    /// Open an equal-distance chamfer draft for selected edges.
    OpenChamfer,
    OpenFilletEdit(oblikovati_model::feature::FeatureId),
    OpenChamferEdit(oblikovati_model::feature::FeatureId),
    PreviewFilletEdit(GuiFilletEditRequest),
    EditFilletRadius(GuiFilletEditRequest),
    PreviewChamferEdit(GuiFilletEditRequest),
    EditChamferDistance(GuiFilletEditRequest),
    /// Orient the view to the selected planar reference.
    LookAtSelection,
    /// Show or clear a private fillet candidate.
    PreviewFillet(Option<GuiFilletRequest>),
    /// Show or clear a private chamfer candidate.
    PreviewChamfer(Option<GuiFilletRequest>),
    /// Commit a fillet after validating its selected edges and radius.
    CreateFillet(GuiFilletRequest),
    /// Commit a chamfer after validating its selected edges and distance.
    CreateChamfer(GuiFilletRequest),
    /// Reopen an existing extrusion's distance draft.
    OpenExtrudeEdit(oblikovati_model::feature::FeatureId),
    /// Commit an existing extrusion's distance through the typed lifecycle operation.
    EditExtrudeDistance(GuiExtrudeEditRequest),
    /// Preview an existing extrusion on a private candidate.
    PreviewExtrudeEdit(GuiExtrudeEditRequest),
    /// Commit the chosen sketch profile through the document owner.
    Extrude {
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        profile: i64,
        distance: String,
        second_distance: Option<String>,
        direction: crate::GuiExtrudeDirection,
    },
    /// Show or clear a transient extrusion preview.
    PreviewExtrude(Option<GuiExtrudePreview>),
    /// Accept a certified extrusion draft and its optional new body name.
    CommitExtrudeDraft(Box<GuiExtrudePreview>, Option<String>, bool),
    /// Ask the document owner to validate the current extrusion expression.
    ValidateExtrudeDistance {
        document: u64,
        distance: String,
        second_distance: Option<String>,
    },
    DeleteSketchSelection,
    ToggleSketchFormat(oblikovati_model::sketch::SketchCurveFormat),
    PlaceSketchTool(crate::GuiSketchTool),
    ConstrainSketch(crate::GuiSketchConstraint),
    Dimension(crate::GuiDimensionAction),
    Save,
    /// Recompute the active part and report failed features.
    UpdatePart,
    /// Set the ambient selection priority for viewport picks.
    SetSelectionMode(GuiSelectionMode),
    /// Show the full selection filter.
    OpenSelectionFilter,
    /// Show the complete transaction timelines.
    OpenHistoryBrowser,
    /// Move a document's transaction cursor.
    JumpHistoryTo {
        document: u64,
        position: i64,
    },
    /// Change one ambient pick category.
    SetSelectionEnabled {
        kind: crate::GuiSelectionKind,
        enabled: bool,
    },
    /// Change the order of ambient pick categories.
    MoveSelectionKind {
        from: usize,
        to: usize,
    },
    /// Restore the default ambient selection order and enabled categories.
    ResetSelectionFilter,
    Undo,
    Redo,
    Home,
}
mod misc;
pub use misc::{GuiOffsetPlaneBase, GuiOffsetPlaneValidation, GuiSelectionMode};
mod feature_requests;
pub use feature_requests::{
    GuiBaseViewRequest, GuiDrawingLayoutAction, GuiDrawingLine, GuiDrawingProjection,
    GuiExtrudeDistanceValidation, GuiExtrudeEditRequest, GuiExtrudePreview, GuiExtrudeProfilePick,
    GuiExtrudeReferenceExtent, GuiFilletEditRequest, GuiFilletRequest, GuiLayoutComponent,
    GuiMoveBodyRequest, GuiShellFeatureRequest,
};
mod parameter_mutation;
pub use parameter_mutation::GuiParameterMutation;
mod asset_kind;
pub use asset_kind::GuiPartAssetKind;
mod line_style;
pub use line_style::GuiSketchLineStyle;
mod browser_choice;
pub use browser_choice::GuiBrowserChoice;
mod presentation;
pub use presentation::{
    GuiDocumentStorage, GuiPreviewImage, GuiProject, GuiProjectAction, GuiProjectGroup,
    GuiProjectSettings,
};
mod state;
pub use state::{GuiAuthoringContext, GuiCommandState};
#[path = "commands_enabled.rs"]
mod enabled;
#[cfg(test)]
#[path = "../tests/commands/unit.rs"]
mod tests;
