// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    GuiAtlas, GuiCommandState, GuiFrame, GuiFrameActivity, GuiInputCapture, GuiInputEvent,
    GuiShellAction,
    context_owner::{frame_activity, require_vacant_context},
    input::{apply_input, input_capture, tool_transaction_key, validate_input},
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
    raster_scale: f32,
    preferences: crate::ui_scale::GuiScalePreferences,
    dimension_editor: crate::dimensions::GuiDimensionEditor,
    parameter_editor: crate::parameters::GuiParameterEditor,
    document_files: crate::files::GuiDocumentFiles,
    ribbon: crate::ribbon::DocumentRibbon,
    document_tabs: crate::document_tabs::DocumentTabs,
    theme_editor: crate::theme_editor::GuiThemeEditor,
}
impl GuiFrameEngine {
    /// Create the GUI owner on its UI thread without automatic file I/O; example: `GuiFrameEngine::new()?`.
    pub fn new() -> Result<Self, String> {
        require_vacant_context()?;
        let mut context = Context::create();
        let atlas = configure_owner_context(&mut context);
        Ok(Self {
            context,
            atlas,
            raster_scale: 1.,
            preferences: Default::default(),
            dimension_editor: Default::default(),
            parameter_editor: Default::default(),
            document_files: Default::default(),
            ribbon: Default::default(),
            document_tabs: Default::default(),
            theme_editor: Default::default(),
        })
    }
    /// Apply semantic chrome colours on the GUI thread; example: `engine.apply_theme(library.active())`.
    pub fn apply_theme(&mut self, theme: &oblikovati_model::theme::ThemeDefinition) {
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
    /// Build controls and copy every draw before returning to the owner.
    /// Example: `engine.frame(&request)?` returns draws, actions, and channel capture requests.
    pub fn frame(&mut self, request: &GuiFrameRequest) -> Result<GuiFrameOutput, String> {
        validate_request(request)?;
        self.document_files.validate()?;
        self.refresh_font_scale(request)?;
        if let Some(theme) = &request.commands.theme {
            crate::theme::apply_chrome_theme(&mut self.context.style_mut().colors, theme.palette());
        }
        self.draw_controls(request)
    }
    /// Request a path for the owner's unsaved document without writing it.
    /// Example: `engine.request_save_path(document)?` after a SavePathRequired notice.
    pub fn request_save_path(&mut self, document: u64) -> Result<(), String> {
        self.document_files.request_save_path(document)
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
        let tool_keyboard = prepare_frame_inputs(&mut self.context, request);
        let ui = self.context.frame();
        crate::dimension_labels::draw_dimension_labels(ui, request);
        let mut selected = Vec::new();
        let band_height = crate::chrome::ribbon_band(
            ui,
            request,
            crate::chrome::RibbonDrawOwners {
                ribbon: &mut self.ribbon,
                parameters: &mut self.parameter_editor,
                files: &mut self.document_files,
                theme_editor: &mut self.theme_editor,
                preferences: &mut self.preferences,
            },
            &mut selected,
        );
        self.document_tabs
            .panel(ui, request, band_height, &mut selected);
        self.parameter_editor
            .panel(ui, &request.commands, &mut selected);
        crate::browser::panel(
            ui,
            &request.commands,
            request.extent[1] as f32 / request.layout_scale()[1],
            band_height,
            &mut selected,
        );
        self.document_files
            .panel(ui, &request.commands, &mut selected);
        self.preferences.panel(ui, &request.commands, &mut selected);
        self.dimension_editor
            .panel(ui, &request.commands, &mut selected);
        crate::sketch_delete::append_keyboard_delete(ui, &request.commands, &mut selected);
        let snapshot = capture_frame(self.context.render(), request.extent)?;
        let mut capture = input_capture(self.context.io());
        capture.keyboard &= !tool_keyboard;
        let mut activity = frame_activity()?;
        activity.pending_layout |= self.ribbon.panels.pending_layout;
        Ok(GuiFrameOutput {
            draws: snapshot,
            actions: selected,
            capture,
            activity,
        })
    }
}
fn prepare_frame_inputs(context: &mut Context, request: &GuiFrameRequest) -> bool {
    let io = context.io_mut();
    io.display_size =
        std::array::from_fn(|axis| request.extent[axis] as f32 / request.layout_scale()[axis]);
    io.display_framebuffer_scale = request.layout_scale();
    io.delta_time = request.delta_seconds;
    let mut tool_keyboard = false;
    for event in &request.events {
        if tool_transaction_key(
            event,
            request.commands.sketch_tool_transaction,
            io.want_text_input,
        ) {
            tool_keyboard = true;
            continue;
        }
        apply_input(io, event, request.layout_scale());
    }
    tool_keyboard
}

#[cfg(test)]
#[path = "../tests/parameter_journey/platform.rs"]
mod parameter_shortcuts;

impl GuiFrameRequest {
    /// Convert UI layout units to backing pixels; native backing density stays independently recorded.
    /// Example: backing 1 and content 2 produce `layout_scale() == [2.; 2]`.
    pub(crate) fn layout_scale(&self) -> [f32; 2] {
        let content = self.commands.ui_scale.unwrap_or_default().font();
        self.framebuffer_scale.map(|backing| backing * content)
    }
}
impl GuiFrameEngine {
    fn refresh_font_scale(&mut self, request: &GuiFrameRequest) -> Result<(), String> {
        let scale = crate::ui_scale::font_raster_scale(request)?;
        if self.raster_scale == scale {
            return Ok(());
        }
        self.atlas = crate::ui_scale::rebuild_font_atlas(&mut self.context, scale);
        self.raster_scale = scale;
        self.ribbon.panels = Default::default();
        Ok(())
    }
}

#[cfg(test)]
#[path = "../tests/ui_scale/engine.rs"]
mod scale_tests;
