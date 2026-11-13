// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    GuiAtlas, GuiCommandState, GuiFrame, GuiFrameActivity, GuiInputCapture, GuiInputEvent,
    GuiShellAction,
    context_owner::{frame_activity, require_vacant_context},
    frame_keys::prepare_frame_inputs,
    input::{input_capture, validate_input},
    snapshot::{FONT_TEXTURE_ID, capture_frame},
};
use imgui::{ConfigFlags, Context, TextureId};
use std::sync::Arc;
#[derive(Clone, Debug)]
pub struct GuiFrameRequest {
    pub extent: [u32; 2],
    pub events: Vec<GuiInputEvent>,
    pub commands: GuiCommandState,
    pub status: Option<String>,
    pub delta_seconds: f32,
    pub framebuffer_scale: [f32; 2],
}
#[derive(Clone, Debug)]
pub struct GuiFrameOutput {
    pub draws: GuiFrame,
    pub actions: Vec<GuiShellAction>,
    pub capture: GuiInputCapture,
    pub activity: GuiFrameActivity,
}
/// One GUI thread retains the active context for its lifetime; packets contain no borrowed FFI pointers.
/// ```compile_fail
/// fn require_send<T: Send>() {}
/// require_send::<oblikovati_gui::GuiFrameEngine>();
/// ```
pub struct GuiFrameEngine {
    context: Context,
    atlas: GuiAtlas,
    resources: crate::GuiTextureResources,
    icons: crate::icons::GuiIconCache,
    raster_scale: f32,
    preferences: crate::ui_scale::GuiScalePreferences,
    extrude_dialog: crate::extrude_dialog::ExtrudeDialog,
    dimension_value_editor: crate::sketch::dimension_edit::GuiDimensionValueEditor,
    precise_input: crate::sketch::precise_input::PreciseInputBar,
    create_block: crate::sketch::create_block::CreateBlockDialog,
    sketch_pattern: crate::sketch::rectangular_pattern::RectangularPatternDialog,
    command_alias: crate::sketch::command_alias::CommandAliasBox,
    /// A popup was open last frame: Escape closes it, and the frame that opens is already too
    /// late to ask, since Dear ImGui acts on the key as the frame begins.
    popup_was_open: bool,
    /// Keys a widget received the press of, so their release reaches it too.
    widget_keys: Vec<crate::GuiKey>,
    quick_access: crate::application::quick_access_toolbar::QuickAccessToolbar,
    material_browser: crate::material_browser::MaterialBrowser,
    home: crate::application::home_recent_documents::HomeWorkspace,
    plane_selection: crate::application::quick_access_toolbar::plane_selection::PlaneSelection,
    parameter_editor: crate::parameters::GuiParameterEditor,
    document_files: crate::files::GuiDocumentFiles,
    ribbon: crate::ribbon::DocumentRibbon,
    document_tabs: crate::document_tabs::DocumentTabs,
    workspace: crate::application::workspace_layout::WorkspaceLayout,
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
        require_vacant_context()?;
        let mut context = Context::create();
        let atlas = configure_owner_context(&mut context);
        let resources = crate::GuiTextureResources::new(atlas.clone(), [])?;
        let icons = crate::icons::GuiIconCache::new(atlas.clone())?;
        Ok(Self {
            context,
            atlas,
            resources,
            icons,
            raster_scale: 1.,
            preferences: Default::default(),
            extrude_dialog: Default::default(),
            dimension_value_editor: Default::default(),
            precise_input: Default::default(),
            create_block: Default::default(),
            sketch_pattern: Default::default(),
            command_alias: Default::default(),
            popup_was_open: false,
            widget_keys: Vec::new(),
            quick_access: Default::default(),
            material_browser: Default::default(),
            home: Default::default(),
            plane_selection: Default::default(),
            parameter_editor: Default::default(),
            document_files: Default::default(),
            ribbon: Default::default(),
            document_tabs: Default::default(),
            workspace: Default::default(),
            orientation: Default::default(),
            known_documents: Vec::new(),
            known_document_type: None,
            workspaces: Default::default(),
            modifiers_held: Default::default(),
            theme_editor: Default::default(),
        })
    }
    /// Apply semantic chrome colours on the GUI thread; example: `engine.apply_theme(library.active())`.
    pub fn apply_theme(&mut self, theme: &oblikovati_model::theme::ThemeDefinition) {
        self.icons.set_theme(theme);
        crate::theme::apply_chrome_theme(&mut self.context.style_mut().colors, theme.palette());
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
        if let Some(theme) = &request.commands.theme {
            crate::theme::apply_chrome_theme(&mut self.context.style_mut().colors, theme.palette());
        }
        self.draw_controls(request)
    }
    /// Show a captured window picture in Save As; example: after `CaptureWindowPreview`.
    pub fn receive_window_preview(&mut self, image: crate::GuiPreviewImage) {
        self.document_files.receive_window_preview(image);
    }
    /// Restore a rejected dimension draft; example: `engine.reject_dimension_edit(&value, "invalid")`.
    pub fn reject_dimension_edit(&mut self, value: &crate::GuiDimensionValue, cause: &str) {
        self.dimension_value_editor.reject(value, cause);
    }
    /// Request a path for the owner's unsaved document without writing it.
    /// Example: `engine.request_save_path(document)?` after a SavePathRequired notice.
    pub fn request_save_path(&mut self, document: u64) -> Result<(), String> {
        let name = self.document_name(document);
        self.document_files
            .request_save_path(document, &name, self.known_document_type)
    }
}

