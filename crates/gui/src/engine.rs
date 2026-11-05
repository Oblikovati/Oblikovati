// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    GuiAtlas, GuiCommandState, GuiFrame, GuiFrameActivity, GuiInputCapture, GuiInputEvent,
    GuiShellAction,
    context_owner::{frame_activity, require_vacant_context},
    input::{apply_input, input_capture, validate_input},
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
    parameter_editor: crate::parameters::GuiParameterEditor,
    document_files: crate::files::GuiDocumentFiles,
    ribbon: crate::ribbon::DocumentRibbon,
    document_tabs: crate::document_tabs::DocumentTabs,
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
            parameter_editor: Default::default(),
            document_files: Default::default(),
            ribbon: Default::default(),
            document_tabs: Default::default(),
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
        if let Some(theme) = &request.commands.theme {
            crate::theme::apply_chrome_theme(&mut self.context.style_mut().colors, theme.palette());
        }
        draw_controls(
            &mut self.context,
            request,
            &mut self.parameter_editor,
            &mut self.document_files,
            &mut self.ribbon,
            &mut self.document_tabs,
        )
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
fn copy_atlas(context: &mut Context) -> GuiAtlas {
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
        validate_input(event, request.framebuffer_scale)?;
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
        let scale = request.framebuffer_scale[axis];
        let size = request.extent[axis] as f32 / scale;
        if !size.is_finite() || size <= 0. {
            return Err(format!(
                "GUI request {request:?}: expected positive finite logical extent"
            ));
        }
    }
    Ok(())
}
fn draw_controls(
    context: &mut Context,
    request: &GuiFrameRequest,
    parameter_editor: &mut crate::parameters::GuiParameterEditor,
    document_files: &mut crate::files::GuiDocumentFiles,
    ribbon: &mut crate::ribbon::DocumentRibbon,
    document_tabs: &mut crate::document_tabs::DocumentTabs,
) -> Result<GuiFrameOutput, String> {
    prepare_frame_inputs(context, request);
    let ui = context.frame();
    let mut selected = Vec::new();
    crate::chrome::ribbon_band(
        ui,
        request,
        ribbon,
        parameter_editor,
        document_files,
        &mut selected,
    );
    document_tabs.panel(ui, request, &mut selected);
    parameter_editor.panel(ui, &request.commands, &mut selected);
    crate::browser::panel(
        ui,
        &request.commands,
        request.extent[1] as f32 / request.framebuffer_scale[1],
        &mut selected,
    );
    document_files.panel(ui, &request.commands, &mut selected);
    let snapshot = capture_frame(context.render(), request.extent)?;
    Ok(GuiFrameOutput {
        draws: snapshot,
        actions: selected,
        capture: input_capture(context.io()),
        activity: frame_activity()?,
    })
}
fn prepare_frame_inputs(context: &mut Context, request: &GuiFrameRequest) {
    let io = context.io_mut();
    io.display_size =
        std::array::from_fn(|axis| request.extent[axis] as f32 / request.framebuffer_scale[axis]);
    io.display_framebuffer_scale = request.framebuffer_scale;
    io.delta_time = request.delta_seconds;
    for event in &request.events {
        apply_input(io, event, request.framebuffer_scale);
    }
}

#[cfg(test)]
#[path = "../tests/parameter_journey/platform.rs"]
mod parameter_shortcuts;
