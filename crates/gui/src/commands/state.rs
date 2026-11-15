// SPDX-License-Identifier: GPL-2.0-only
//! Snapshot of command availability and validation.
use super::*;

/// Command availability derived from the active document owner.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiCommandState {
    pub selection_mode: GuiSelectionMode,
    pub histories: Vec<oblikovati_api::wire::TransactionHistory>,
    pub selection_filter: Option<crate::GuiSelectionFilter>,
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
    /// Exact stable key of the selected body in the active document.
    pub selected_body_key: Option<String>,
    /// Exact keys of all selected bodies, in selection order.
    pub selected_body_keys: Vec<String>,
    pub documents: Vec<oblikovati_api::wire::DocumentInfo>,
    /// Active model-state names, keyed by document identity.
    pub document_model_states: std::collections::BTreeMap<u64, String>,
    pub document_storage: std::collections::BTreeMap<u64, GuiDocumentStorage>,
    /// Current drawing recipes, including drawings in detached windows.
    pub drawing_documents:
        std::collections::BTreeMap<u64, oblikovati_model::drawing::DrawingDocument>,
    pub drawing_projections: std::collections::BTreeMap<u64, GuiDrawingProjection>,
    /// Whether the active drawing uses frozen saved views.
    pub drawing_updates_deferred: bool,
    pub prompts: oblikovati_model::application_prompts::PromptPreferences,
    pub projects: Vec<GuiProject>,
    /// What a new document from each template measures as a file, in bytes.
    pub template_sizes: Vec<(crate::GuiTemplateKind, u64)>,
    /// Error retained for the chooser when template sizes cannot be inspected.
    pub template_size_issue: Option<String>,
    /// Material information read from the creation recipe for each Part template.
    pub template_materials: Vec<crate::GuiTemplateMaterialEntry>,
    pub document: Option<u64>,
    pub document_type: Option<oblikovati_api::types::DocumentType>,
    pub parameters: Vec<oblikovati_api::wire::ParameterInfo>,
    /// Typed document metadata for the iProperties editor.
    pub document_properties: Vec<oblikovati_api::wire::PropertyInfo>,
    pub authoring: GuiAuthoringContext,
    pub undo_label: Option<String>,
    pub redo_label: Option<String>,
    /// Every step that can be undone, oldest first.
    pub undo_labels: Vec<String>,
    pub transaction_open: bool,
    /// The owner has a certified extrusion service.
    pub extrude_available: bool,
    /// The owner has accepted an extrusion preview for the active document.
    pub extrude_preview_active: bool,
    /// The owner-validated distance for the current document generation.
    pub extrude_distance_validation: Option<crate::GuiExtrudeDistanceValidation>,
    /// The owner-validated offset for the current document generation.
    pub offset_plane_validation: Option<crate::GuiOffsetPlaneValidation>,
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
    /// A selected model entity can define an orientation pivot.
    pub orientation_selection_available: bool,
    /// The saved orientation control selection pivot preference.
    pub orientation_selection_locked: bool,
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
    pub(crate) fn active_express(&self) -> bool {
        self.documents
            .iter()
            .any(|row| Some(row.id) == self.document && row.express)
    }

    pub(crate) fn validate(&self) -> Result<(), String> {
        crate::history_browser::validate_histories(&self.histories)?;
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
