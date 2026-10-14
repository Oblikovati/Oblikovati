// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    GuiAtlas, GuiCommandState, GuiFrame, GuiFrameActivity, GuiInputCapture, GuiInputEvent,
    GuiShellAction,
    context_owner::{frame_activity, require_vacant_context},
    input::{apply_input, input_capture, validate_input},
    snapshot::{FONT_TEXTURE_ID, capture_frame},
};
use imgui::{Condition, ConfigFlags, Context, TextureId};
use std::sync::Arc;
#[derive(Clone, Debug)]
pub struct GuiFrameRequest {
    pub extent: [u32; 2],
    pub events: Vec<GuiInputEvent>,
    pub commands: GuiCommandState,
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
}
impl GuiFrameEngine {
    /// Create the GUI owner on its UI thread without automatic file I/O; example: `GuiFrameEngine::new()?`.
    pub fn new() -> Result<Self, String> {
        require_vacant_context()?;
        let mut context = Context::create();
        let atlas = configure_owner_context(&mut context);
        Ok(Self { context, atlas })
    }
    /// Retain the atlas independently of the next UI frame; example: `engine.atlas().clone()` for submission.
    pub fn atlas(&self) -> &GuiAtlas {
        &self.atlas
    }
    /// Build controls and copy every draw before returning to the owner.
    /// Example: `engine.frame(&request)?` returns draws, actions, and channel capture requests.
    pub fn frame(&mut self, request: &GuiFrameRequest) -> Result<GuiFrameOutput, String> {
        validate_request(request)?;
        draw_controls(&mut self.context, request)
    }
}

fn configure_owner_context(context: &mut Context) -> GuiAtlas {
    context.set_ini_filename(None);
    context.set_log_filename(None);
    context
        .io_mut()
        .config_flags
        .insert(ConfigFlags::DOCKING_ENABLE);
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
) -> Result<GuiFrameOutput, String> {
    prepare_frame_inputs(context, request);
    let ui = context.frame();
    let mut selected = Vec::new();
    shell_panel(ui, &request.commands, &mut selected);
    let snapshot = capture_frame(context.render(), request.extent)?;
    Ok(GuiFrameOutput {
        draws: snapshot,
        actions: selected,
        capture: input_capture(context.io()),
        activity: frame_activity()?,
    })
}
fn shell_panel(ui: &imgui::Ui, state: &GuiCommandState, selected: &mut Vec<GuiShellAction>) {
    ui.window("Oblikovati")
        .position([8., 8.], Condition::Always)
        .size([300., 110.], Condition::Always)
        .movable(false)
        .resizable(false)
        .build(|| shell_buttons(ui, state, selected));
}
fn shell_buttons(ui: &imgui::Ui, state: &GuiCommandState, selected: &mut Vec<GuiShellAction>) {
    for (label, action) in [
        ("Save", GuiShellAction::Save),
        ("Undo", GuiShellAction::Undo),
        ("Redo", GuiShellAction::Redo),
        ("Home", GuiShellAction::Home),
    ] {
        if shell_button(ui, label, action, state) {
            selected.push(action);
        }
        ui.same_line();
    }
    ui.new_line();
    ui.text("Document commands");
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

fn shell_button(
    ui: &imgui::Ui,
    label: &str,
    action: GuiShellAction,
    state: &GuiCommandState,
) -> bool {
    let disabled = ui.begin_disabled(!action.enabled(state));
    let clicked = ui.button(label);
    drop(disabled);
    if ui.is_item_hovered()
        && let Some(next) = action.next_label(state)
    {
        ui.tooltip_text(format!("{label} {next}"));
    }
    clicked && action.dispatchable(state)
}
