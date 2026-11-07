// SPDX-License-Identifier: GPL-2.0-only
#[derive(Clone, Debug, PartialEq)]
pub enum GuiShellAction {
    SetUiScale(oblikovati_model::ui_scale::UiScaleOptions),
    SetPromptFrequency {
        prompt: oblikovati_model::application_prompts::ApplicationPrompt,
        frequency: oblikovati_model::application_prompts::PromptFrequency,
    },
    SetSketchOptions(oblikovati_model::application_options::SketchApplicationOptions),
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
/// Command availability derived from the active document owner.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiCommandState {
    pub ui_scale: Option<oblikovati_model::ui_scale::UiScaleOptions>,
    pub dimension_labels: Vec<crate::GuiDimensionLabel>,
    pub dimension_draft: Option<crate::GuiDimensionDraft>,
    pub sketch_selection: oblikovati_viewport::ViewportSketchSelection,
    pub color_scheme: Option<crate::GuiColorSchemeState>,
    pub theme: Option<crate::GuiThemeState>,
    pub browser: Option<crate::GuiModelBrowser>,
    pub documents: Vec<oblikovati_api::wire::DocumentInfo>,
    pub document_storage: std::collections::BTreeMap<u64, GuiDocumentStorage>,
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
    /// The running Line command reads the keyboard before any GUI widget does.
    pub sketch_tool_keyboard: bool,
    pub dimension_tool_active: bool,
    /// A running command has no field for what is typed, so it opens the command box.
    pub takes_command_alias: bool,
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
    pub repeat_tool: Option<crate::GuiSketchTool>,
    pub line_command: Option<crate::GuiLineCommandMenu>,
    pub sketch_options: Option<oblikovati_model::application_options::SketchApplicationOptions>,
    /// A sketch is open with no command running: the canvas offers its own menu.
    pub sketch_at_ready: bool,
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
impl GuiShellAction {
    /// Preserve Go empty-stack enablement; example: `GuiShellAction::Undo.enabled(&state)`.
    pub fn enabled(&self, state: &GuiCommandState) -> bool {
        match self {
            Self::SetUiScale(_) => state.ui_scale.is_some(),
            Self::ShowHome | Self::Recent(_) => true,
            Self::ActivateWindow(_) | Self::CloseDocumentWindow(_) | Self::NewDocumentWindow => {
                !state.transaction_open
            }
            Self::SetSketchOptions(_) => state.sketch_options.is_some(),
            Self::LineCommand(_) => state.line_command.is_some(),
            Self::PrecisePoint(_) | Self::PreciseTriad(_) => state.precise_target.is_some(),
            Self::ToggleCenterPointFormat => state.authoring == GuiAuthoringContext::Sketch,
            Self::SetCenterPoint { .. } => state.sketch_at_ready && !state.transaction_open,
            Self::Inspect(_) => state.authoring == GuiAuthoringContext::Sketch,
            Self::EditDimensionValue(_) => {
                (state.sketch_at_ready || state.dimension_tool_active) && !state.transaction_open
            }
            Self::SetPromptFrequency { .. } | Self::CaptureWindowPreview | Self::Exit => true,
            Self::ActivateDocument(request) => {
                !state.transaction_open
                    && state
                        .documents
                        .iter()
                        .any(|row| row.id == request.id && row.visible)
            }
            Self::EditThemeColor { name, .. } => state
                .theme
                .as_ref()
                .is_some_and(|theme| theme.editable() && theme.active_name() == name),
            Self::DuplicateTheme { base, .. } => state
                .theme
                .as_ref()
                .is_some_and(|theme| theme.creation_available() && theme.active_name() == base),
            Self::DeleteTheme { name } => state
                .theme
                .as_ref()
                .is_some_and(|theme| theme.deletion_available() && theme.active_name() == name),
            Self::SaveTheme { name } => state
                .theme
                .as_ref()
                .is_some_and(|theme| theme.save_available() && theme.active_name() == name),
            Self::SetColorScheme(request) => state
                .color_scheme
                .as_ref()
                .is_some_and(|palette| palette.contains(&request.name)),
            Self::SetTheme(name) => state
                .theme
                .as_ref()
                .is_some_and(|theme| theme.contains(name)),
            Self::OpenDocument(_) | Self::CreateDocument(_) => !state.transaction_open,
            Self::CloseDocument(request) => {
                (state.document == Some(request.id)
                    || state.documents.iter().any(|row| row.id == request.id))
                    && !state.transaction_open
            }
            Self::SaveDocumentAs(request) => {
                state.document == Some(request.document) && !state.transaction_open
            }
            Self::OrientView { .. } | Self::SetViewProjection(_) | Self::ViewCommand(_) => {
                state.document.is_some()
            }
            Self::InspectFile(_) | Self::KeepDocumentPicture { .. } => true,
            Self::ClickModel { document, .. } | Self::UpdateBodyProperties { document, .. } => {
                state.document == Some(*document)
            }
            Self::SetBodyAppearance { document, .. } => {
                state.document == Some(*document) && !state.transaction_open
            }
            Self::EditFeature { document, .. }
            | Self::SetSketchLineStyle { document, .. }
            | Self::BrowserChoice { document, .. } => {
                state.document == Some(*document) && !state.transaction_open
            }
            Self::SelectBrowserFeature { document, feature } => {
                state.document == Some(*document)
                    && !state.transaction_open
                    && state.browser.as_ref().is_some_and(|browser| {
                        browser.features.iter().any(|row| row.id == *feature)
                    })
            }
            Self::SelectBrowserSketch {
                document, sketch, ..
            }
            | Self::SetSketchVisibility {
                document, sketch, ..
            } => {
                state.document == Some(*document)
                    && !state.transaction_open
                    && state
                        .browser
                        .as_ref()
                        .is_some_and(|browser| browser.sketches.iter().any(|row| row.id == *sketch))
            }
            Self::EditSketch { document, sketch } => {
                state.document == Some(*document)
                    && state.authoring != GuiAuthoringContext::Sketch
                    && state
                        .browser
                        .as_ref()
                        .is_some_and(|browser| browser.sketches.iter().any(|row| row.id == *sketch))
            }
            Self::Parameter { .. } => state.document.is_some(),
            Self::Undo => state.undo_label.is_some(),
            Self::Redo => state.redo_label.is_some(),
            Self::ChooseSketchPlane | Self::StartSketchOn(_) => {
                state.authoring == GuiAuthoringContext::Part
            }
            Self::DeleteSketchSelection => {
                state.authoring == GuiAuthoringContext::Sketch
                    && state.sketch_delete_available
                    && !state.transaction_open
            }
            Self::Dimension(crate::GuiDimensionAction::Start) => {
                state.authoring == GuiAuthoringContext::Sketch && !state.transaction_open
            }
            Self::Dimension(crate::GuiDimensionAction::Apply { id, .. }) => {
                state
                    .dimension_draft
                    .as_ref()
                    .is_some_and(|draft| draft.id == *id)
                    && !state.transaction_open
            }
            Self::Dimension(crate::GuiDimensionAction::Cancel { id }) => state
                .dimension_draft
                .as_ref()
                .is_some_and(|draft| draft.id == *id),
            Self::ConstrainSketch(_) => {
                state.authoring == GuiAuthoringContext::Sketch && !state.transaction_open
            }
            Self::ToggleSketchFormat(_) => {
                state.authoring == GuiAuthoringContext::Sketch
                    && state.sketch_format_available
                    && !state.transaction_open
            }
            Self::PlaceSketchTool(tool) => {
                (*tool != crate::GuiSketchTool::Point || !state.sketch_formats.construction)
                    && state.authoring == GuiAuthoringContext::Sketch
                    && !state.transaction_open
            }
            Self::FinishSketch => {
                state.authoring == GuiAuthoringContext::Sketch && !state.transaction_open
            }
            Self::NewPart | Self::Save | Self::Home => true,
        }
    }
    /// Keep available history controls inert during an edit; example: `action.dispatchable(&state)` after a click.
    pub fn dispatchable(&self, state: &GuiCommandState) -> bool {
        let history = matches!(self, Self::Undo | Self::Redo | Self::EditSketch { .. });
        self.enabled(state) && !(history && state.transaction_open)
    }
    pub(crate) fn next_label<'a>(&self, state: &'a GuiCommandState) -> Option<&'a str> {
        match self {
            Self::Undo => state.undo_label.as_deref(),
            Self::Redo => state.redo_label.as_deref(),
            _ => None,
        }
    }
}
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
