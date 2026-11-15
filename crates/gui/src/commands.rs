// SPDX-License-Identifier: GPL-2.0-only
#[derive(Clone, Debug, PartialEq)]
pub enum GuiShellAction {
    /// Apply one staged marking menu.
    SetMarkingMenu(oblikovati_api::wire::SetMarkingMenuArgs),
    /// Apply a staged keyboard customization.
    SetKeymap(oblikovati_api::wire::KeymapExport),
    /// Place instances of a sketch block definition.
    PlaceSketchBlock(String),
    /// Enter or leave the Create Block geometry selection mode.
    SelectBlockGeometry(bool),
    /// Group the editor's current checked selection under this name.
    CreateSketchBlock(String),
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
    SketchMove(crate::GuiSketchMoveAction),
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
    /// Calculate one solid's physical values, for the Body Properties dialog.
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
    /// Open the clear tool for the active document.
    OpenClearBodyAppearance {
        document: u64,
    },
    /// Clear the selected solid's own appearance override.
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
    NewPart,
    CreateDocument(crate::GuiDocumentTemplate),
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
    StartSketchOn(crate::GuiOriginPlaneKind),
    /// Start on a selected authored work plane.
    StartSketchOnWorkPlane(oblikovati_model::feature::FeatureId),
    /// Open the offset-plane editor for the active part.
    OpenOffsetPlane,
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
    /// Open the Extrude dialog for the active part.
    OpenExtrude,
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
    },
    /// Show or clear a transient extrusion preview.
    PreviewExtrude(Option<GuiExtrudePreview>),
    /// Ask the document owner to validate the current extrusion expression.
    ValidateExtrudeDistance {
        document: u64,
        distance: String,
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
/// The current selection used to define an offset plane.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiOffsetPlaneBase {
    Origin(crate::GuiOriginPlaneKind),
    WorkPlane(oblikovati_model::feature::FeatureId),
    SelectedFace,
}
/// The owner-validated offset for one document generation.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiOffsetPlaneValidation {
    pub generation: u64,
    pub document: u64,
    pub offset: String,
    pub error: Option<String>,
}
/// The selection categories supported by the viewport's current pickers.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum GuiSelectionMode {
    #[default]
    All,
    Faces,
    Components,
    Custom,
}
/// A drawing view chosen in the desktop dialog.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiBaseViewRequest {
    pub drawing: u64,
    pub source: u64,
    pub orientation: String,
    pub scale: f64,
    pub center: [f64; 2],
}
/// Transient profile preview request owned by the current document.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiExtrudePreview {
    pub generation: Option<u64>,
    pub document: u64,
    pub sketch: oblikovati_model::document_sketch::DocumentSketchId,
    pub profile: i64,
    pub distance: String,
}
/// A document-scoped extrusion distance draft.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiExtrudeEditRequest {
    pub generation: Option<u64>,
    pub document: u64,
    pub feature: oblikovati_model::feature::FeatureId,
    pub distance: String,
}
/// Owner validation for one extrusion expression and active document.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiExtrudeDistanceValidation {
    pub generation: u64,
    pub document: u64,
    pub distance: String,
    pub error: Option<String>,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiParameterMutation {
    Add,
    Set,
}
/// A bounded owner snapshot; the GUI holds no document history or transaction token.
/// The part-level asset a ribbon selection changes.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiPartAssetKind {
    Material,
    Appearance,
}
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiAuthoringContext {
    #[default]
    NoDocument,
    OtherDocument,
    Part,
    Sketch,
}
pub use crate::application::model_browser::feature_properties::{
    GuiFeatureCondition, GuiFeatureEdit,
};
/// How a sketch's lines are drawn; a part left unset draws as the application's default.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiSketchLineStyle {
    pub color: Option<[u8; 3]>,
    pub line_type: Option<String>,
    /// In millimeters.
    pub weight: Option<f32>,
    pub scale: Option<f32>,
}
/// What a press on a browser row other than a feature or sketch asks for.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiBrowserChoice {
    /// Create and activate a new model-state member.
    CreateModelState,
    /// Copy a member without activating the copy.
    CopyModelState(u64),
    /// Rename one document-owned member.
    RenameModelState {
        id: u64,
        name: String,
    },
    /// Activate a document-owned model state.
    ActivateModelState(u64),
    /// Change whether a feature edit affects one member or all members.
    ToggleModelStateEditScope,
    /// No row stays chosen, as after a click on empty space.
    Nothing,
    /// Show this feature's dimensions in the view, or none.
    ShowFeatureDimensions(Option<oblikovati_model::feature::FeatureId>),
    /// Whether a sketch shows its dimensions while it is visible.
    SketchDimensionsShown {
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        shown: bool,
    },
    Root,
    /// Choose a placed component by its generation-scoped owner path.
    Occurrence(crate::GuiOccurrencePath),
    /// Change the suppression state of an assembly-owned occurrence.
    SetOccurrenceSuppressed {
        path: crate::GuiOccurrencePath,
        suppressed: bool,
    },
    Body {
        key: String,
        feature: oblikovati_model::feature::FeatureId,
        toggle: bool,
    },
    BodyVisible {
        key: String,
        visible: bool,
    },
}
/// A small picture of the window: enough cells for a preview area, drawn without a texture.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiPreviewImage {
    pub extent: [u32; 2],
    pub rgb: std::sync::Arc<Vec<[u8; 3]>>,
}
/// One loaded project, with the settings its dialog lists.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiProject {
    pub name: String,
    pub location: String,
    pub active: bool,
    pub settings: Vec<(String, String)>,
    /// What the tree folds away until it is asked for; example: `Folder Options` and its rows.
    pub groups: Vec<GuiProjectGroup>,
}
/// One foldable branch of a project's settings.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiProjectGroup {
    pub label: String,
    pub rows: Vec<String>,
}
/// What the owner knows about one open document's file beyond its wire row.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiDocumentStorage {
    pub path: Option<String>,
    /// Closing without saving would lose work; a new, untouched part has none to lose.
    pub unsaved_edits: bool,
}
/// Sheet-space display geometry and any model-resolution failure.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiDrawingProjection {
    pub lines: Vec<GuiDrawingLine>,
    pub error: Option<String>,
}
/// One sampled edge in sheet millimetres.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiDrawingLine {
    pub points: Vec<[f64; 2]>,
    pub hidden: bool,
    pub preview: bool,
}
mod state;
pub use state::GuiCommandState;
#[path = "commands_enabled.rs"]
mod enabled;
#[cfg(test)]
#[path = "../tests/commands/unit.rs"]
mod tests;
