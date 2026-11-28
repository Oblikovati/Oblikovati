// SPDX-License-Identifier: GPL-2.0-only
//! Snapshot of command availability and validation.
use super::*;

/// The current authoring workspace.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiAuthoringContext {
    #[default]
    NoDocument,
    OtherDocument,
    Part,
    Sketch,
}
/// Command availability derived from the active document owner.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiCommandState {
    /// The application window is maximized by the desktop.
    pub window_maximized: bool,
    pub selection_mode: GuiSelectionMode,
    pub histories: Vec<oblikovati_api::wire::TransactionHistory>,
    pub selection_filter: Option<crate::GuiSelectionFilter>,
    /// Per-environment marking-menu settings owned by the desktop.
    pub marking_menus: Option<oblikovati_model::marking_menu::MarkingMenus>,
    /// Marking menu for the current sketch environment.
    pub sketch_marking_menu: Option<oblikovati_api::wire::MarkingMenuView>,
    /// Render sketch marking commands as a classic linear menu.
    pub sketch_classic_menu: bool,
    /// Saved presentation of canvas marking-menu commands.
    pub marking_menu_display: oblikovati_model::marking_menu::MarkingMenuDisplay,
    /// The owner's effective shortcut state for a customization draft.
    pub keymap: Option<oblikovati_model::keymap::Keymap>,
    /// Bindable commands currently provided by the desktop owner.
    pub keymap_bindings: Vec<oblikovati_api::wire::BindingInfo>,
    /// Pointer coordinates in the current document’s displayed length unit.
    pub viewport_coordinates: Option<[f64; 2]>,
    /// Camera magnification as a percentage of the default camera scale.
    pub viewport_zoom_percent: Option<u64>,
    pub document_units: Option<oblikovati_api::wire::DocumentUnitsInfo>,
    pub ui_scale: Option<oblikovati_model::ui_scale::UiScaleOptions>,
    pub dimension_labels: Vec<crate::GuiDimensionLabel>,
    pub dimension_draft: Option<crate::GuiDimensionDraft>,
    pub sketch_selection: oblikovati_viewport::ViewportSketchSelection,
    /// Effective sketch authoring settings for the active document.
    pub document_sketch_settings: Option<oblikovati_api::types::SketchSettings>,
    /// Applied scope for the active sketch session, if available.
    pub sketch_inference_scope: Option<oblikovati_api::wire::SketchInferenceScopeView>,
    /// Explain an invalid saved selection until a replacement is accepted.
    pub sketch_inference_scope_error: Option<String>,
    /// Transient type cue for a checked canvas pick.
    pub sketch_pick_cue: Option<crate::GuiSketchPickCue>,
    /// Transient type cue for a checked model face pick.
    pub model_pick_cue: Option<crate::GuiModelPickCue>,
    /// The picked point remains the toolbar anchor until the selected face changes.
    pub face_toolbar_anchor: Option<crate::GuiModelPickCue>,
    /// Names in the active document's block registry.
    pub sketch_block_names: Vec<String>,
    /// Last block creation acknowledged by the document owner during this dialog.
    pub sketch_block_created: Option<String>,
    pub sketch_pattern_created: Option<(u64, bool)>,
    /// The host provides the rectangular sketch-pattern command.
    pub sketch_pattern_available: bool,
    pub active_sketch: Option<oblikovati_model::document_sketch::DocumentSketchId>,
    /// The current sketch plane, used to project display-only axes.
    pub active_sketch_plane: Option<oblikovati_model::sketch::SketchPlane>,
    /// Sketch axes projected into framebuffer pixels by the desktop frame.
    pub sketch_axes: Vec<[[i32; 2]; 2]>,
    pub color_scheme: Option<crate::GuiColorSchemeState>,
    pub theme: Option<crate::GuiThemeState>,
    pub browser: Option<crate::GuiModelBrowser>,
    /// Exact stable key of the selected body in the active document.
    pub selected_body_key: Option<String>,
    /// Exact keys of all selected bodies, in selection order.
    pub selected_body_keys: Vec<String>,
    /// Current exact face references and whether each has an override.
    pub selected_face_appearances: Vec<(String, bool)>,
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
    /// Completion of a specific project file request, with the registered location or error.
    pub project_file_result: Option<(u64, Result<String, String>)>,
    /// Native template files in the selected project, if it supplies a catalog.
    pub project_templates: std::sync::Arc<[crate::GuiProjectTemplate]>,
    /// Changes when the active project's catalog is rebuilt.
    pub project_template_revision: u64,
    pub project_template_issue: Option<String>,
    /// The component awaiting acknowledgement before a populated drawing is created.
    pub layout_warning: Option<String>,
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
    /// The active part model-state property table.
    pub model_state_table: Option<std::sync::Arc<oblikovati_api::wire::ModelStateTableView>>,
    /// Completion of a specific table edit request, including host failures.
    pub model_state_table_ack: Option<(u64, u64, Result<(), String>)>,
    /// The last checked table draft and its diagnostics.
    pub model_state_verification: Option<(
        u64,
        oblikovati_api::wire::SetModelStateTableArgs,
        oblikovati_api::wire::ModelStateTableVerification,
    )>,
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
    /// Current certified Sweep preview for the active document.
    pub loft_sections: Vec<crate::GuiLoftSection>,
    pub loft_center_line: Option<String>,
    pub loft_placed: Option<crate::GuiLoftPlacedView>,
    pub loft_takeoff_dirty: bool,
    pub loft_edit_warning: Option<String>,
    pub loft_preview: Option<Result<(), String>>,
    pub sweep_preview: Option<crate::GuiSweepPreviewState>,
    pub revolve_preview: Option<crate::GuiRevolvePreviewState>,
    pub coil_preview: Option<crate::GuiCoilPreviewState>,
    pub emboss_preview: Option<crate::GuiEmbossPreviewState>,
    pub derive_preview: Option<crate::GuiDerivePreviewState>,
    pub derive_source: Option<crate::GuiDeriveSourceSummary>,
    pub rib_preview: Option<crate::GuiRibPreviewState>,
    pub rib_sketches: Vec<crate::GuiRibSketchCandidate>,
    pub decal_preview: Option<crate::GuiDecalPreviewState>,
    pub decal_images: Vec<crate::GuiDecalImageCandidate>,
    pub import_preview: Option<crate::GuiImportPreviewState>,
    pub import_source: Option<crate::GuiImportSourceSummary>,
    pub unwrap_preview: Option<crate::GuiUnwrapPreviewState>,
    pub unwrap_pick: Option<crate::GuiUnwrapFacePick>,
    pub hole_preview: Option<crate::GuiHolePreviewState>,
    pub hole_pick: Option<crate::GuiHolePick>,
    pub slot_preview: Option<crate::GuiSlotPreviewState>,
    pub slot_pick: Option<crate::GuiSlotPick>,
    pub combine_preview: Option<crate::GuiCombinePreviewState>,
    pub combine_pick: Option<crate::GuiCombinePick>,
    pub draft_preview: Option<crate::GuiDraftPreviewState>,
    pub draft_pick: Option<crate::GuiDraftPick>,
    pub has_threadable_faces: bool,
    pub thread_preview: Option<crate::GuiThreadPreviewState>,
    pub thread_pick: Option<crate::GuiThreadPick>,
    pub delete_face_preview: Option<crate::GuiDeleteFacePreviewState>,
    pub delete_face_pick: Option<crate::GuiDeleteFacePick>,
    pub fillet_v2_preview: Option<crate::GuiFilletV2PreviewState>,
    pub fillet_v2_pick: Option<crate::GuiFilletV2Pick>,
    pub chamfer_v2_preview: Option<crate::GuiChamferV2PreviewState>,
    pub chamfer_v2_pick: Option<crate::GuiChamferV2Pick>,
    pub shell_v2_preview: Option<crate::GuiShellV2PreviewState>,
    pub shell_v2_pick: Option<crate::GuiShellV2Pick>,
    /// Selected named solid face reference key when a model face is chosen.
    pub selected_model_face: Option<Vec<u8>>,
    /// The exact extrusion draft accepted for preview at this document generation.
    pub extrude_draft_preview: Option<crate::GuiExtrudePreview>,
    pub extrude_profile_pick: Option<(crate::GuiExtrudeProfilePick, i32)>,
    pub extrude_handle: Option<crate::GuiExtrudeHandle>,
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
    pub curvature_settings: Option<crate::GuiCurvatureSettings>,
    /// The command the canvas menu offers to run again.
    pub repeat_command: Option<crate::GuiSketchRepeat>,
    pub line_command: Option<crate::GuiLineCommandMenu>,
    pub sketch_options: Option<oblikovati_model::application_options::SketchApplicationOptions>,
    pub part_options: Option<oblikovati_model::application_options::PartApplicationOptions>,
    /// A sketch is open with no command running: the canvas offers its own menu.
    pub sketch_at_ready: bool,
    /// Surrounding part bodies are temporarily hidden for the active sketch.
    pub sketch_isolated: bool,
    /// Current canvas move stage when the Move tool is open.
    pub sketch_move: Option<crate::GuiSketchMoveStatus>,
    pub sketch_text: Option<crate::GuiSketchTextDraft>,
    pub sketch_text_existing: bool,
    pub sketch_text_preset: Option<crate::GuiSketchTextPresetId>,
    pub sketch_text_font_families: Option<std::sync::Arc<[String]>>,
    pub sketch_text_field_sources: Option<std::sync::Arc<crate::GuiSketchTextFieldSources>>,
    pub sketch_text_editor_fields: Option<std::sync::Arc<crate::GuiSketchTextEditorFields>>,
    pub origin_planes: Vec<crate::GuiOriginPlane>,
    /// The chosen browser row's outline in the view, in pixels: a solid's edges or a sketch.
    pub feature_edges: Vec<[[i32; 2]; 2]>,
    /// Projected display segments grouped by the selected exact edge.
    pub selected_edge_outlines: Vec<Vec<[[i32; 2]; 2]>>,
    pub origin_axes: Vec<crate::GuiOriginAxis>,
    /// What the owner read from the file an Open dialog has selected.
    pub inspected_file: Option<crate::GuiInspectedFile>,
    /// A face of the model is chosen in the view.
    pub model_face_chosen: bool,
    /// Current exact solid edge selection count.
    pub selected_edge_count: usize,
    /// Generation of the active document for generation-scoped feature drafts.
    pub document_generation: Option<u64>,
    /// Certified part feature replay is configured for this document.
    pub certified_part_feature_available: bool,
    /// Current exact edge keys and document generation for a fillet draft.
    pub fillet_selection: Option<(u64, Vec<Vec<u8>>)>,
    pub fillet_available: bool,
    /// The selected work plane or face has a current planar frame.
    pub look_at_available: bool,
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
    pub face_properties: Option<crate::GuiFaceProperties>,
    pub region_properties: Option<crate::GuiRegionPanel>,
    pub sketch_delete_available: bool,
    pub sketch_format_available: bool,
    pub sketch_formats: oblikovati_api::types::SketchFormatModes,
}
impl GuiCommandState {
    /// Find an unambiguous sketch for starting a placement from the Part workspace.
    /// For example, `state.placement_sketch_entry()` returns None for an empty Part.
    pub fn placement_sketch_entry(
        &self,
    ) -> Option<(u64, oblikovati_model::document_sketch::DocumentSketchId)> {
        if self.authoring != GuiAuthoringContext::Part
            || self.document_type != Some(oblikovati_api::types::DOCUMENT_PART)
            || self.transaction_open
        {
            return None;
        }
        let [sketch] = self.browser.as_ref()?.sketches.as_slice() else {
            return None;
        };
        sketch.inspection.as_ref().ok()?;
        Some((self.document?, sketch.id))
    }

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
