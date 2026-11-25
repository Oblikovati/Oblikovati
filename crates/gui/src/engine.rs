// SPDX-License-Identifier: GPL-2.0-only
#[path = "engine/model_click.rs"]
mod model_click;
use crate::{
    GuiAtlas, GuiCommandState, GuiFrame, GuiFrameActivity, GuiInputCapture, GuiInputEvent,
    GuiShellAction,
    context_owner::{create_owner_context, frame_activity},
    frame_keys::prepare_frame_inputs,
    input::{input_capture, validate_input},
    snapshot::capture_frame,
};
use imgui::{ConfigFlags, Context};
use std::sync::Arc;
#[path = "engine/close_prompt_scale.rs"]
mod close_prompt_scale;
pub(crate) mod dock_registration;
#[path = "engine/dock_tab_actions.rs"]
mod dock_tab_actions;
#[path = "engine/document_name.rs"]
mod document_name;
#[path = "engine/draw_controls.rs"]
mod draw_controls;
mod external_panels;
mod work_plane;
use work_plane::selected_work_plane;
mod frame_inputs;
mod inference_dialog;
mod quick_access;
use quick_access::{apply_home_requests, apply_quick_access};
#[path = "engine/template_selection.rs"]
mod template_selection;
#[path = "engine/workspace_layout.rs"]
mod workspace_layout;
#[derive(Clone, Debug)]
pub struct GuiFrameRequest {
    pub extent: [u32; 2],
    pub events: Vec<GuiInputEvent>,
    pub commands: GuiCommandState,
    pub status: Option<String>,
    pub delta_seconds: f32,
    pub framebuffer_scale: [f32; 2],
}
/// A request from the application window controls.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiWindowAction {
    Minimize,
    ToggleMaximize,
    Move,
}
#[derive(Clone, Debug)]
pub struct GuiWindowControls {
    pub bounds: [[f32; 4]; 3],
    pub focused: Option<usize>,
}
#[derive(Clone, Debug)]
pub struct GuiFrameOutput {
    pub window_action: Option<GuiWindowAction>,
    pub window_controls: GuiWindowControls,
    pub close_prompt: Option<crate::GuiClosePrompt>,
    pub accessibility_popup_active: bool,
    pub draws: GuiFrame,
    pub actions: Vec<GuiShellAction>,
    pub parameter_actions: Vec<crate::GuiParameterAction>,
    pub constraint_hover: Option<crate::GuiConstraintHover>,
    pub capture: GuiInputCapture,
    pub sketch_double_click: bool,
    pub activity: GuiFrameActivity,
    /// The unobstructed document area in framebuffer pixels, if it has positive size.
    pub document_area: Option<[u32; 4]>,
}
/// One GUI thread retains the active context for its lifetime; packets contain no borrowed FFI pointers.
/// ```compile_fail
/// fn require_send<T: Send>() {}
/// require_send::<oblikovati_gui::GuiFrameEngine>();
/// ```
pub struct GuiFrameEngine {
    context: Context,
    dock_lifecycle: crate::dockable_lifecycle::DockRegistry,
    external_docks: crate::external_dock_window::ExternalDockWindows,
    #[cfg(test)]
    _test_context_lease: crate::test_context::TestContextLease,
    atlas: GuiAtlas,
    resources: crate::GuiTextureResources,
    icons: crate::icons::GuiIconCache,
    raster_scale: f32,
    preferences: crate::ui_scale::GuiScalePreferences,
    extrude_dialog: crate::extrude_dialog::ExtrudeDialog,
    blend_dialog: crate::blend_dialog::BlendDialog,
    hole_dialog: crate::hole_dialog::HoleDialog,
    shell_feature_dialog: crate::shell_feature_dialog::ShellFeatureDialog,
    move_body_dialog: crate::move_body_dialog::MoveBodyDialog,
    part_context_open: bool,
    selector_context_gesture: bool,
    offset_plane: crate::offset_plane::OffsetPlaneDialog,
    pending_extrusion_shortcut: bool,
    window_frame_interaction: crate::window_frame::FrameInteraction,
    accessible_focus: Option<crate::GuiAccessibleFocus>,
    dimension_value_editor: crate::sketch::dimension_edit::GuiDimensionValueEditor,
    precise_input: crate::sketch::precise_input::PreciseInputBar,
    create_block: crate::sketch::create_block::CreateBlockDialog,
    sketch_text: crate::sketch::text::SketchTextDialog,
    curvature_settings: crate::sketch::curvature::CurvatureSettingsWindow,
    inference_scope: crate::sketch::inference_scope::InferenceScopeDialog,
    sketch_pattern: crate::sketch::rectangular_pattern::RectangularPatternDialog,
    command_alias: crate::sketch::command_alias::CommandAliasBox,
    /// A popup was open last frame: Escape closes it, and the frame that opens is already too
    /// late to ask, since Dear ImGui acts on the key as the frame begins.
    popup_was_open: bool,
    sketch_popup_token: Option<u64>,
    /// Keys a widget received the press of, so their release reaches it too.
    widget_keys: Vec<crate::GuiKey>,
    quick_access: crate::application::quick_access_toolbar::QuickAccessToolbar,
    metadata: crate::application::document_metadata::MetadataEditor,
    model_state_table: crate::model_state_table::ModelStateTable,
    clear_appearance: crate::clear_appearance::ClearAppearance,
    material_browser: crate::material_browser::MaterialBrowser,
    appearance_browser: crate::material_browser::MaterialBrowser,
    home: crate::application::home_recent_documents::HomeWorkspace,
    plane_selection: crate::application::quick_access_toolbar::plane_selection::PlaneSelection,
    parameter_editor: crate::parameters::GuiParameterEditor,
    selection_filter: crate::selection_filter::SelectionFilterPanel,
    history_browser: crate::history_browser::HistoryBrowser,
    document_files: crate::files::GuiDocumentFiles,
    ribbon: crate::ribbon::DocumentRibbon,
    document_tabs: crate::document_tabs::DocumentTabs,
    workspace: crate::application::workspace_layout::WorkspaceLayout,
    workspace_layout_restored: bool,
    orientation: crate::application::viewport_orientation::OrientationControl,
    /// The documents of the latest frame, for dialogs an owner notice opens between frames.
    known_documents: Vec<oblikovati_api::wire::DocumentInfo>,
    known_document_type: Option<oblikovati_api::types::DocumentType>,
    workspaces: crate::application::document_creation::workspaces::DocumentWorkspaces,
    /// Modifier state across frames: a modifier and Tab can arrive in different frames.
    modifiers_held: crate::GuiModifiers,
    theme_editor: crate::theme_editor::GuiThemeEditor,
}
impl GuiFrameEngine {
    /// Read the drawings for which the desktop owns secondary windows.
    /// Example: `engine.detached_drawings()` before synchronizing native targets.
    pub fn detached_drawings(&self) -> &[u64] {
        self.document_tabs.detached_drawings()
    }
    /// Restore a drawing tab when its native window closes.
    /// Example: `engine.restore_detached_drawing(drawing_id)` after a native close.
    pub fn restore_detached_drawing(&mut self, drawing: u64) {
        self.document_tabs.restore_detached_drawing(drawing);
    }
    /// Open an extrusion draft for the active part. Example: `engine.request_extrusion()`.
    pub fn request_extrusion(&mut self) {
        self.pending_extrusion_shortcut = true;
    }
    /// Load and save applied ribbon settings through a host store.
    /// Example: `engine.set_ribbon_settings_store(store)?`.
    pub fn set_ribbon_settings_store(
        &mut self,
        store: Box<dyn crate::GuiRibbonSettingsStore>,
    ) -> Result<(), String> {
        self.ribbon.customization.set_store(store)
    }
    /// Create the GUI owner on its UI thread without automatic file I/O; example: `GuiFrameEngine::new()?`.
    pub fn new() -> Result<Self, String> {
        #[cfg(test)]
        let test_context_lease = crate::test_context::acquire();
        let mut context = create_owner_context()?;
        let atlas = configure_owner_context(&mut context)?;
        let resources = crate::GuiTextureResources::new(atlas.clone(), [])?;
        let icons = crate::icons::GuiIconCache::new(atlas.clone())?;
        let mut engine = Self {
            context,
            dock_lifecycle: Default::default(),
            external_docks: Default::default(),
            #[cfg(test)]
            _test_context_lease: test_context_lease,
            atlas,
            resources,
            icons,
            raster_scale: 1.,
            preferences: Default::default(),
            extrude_dialog: Default::default(),
            blend_dialog: Default::default(),
            hole_dialog: Default::default(),
            shell_feature_dialog: Default::default(),
            move_body_dialog: Default::default(),
            part_context_open: false,
            selector_context_gesture: false,
            offset_plane: Default::default(),
            pending_extrusion_shortcut: false,
            window_frame_interaction: Default::default(),
            accessible_focus: None,
            dimension_value_editor: Default::default(),
            precise_input: Default::default(),
            create_block: Default::default(),
            sketch_text: Default::default(),
            curvature_settings: Default::default(),
            inference_scope: Default::default(),
            sketch_pattern: Default::default(),
            command_alias: Default::default(),
            popup_was_open: false,
            sketch_popup_token: None,
            widget_keys: Vec::new(),
            quick_access: Default::default(),
            metadata: Default::default(),
            model_state_table: Default::default(),
            clear_appearance: Default::default(),
            material_browser: Default::default(),
            appearance_browser: crate::material_browser::MaterialBrowser::appearances(),
            home: Default::default(),
            plane_selection: Default::default(),
            parameter_editor: crate::parameters::GuiParameterEditor::new()?,
            selection_filter: Default::default(),
            history_browser: Default::default(),
            document_files: Default::default(),
            ribbon: Default::default(),
            document_tabs: Default::default(),
            workspace: Default::default(),
            workspace_layout_restored: false,
            orientation: Default::default(),
            known_documents: Vec::new(),
            known_document_type: None,
            workspaces: Default::default(),
            modifiers_held: Default::default(),
            theme_editor: Default::default(),
        };
        dock_registration::register(&mut engine)?;
        Ok(engine)
    }
    /// Apply semantic chrome colours on the GUI thread; example: `engine.apply_theme(library.active())`.
    pub fn apply_theme(&mut self, theme: &oblikovati_model::theme::ThemeDefinition) {
        self.icons.set_theme(theme);
        crate::theme::apply_chrome_theme(&mut self.context.style_mut().colors, theme.palette());
    }
    /// Supply the current document parameters to the editor.
    /// Example: `engine.set_parameter_snapshot(Some(snapshot))?`.
    pub fn set_parameter_snapshot(
        &mut self,
        snapshot: Option<crate::GuiParameterSnapshot>,
    ) -> Result<bool, String> {
        self.parameter_editor.set_snapshot(snapshot)
    }
    /// Show a parameter metadata error and discard stale rows.
    /// Example: `engine.report_parameter_view_issue(issue)?`.
    pub fn report_parameter_view_issue(&mut self, issue: String) -> Result<bool, String> {
        self.parameter_editor.report_issue(issue)
    }
    /// Show or close the editor for the current document.
    /// Example: `engine.set_parameters_open(true)`.
    pub fn set_parameters_open(&mut self, open: bool) {
        self.parameter_editor.set_open(open);
    }
    /// Report whether the editor needs parameter metadata.
    /// Example: `engine.parameters_open()` before reading parameter rows.
    pub fn parameters_open(&self) -> bool {
        self.parameter_editor.is_open()
    }
    /// Request current rows while a tool consumes them. Example: `engine.parameter_metadata_requested()`.
    pub fn parameter_metadata_requested(&self) -> bool {
        self.parameter_editor.needs_snapshot() || self.extrude_dialog.uses_parameter_metadata()
    }
    /// Return the outcome of a parameter edit to its control.
    /// Example: `engine.acknowledge_parameter_action(&action, result)`.
    pub fn acknowledge_parameter_action(
        &mut self,
        action: &crate::GuiParameterAction,
        result: Result<(), String>,
    ) {
        self.parameter_editor.acknowledge(action, result);
    }
    /// Report a document-units commit result to its draft; example: `engine.acknowledge_document_units(1, Ok(()))`.
    pub fn acknowledge_document_units(&mut self, document: u64, result: Result<(), String>) {
        self.preferences
            .acknowledge_document_units(document, result);
    }
    /// Check the pending settings before a document write; example: `engine.validate_constraint_settings(action)?`.
    pub fn validate_constraint_settings(
        &self,
        action: &crate::GuiShellAction,
    ) -> Result<(), String> {
        self.preferences.validate_constraint_settings(action)
    }
    /// Return the document settings result; example: `engine.acknowledge_constraint_settings(action, Some(1), Ok(()))`.
    pub fn acknowledge_constraint_settings(
        &mut self,
        action: &crate::GuiShellAction,
        active_document: Option<u64>,
        result: Result<(), String>,
    ) {
        self.preferences
            .acknowledge_constraint_settings(action, active_document, result);
    }
    /// Attach owner-thread clipboard I/O; example: `GuiFrameEngine::with_clipboard(clipboard)?`.
    pub fn with_clipboard(clipboard: crate::GuiClipboard) -> Result<Self, String> {
        let mut owner = Self::new()?;
        owner.context.set_clipboard_backend(clipboard);
        Ok(owner)
    }
    /// Retain the atlas independently of the next UI frame; example: `engine.atlas().clone()` for submission.
    pub fn atlas(&self) -> &GuiAtlas {
        &self.atlas
    }
    /// Retain the exact resource generation; example: `GuiFrameCompiler::with_resources(engine.resources())?`.
    pub fn resources(&self) -> &crate::GuiTextureResources {
        &self.resources
    }
    /// Build controls and copy every draw before returning to the owner.
    /// Example: `engine.frame(&request)?` returns draws, actions, and channel capture requests.
    pub fn frame(&mut self, request: &GuiFrameRequest) -> Result<GuiFrameOutput, String> {
        validate_request(request)?;
        self.document_files.validate()?;
        self.refresh_font_scale(request)?;
        if self.icons.prepare(request, &self.atlas)? {
            self.ribbon.panels = Default::default();
        }
        self.resources = self.icons.resources.clone();
        let restyled = self.refresh_chrome(request);
        self.external_docks.synchronize(&request.commands);
        let mut frame = self.draw_controls(request)?;
        frame.activity.pending_layout |= restyled;
        Ok(frame)
    }
    /// Focus an accessible control; example: `engine.focus_accessible(GuiAccessibleFocus::Frame(0))`.
    pub fn focus_accessible(&mut self, target: crate::GuiAccessibleFocus) {
        self.accessible_focus = Some(target);
    }
    /// Answer the active guarded-close prompt; example: `engine.answer_close_prompt(GuiClosePromptChoice::Cancel)`.
    pub fn answer_close_prompt(&mut self, choice: crate::GuiClosePromptChoice) {
        self.document_files.answer_close_prompt(choice);
    }
    /// Request a guarded application exit; example: `engine.request_application_close()`.
    pub fn request_application_close(&mut self) {
        self.document_files.request_application_close();
    }
    /// Show a captured window picture in Save As; example: after `CaptureWindowPreview`.
    pub fn receive_window_preview(&mut self, image: crate::GuiPreviewImage) {
        self.document_files.receive_window_preview(image);
    }
    /// Supply an isometric Save As preview for its document.
    /// Example: `engine.receive_isometric_preview(document, image)`.
    pub fn receive_isometric_preview(&mut self, document: u64, image: crate::GuiPreviewImage) {
        self.document_files
            .receive_isometric_preview(document, image);
    }
    /// Report a failed preview on the matching Save As dialog.
    /// Example: `engine.reject_isometric_preview(document, cause)`.
    pub fn reject_isometric_preview(&mut self, document: u64, cause: String) {
        self.document_files
            .reject_isometric_preview(document, cause);
    }
    /// Restore a rejected dimension draft; example: `engine.reject_dimension_edit(&value, "invalid")`.
    pub fn reject_dimension_edit(&mut self, value: &crate::GuiDimensionValue, cause: &str) {
        self.dimension_value_editor.reject(value, cause);
    }
    /// Restore an extrusion draft when the owner rejects it; example: `engine.reject_extrusion(&request, "profile missing")`.
    pub fn reject_extrusion(&mut self, request: &crate::GuiExtrudePreview, cause: &str) {
        self.extrude_dialog.reject(request, cause);
    }
    /// Restore an offset-plane draft rejected by the document owner.
    /// Example: `engine.reject_offset_plane(document, base, "bad", "invalid")`.
    pub fn reject_offset_plane(
        &mut self,
        document: u64,
        base: crate::GuiOffsetPlaneBase,
        offset: &str,
        cause: &str,
    ) {
        self.offset_plane.reject(document, base, offset, cause);
    }
    /// Restore a rejected extrusion edit draft; example: `engine.reject_extrusion_edit(&request, "invalid")`.
    pub fn reject_extrusion_edit(&mut self, request: &crate::GuiExtrudeEditRequest, cause: &str) {
        self.extrude_dialog.reject_edit(request, cause);
    }
    /// Request a path for the owner's unsaved document without writing it.
    /// Example: `engine.request_save_path(document)?` after a SavePathRequired notice.
    pub fn request_save_path(&mut self, document: u64) -> Result<(), String> {
        let name = self.document_name(document);
        self.document_files
            .request_save_path(document, &name, self.known_document_type)
    }
}
pub(crate) fn configure_owner_context(context: &mut Context) -> Result<GuiAtlas, String> {
    context.set_ini_filename(None);
    context.set_log_filename(None);
    context
        .io_mut()
        .config_flags
        .insert(ConfigFlags::DOCKING_ENABLE | ConfigFlags::NAV_ENABLE_KEYBOARD);
    crate::theme::configure_chrome(context.style_mut())?;
    Ok(crate::ui_scale::rebuild_font_atlas(context, 1.))
}
pub(crate) fn copy_atlas(context: &mut Context) -> GuiAtlas {
    let atlas = context.fonts().build_rgba32_texture();
    let rgba = atlas.data.as_chunks::<4>().0.to_vec();
    GuiAtlas {
        extent: [atlas.width, atlas.height],
        rgba: Arc::new(rgba),
    }
}
fn validate_request(request: &GuiFrameRequest) -> Result<(), String> {
    if !valid_frame_scalars(request) {
        return Err(format!(
            "GUI request {request:?}: expected positive finite extent, scale, and elapsed time"
        ));
    }
    validate_logical_inputs(request)?;
    crate::ui_scale::font_raster_scale(request)?;
    request.commands.validate()?;
    if request
        .status
        .as_ref()
        .is_some_and(|text| text.contains('\0'))
    {
        return Err(format!(
            "GUI status {:?}: expected text without NUL",
            request.status
        ));
    }
    for event in &request.events {
        validate_input(event, request.layout_scale())?;
    }
    Ok(())
}
fn valid_frame_scalars(request: &GuiFrameRequest) -> bool {
    !request.extent.contains(&0)
        && request.delta_seconds.is_finite()
        && request.delta_seconds > 0.
        && request
            .framebuffer_scale
            .iter()
            .all(|value| value.is_finite() && *value > 0.)
}
fn validate_logical_inputs(request: &GuiFrameRequest) -> Result<(), String> {
    for axis in 0..2 {
        let scale = request.layout_scale()[axis];
        let size = request.extent[axis] as f32 / scale;
        if !size.is_finite() || size <= 0. {
            return Err(format!(
                "GUI request {request:?}: expected positive finite logical extent"
            ));
        }
    }
    Ok(())
}