impl GuiFrameEngine {
    fn document_name(&self, document: u64) -> String {
        self.known_documents
            .iter()
            .find(|row| row.id == document)
            .map_or_else(|| "Part".to_string(), |row| row.name.clone())
    }
}
fn configure_owner_context(context: &mut Context) -> GuiAtlas {
    context.set_ini_filename(None);
    context.set_log_filename(None);
    context
        .io_mut()
        .config_flags
        .insert(ConfigFlags::DOCKING_ENABLE | ConfigFlags::NAV_ENABLE_KEYBOARD);
    let atlas = copy_atlas(context);
    // imgui-rs e246a334 rebuilds and clears TexID; bind after the atlas is built.
    context.fonts().tex_id = TextureId::new(FONT_TEXTURE_ID);
    atlas
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
impl GuiFrameEngine {
    fn draw_controls(&mut self, request: &GuiFrameRequest) -> Result<GuiFrameOutput, String> {
        let keys = prepare_frame_inputs(
            &mut self.context,
            request,
            &mut self.modifiers_held,
            self.popup_was_open,
            self.ribbon.tips.active(),
            &mut self.widget_keys,
        );
        let ui = self.context.frame();
        crate::dimension_labels::draw_dimension_labels(ui, request);
        crate::sketch::constraint_glyphs::draw_constraint_glyphs(ui, request);
        crate::application::workspace_layout::browser_feature::draw_feature_edges(ui, request);
        self.workspace.draw_selected_origin(ui, request);
        let mut selected = Vec::new();
        self.document_files.home_in_front = self.home.active;
        let band_height = crate::chrome::ribbon_band(
            ui,
            request,
            crate::chrome::RibbonDrawOwners {
                home_active: self.home.active,
                icons: &self.icons.presentation,
                ribbon: &mut self.ribbon,
                parameters: &mut self.parameter_editor,
                files: &mut self.document_files,
                theme_editor: &mut self.theme_editor,
                preferences: &mut self.preferences,
                precise_input: &mut self.precise_input,
                create_block: &mut self.create_block,
                sketch_pattern: &mut self.sketch_pattern,
                workspace: &mut self.workspace,
                workspaces: &mut self.workspaces,
                quick_access: &mut self.quick_access,
                plane_selection_active: self.plane_selection.active,
            },
            &mut selected,
        );
        apply_quick_access(
            &mut self.quick_access,
            &mut self.document_files,
            &mut self.plane_selection,
            &mut self.extrude_dialog,
            &mut self.material_browser,
            &selected,
        );
        if !self.ribbon.tips.active() && !self.ribbon.tips.consumed {
            self.command_alias.panel(ui, &request.commands, &keys.typed);
        }
        crate::application::workspace_layout::status_bar::status_bar(
            ui,
            request,
            self.command_alias.text(),
        );
        let over_fields = crate::sketch::line::hud_panel(ui, request);
        self.document_tabs.panel(
            ui,
            request,
            band_height,
            &self.workspace,
            &mut self.home,
            &mut selected,
        );
        self.parameter_editor
            .panel(ui, &request.commands, &mut selected);
        self.workspace
            .panels
            .record_status(request.status.as_deref());
        self.workspace.view_clicks_taken = self.orientation.owns_view_clicks();
        self.workspace.browser_panel(
            ui,
            &request.commands,
            request.extent[1] as f32 / request.layout_scale()[1]
                - crate::application::workspace_layout::status_bar::STATUS_BAR_HEIGHT,
            band_height,
            &mut selected,
        );
        for action in &selected {
            if let GuiShellAction::OpenSketchPattern { sketch, pattern } = action {
                self.sketch_pattern
                    .open_edit(&request.commands, *sketch, *pattern);
            }
        }
        selected.retain(|action| !matches!(action, GuiShellAction::OpenSketchPattern { .. }));
        let logical =
            std::array::from_fn(|axis| request.extent[axis] as f32 / request.layout_scale()[axis]);
        let area = self.workspace.document_area(logical, band_height);
        // Over Home there is no view for it to turn.
        if !self.home.active {
            self.orientation.panel(ui, request, area, &mut selected);
        }
        let detached = self.document_tabs.is_detached(request.commands.document);
        self.workspaces
            .panel(ui, &request.commands, area, detached, &mut selected);
        self.plane_selection.panel(
            ui,
            &request.commands,
            request.layout_scale(),
            area,
            &mut selected,
        );
        self.document_tabs.previews.showing = workspace_front(request, area, self.home.active);
        self.document_tabs.previews.showing_document =
            request.commands.document.filter(|_| !self.home.active);
        if !self.home.active {
            crate::application::workspace_layout::browser_feature::selection_toolbar(
                ui,
                request,
                &mut selected,
            );
        }
        self.home.panel(ui, &request.commands, area, &mut selected);
        let home_requests = std::mem::take(&mut self.home.requests);
        if home_requests.new_document {
            self.document_files.request_new();
        }
        if home_requests.open_document {
            self.document_files.request_open();
        }
        if let Some(path) = home_requests.file_properties {
            self.document_tabs.menu.properties.request_file(&path);
        }
        self.document_files.panel(
            ui,
            &request.commands,
            &mut selected,
            &self.icons.presentation,
        );
        self.preferences
            .panel(ui, &request.commands, &mut selected, self.home.active);
        self.material_browser
            .panel(ui, &request.commands, &mut selected);
        self.extrude_dialog
            .panel(ui, &request.commands, &mut selected);
        self.ribbon
            .customization
            .panel(ui, &request.commands, &mut selected);
        self.create_block
            .panel(ui, &request.commands, &mut selected);
        self.sketch_pattern
            .panel(ui, &request.commands, &mut selected);
        crate::sketch::line::command_menu(ui, &request.commands, &mut selected);
        self.precise_input
            .panel(ui, &request.commands, &mut selected);
        crate::sketch::inspect::canvas_menu(ui, &request.commands, &mut selected);
        self.dimension_value_editor
            .panel(ui, request, &mut selected);
        crate::sketch::move_tool::panel(
            ui,
            &request.commands,
            &mut self.precise_input,
            &mut selected,
        );
        crate::sketch::inspect::measure_panel(ui, &request.commands, &mut selected);
        crate::sketch::region_properties::panel(ui, &request.commands, &mut selected);
        crate::sketch_delete::append_keyboard_delete(ui, &request.commands, &mut selected);
        crate::application::document_tabs::document_keys(
            ui,
            keys.switch_document,
            self.home.active,
            &request.commands,
            &mut selected,
        );
        self.workspace
            .switcher
            .panel(ui, &request.commands, &mut selected);
        if self.document_files.take_options_request() {
            self.preferences.request();
        }
        if self.document_files.take_properties_request()
            && let Some(id) = request.commands.document
        {
            self.document_tabs
                .menu
                .properties
                .request(&request.commands, id);
        }
        self.document_tabs
            .menu
            .properties
            .panel(ui, &request.commands);
        self.document_files
            .closing_panel(ui, &request.commands, &mut selected);
        let home_changed = home_follows_actions(&mut self.home, &selected);
        self.known_documents.clone_from(&request.commands.documents);
        self.known_document_type = request.commands.document_type;
        self.workspaces.note_created(&selected);
        // A popup that closed this frame still owned the key that closed it.
        let popup_open = self.popup_was_open || crate::widget_abi::any_popup_open(ui);
        self.popup_was_open = crate::widget_abi::any_popup_open(ui);
        // A dialog over the workspace is not a picture of the document, nor of its tab.
        if popup_open {
            self.document_tabs.previews.showing = None;
        }
        selected.extend(self.document_tabs.previews.take_changed_picture().map(
            |(document, picture)| GuiShellAction::KeepDocumentPicture {
                document,
                picture: Some(picture),
            },
        ));
        let snapshot = capture_frame(self.context.render(), request.extent, &self.resources)?;
        let mut capture = input_capture(self.context.io());
        capture.keyboard &= !keys.tool_keyboard;
        // Dear ImGui claims the keyboard for whichever window last had focus. A click on the
        // ribbon must leave M and F8 to the sketch unless a widget is really using the keys.
        // A popup always owns them, even on the frame Escape closes it and focus moves on.
        capture.keyboard &= capture.text || self.context.io().nav_visible;
        capture.keyboard |= popup_open
            || self.ribbon.tips.consumed
            || self.workspace.browser_consumed_arrow()
            || self.ribbon.customization.navigation_consumed;
        let arrow_press = request.events.iter().any(|event| {
            matches!(
                event,
                GuiInputEvent::Key {
                    key: crate::GuiKey::LeftArrow
                        | crate::GuiKey::RightArrow
                        | crate::GuiKey::UpArrow
                        | crate::GuiKey::DownArrow,
                    pressed: true
                }
            )
        });
        if arrow_press
            && self.ribbon.tips.active()
            && !capture.text
            && !popup_open
            && !self.workspace.browser_consumed_arrow()
        {
            capture.keyboard = false;
        }
        if arrow_press && self.workspace.feature_keyboard_focus() {
            capture.keyboard = true;
        }
        capture.text |= self.ribbon.tips.consumed
            || (self.ribbon.tips.active()
                && request
                    .events
                    .iter()
                    .any(|event| matches!(event, GuiInputEvent::Text(_))));
        capture.popup = popup_open
            || self.dimension_value_editor.had_keyboard()
            || self.ribbon.tips.consumed
            || self.ribbon.customization.escape_consumed
            || self.ribbon.display.temporary_escape_consumed;
        capture.pointer |= self.dimension_value_editor.over_label || over_fields;
        let mut activity = frame_activity()?;
        activity.pending_layout |= self.orientation.needs_frame();
        activity.pending_layout |= self.ribbon.panels.pending_layout || self.ribbon.needs_frame();
        activity.pending_layout |=
            self.workspace.pending_layout || home_changed || self.home.hover_changed;
        activity.pending_layout |= self.document_files.needs_frame()
            || self.document_tabs.needs_frame()
            || self.workspaces.needs_frame()
            || self.workspace.switcher.needs_frame()
            || self.document_tabs.menu.properties.needs_frame();
        Ok(GuiFrameOutput {
            draws: snapshot,
            actions: selected,
            capture,
            activity,
        })
    }
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
    use crate::application::document_tabs::previews::PreviewKey;
    let scale = request.layout_scale();
    let bounds = [
        [area.left * scale[0], area.top * scale[1]],
        [area.right * scale[0], area.bottom * scale[1]],
    ];
    if home_active {
        return Some((PreviewKey::Home, bounds));
    }
    let document = request.commands.document?;
    let window = request.commands.active_window.unwrap_or(document);
    Some((PreviewKey::Window(window), bounds))
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
            | GuiShellAction::CreateDocument(_)
            | GuiShellAction::NewPart => home.active = false,
            _ => {}
        }
    }
    home.active != before
}
/// The toolbar and ribbon ask; the owners that can answer are the engine's other parts.
fn apply_quick_access(
    toolbar: &mut crate::application::quick_access_toolbar::QuickAccessToolbar,
    files: &mut crate::files::GuiDocumentFiles,
    planes: &mut crate::application::quick_access_toolbar::plane_selection::PlaneSelection,
    extrude: &mut crate::extrude_dialog::ExtrudeDialog,
    material_browser: &mut crate::material_browser::MaterialBrowser,
    selected: &[GuiShellAction],
) {
    let requests = toolbar.take_requests();
    if requests.new_document {
        files.request_new();
    }
    if requests.open_document {
        files.request_open();
    }
    if requests.open_material_browser {
        material_browser.open();
    }
    planes.active |= requests.choose_plane || selected.contains(&GuiShellAction::ChooseSketchPlane);
    extrude.requested |= requests.open_extrude || selected.contains(&GuiShellAction::OpenExtrude);
}
#[cfg(test)]
#[path = "../tests/parameter_journey/platform.rs"]
mod parameter_shortcuts;

#[path = "engine_resources.rs"]
mod resources;
