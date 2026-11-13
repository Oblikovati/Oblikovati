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
    /// Give a solid its own appearance; an empty id clears it.
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
    NewPart,
    CreateDocument(crate::GuiDocumentTemplate),
    /// Picture the active window for the Save As preview; the file is not changed.
    CaptureWindowPreview,
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
    StartSketchOn(crate::GuiOriginPlaneKind),
    FinishSketch,
    /// Open the Extrude dialog for the active part.
    OpenExtrude,
    /// Commit the chosen sketch profile through the document owner.
    Extrude {
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        profile: i64,
        distance: String,
    },
    /// Show or clear a transient extrusion preview.
    PreviewExtrude(Option<GuiExtrudePreview>),
    DeleteSketchSelection,
    ToggleSketchFormat(oblikovati_model::sketch::SketchCurveFormat),
    PlaceSketchTool(crate::GuiSketchTool),
    ConstrainSketch(crate::GuiSketchConstraint),
    Dimension(crate::GuiDimensionAction),
    Save,
    Undo,
    Redo,
    Home,
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
    pub document: u64,
    pub sketch: oblikovati_model::document_sketch::DocumentSketchId,
    pub profile: i64,
    pub distance: String,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiParameterMutation {
    Add,
    Set,
}
/// A bounded owner snapshot; the GUI holds no document history or transaction token.
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
    Body {
        key: String,
        feature: oblikovati_model::feature::FeatureId,
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
/// Command availability derived from the active document owner.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiCommandState {
    /// Per-environment marking-menu settings owned by the desktop.
    pub marking_menus: Option<oblikovati_model::marking_menu::MarkingMenus>,
    /// Marking menu for the current sketch environment.
    pub sketch_marking_menu: Option<oblikovati_api::wire::MarkingMenuView>,
    /// Render sketch marking commands as a classic linear menu.
    pub sketch_classic_menu: bool,
    /// The owner's effective shortcut state for a customization draft.
    pub keymap: Option<oblikovati_model::keymap::Keymap>,
    /// Bindable commands currently provided by the desktop owner.
    pub keymap_bindings: Vec<oblikovati_api::wire::BindingInfo>,
    pub document_units: Option<oblikovati_api::wire::DocumentUnitsInfo>,
    pub ui_scale: Option<oblikovati_model::ui_scale::UiScaleOptions>,
    pub dimension_labels: Vec<crate::GuiDimensionLabel>,
    pub dimension_draft: Option<crate::GuiDimensionDraft>,
    pub sketch_selection: oblikovati_viewport::ViewportSketchSelection,
    /// Names in the active document's block registry.
    pub sketch_block_names: Vec<String>,
    /// Last block creation acknowledged by the document owner during this dialog.
    pub sketch_block_created: Option<String>,
    pub sketch_pattern_created: Option<(u64, bool)>,
    /// The host provides the rectangular sketch-pattern command.
    pub sketch_pattern_available: bool,
    pub active_sketch: Option<oblikovati_model::document_sketch::DocumentSketchId>,
    pub color_scheme: Option<crate::GuiColorSchemeState>,
    pub theme: Option<crate::GuiThemeState>,
    pub browser: Option<crate::GuiModelBrowser>,
    pub documents: Vec<oblikovati_api::wire::DocumentInfo>,
    /// Active model-state names, keyed by document identity.
    pub document_model_states: std::collections::BTreeMap<u64, String>,
    pub document_storage: std::collections::BTreeMap<u64, GuiDocumentStorage>,
    /// Current drawing recipes, including drawings in detached windows.
    pub drawing_documents:
        std::collections::BTreeMap<u64, oblikovati_model::drawing::DrawingDocument>,
    pub drawing_projections: std::collections::BTreeMap<u64, GuiDrawingProjection>,
    pub prompts: oblikovati_model::application_prompts::PromptPreferences,
    pub projects: Vec<GuiProject>,
    /// What a new document from each template measures as a file, in bytes.
    pub template_sizes: Vec<(crate::GuiTemplateKind, u64)>,
    pub document: Option<u64>,
    pub document_type: Option<oblikovati_api::types::DocumentType>,
    pub parameters: Vec<oblikovati_api::wire::ParameterInfo>,
    pub authoring: GuiAuthoringContext,
    pub undo_label: Option<String>,
    pub redo_label: Option<String>,
    /// Every step that can be undone, oldest first.
    pub undo_labels: Vec<String>,
    pub transaction_open: bool,
    /// The owner has a certified extrusion service.
    pub extrude_available: bool,
    /// The running Line command reads the keyboard before any GUI widget does.
    pub sketch_tool_keyboard: bool,
    pub dimension_tool_active: bool,
    /// A running command has no field for what is typed, so it opens the command box.
    pub takes_command_alias: bool,
    /// Alias text entered while a sketch is ready for a command.
    pub sketch_alias: Option<String>,
    /// The edited sketch's remaining freedom; `None` outside a sketch.
    pub dimensions_needed: Option<usize>,
    pub sketch_hud: Option<crate::GuiSketchHud>,
    /// Projected by the frame, which owns the camera.
    pub constraint_glyphs: Vec<crate::GuiConstraintGlyph>,
    /// A running command that accepts typed points; the frame adds the triad's pixels.
    pub precise_target: Option<crate::GuiPreciseTarget>,
    /// New points are center points.
    pub center_point_format: bool,
    /// What the pointer was on when the canvas menu was asked for.
    pub sketch_context: Option<crate::GuiSketchContext>,
    /// The command the canvas menu offers to run again.
    pub repeat_command: Option<crate::GuiSketchRepeat>,
    pub line_command: Option<crate::GuiLineCommandMenu>,
    pub sketch_options: Option<oblikovati_model::application_options::SketchApplicationOptions>,
    /// A sketch is open with no command running: the canvas offers its own menu.
    pub sketch_at_ready: bool,
    /// Current canvas move stage when the Move tool is open.
    pub sketch_move: Option<crate::GuiSketchMoveStatus>,
    pub origin_planes: Vec<crate::GuiOriginPlane>,
    /// The chosen browser row's outline in the view, in pixels: a solid's edges or a sketch.
    pub feature_edges: Vec<[[i32; 2]; 2]>,
    pub origin_axes: Vec<crate::GuiOriginAxis>,
    /// What the owner read from the file an Open dialog has selected.
    pub inspected_file: Option<crate::GuiInspectedFile>,
    /// A face of the model is chosen in the view.
    pub model_face_chosen: bool,
    /// How the view looks at the model, for the orientation control.
    pub view_axes: Option<crate::GuiViewAxes>,
    pub document_windows: Vec<crate::GuiDocumentWindow>,
    pub active_window: Option<u64>,
    pub recent_documents: Vec<crate::GuiRecentDocument>,
    pub measure: Option<crate::GuiMeasurePanel>,
    pub region_properties: Option<crate::GuiRegionPanel>,
    pub sketch_delete_available: bool,
    pub sketch_format_available: bool,
    pub sketch_formats: oblikovati_api::types::SketchFormatModes,
}
impl GuiCommandState {
    pub(crate) fn validate(&self) -> Result<(), String> {
        crate::dimensions::validate_dimension_draft(self)?;
        crate::dimension_labels::validate_dimension_labels(&self.dimension_labels)?;
        crate::document_tabs::validate_documents(&self.documents)?;
        for (id, name) in &self.document_model_states {
            if name.contains('\0') || !self.documents.iter().any(|document| document.id == *id) {
                return Err(format!(
                    "document model state {id}={name:?}: expected an open document and text without NUL"
                ));
            }
        }
        if self.browser.is_some() && self.document.is_none() {
            return Err("GUI browser document None: expected an active document identity".into());
        }
        if let Some(browser) = &self.browser {
            browser.validate()?;
        }
        let named = [&self.undo_label, &self.redo_label].into_iter().flatten();
        for label in named.chain(&self.undo_labels) {
            if label.contains('\0') {
                return Err(format!(
                    "GUI history label {label:?}: expected text without NUL"
                ));
            }
        }
        for parameter in &self.parameters {
            validate_parameter_row(parameter)?;
        }
        Ok(())
    }
}
#[path = "commands_enabled.rs"]
mod enabled;
fn validate_parameter_row(parameter: &oblikovati_api::wire::ParameterInfo) -> Result<(), String> {
    for value in [
        &parameter.name,
        &parameter.kind,
        &parameter.expression,
        &parameter.value,
        &parameter.health,
    ] {
        if value.contains('\0') {
            return Err(format!(
                "GUI parameter value {value:?}: expected text without NUL"
            ));
        }
    }
    Ok(())
}
#[cfg(test)]
#[path = "../tests/commands/unit.rs"]
mod tests;