/// What the workspace shows, and where in window pixels, for the picture kept for its tab.
fn workspace_front(
    request: &GuiFrameRequest,
    area: crate::application::document_creation::workspaces::WorkspaceArea,
    home_active: bool,
) -> Option<(
    crate::application::document_tabs::previews::PreviewKey,
    [[f32; 2]; 2],
)> {
    use crate::application::document_tabs::previews::front_key;
    let scale = request.layout_scale();
    let bounds = [
        [area.left * scale[0], area.top * scale[1]],
        [area.right * scale[0], area.bottom * scale[1]],
    ];
    front_key(&request.commands, home_active).map(|key| (key, bounds))
}
impl GuiFrameEngine {
    /// Keep a window's picture for its tab; example: after each completed presentation.
    pub fn note_presented_frame(&mut self, extent: [u32; 2], rgba: &[u8]) {
        self.document_tabs.previews.note_frame(extent, rgba);
    }
}
/// Home is left by anything that puts a document in front of it; true when that changed.
fn home_follows_actions(
    home: &mut crate::application::home_recent_documents::HomeWorkspace,
    selected: &[GuiShellAction],
) -> bool {
    let before = home.active;
    for action in selected {
        match action {
            GuiShellAction::ShowHome => home.active = true,
            GuiShellAction::CloseDocumentWindow(_) if home.reached_from_home => {
                home.active = true;
            }
            GuiShellAction::ActivateDocument(_)
            | GuiShellAction::ActivateWindow(_)
            | GuiShellAction::OpenDocument(_)
            | GuiShellAction::OpenDeferredDrawing(_)
            | GuiShellAction::CreateDocument(_)
            | GuiShellAction::CreateProjectTemplate(_)
            | GuiShellAction::NewPart => home.active = false,
            _ => {}
        }
    }
    home.active != before
}
#[cfg(test)]
#[path = "../tests/parameter_journey/platform.rs"]
mod parameter_shortcuts;
#[path = "engine_resources.rs"]
mod resources;

#[cfg(test)]
#[path = "../tests/engine/work_plane.rs"]
mod selected_work_plane_tests;
