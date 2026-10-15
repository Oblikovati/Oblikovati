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
mod workspace_shell;
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
    pub parameter_actions: Vec<crate::GuiParameterAction>,
}
/// One GUI thread retains the active context for its lifetime; packets contain no borrowed FFI pointers.
/// ```compile_fail
/// fn require_send<T: Send>() {}
/// require_send::<oblikovati_gui::GuiFrameEngine>();
/// ```
pub struct GuiFrameEngine {
    context: Context,
    atlas: GuiAtlas,
    parameter_panel: crate::parameter_panel::ParameterPanel,
    workspace: workspace_shell::WorkspaceShellState,
}
impl GuiFrameEngine {
    /// Create the GUI owner on its UI thread without automatic file I/O; example: `GuiFrameEngine::new()?`.
    pub fn new() -> Result<Self, String> {
        require_vacant_context()?;
        let mut context = Context::create();
        let atlas = configure_owner_context(&mut context);
        let parameter_panel = crate::parameter_panel::ParameterPanel::new()?;
        Ok(Self {
            context,
            atlas,
            parameter_panel,
            workspace: Default::default(),
        })
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
        draw_controls(
            &mut self.context,
            &mut self.parameter_panel,
            &mut self.workspace,
            request,
        )
    }
    /// Publish a copied parameter view; example: `engine.set_parameter_snapshot(Some(snapshot))?`.
    pub fn set_parameter_snapshot(
        &mut self,
        snapshot: Option<crate::GuiParameterSnapshot>,
    ) -> Result<bool, String> {
        self.parameter_panel.replace_snapshot(snapshot)
    }
    /// Show a failed view without retaining stale edit targets; example: `engine.report_parameter_view_issue(issue)?`.
    pub fn report_parameter_view_issue(&mut self, issue: String) -> Result<bool, String> {
        self.parameter_panel.report_metadata_issue(issue)
    }
    /// Open the native parameter editor on this GUI owner; example: `engine.set_parameters_open(true)`.
    pub fn set_parameters_open(&mut self, open: bool) {
        self.parameter_panel.set_open(open);
    }
    /// Query whether the panel needs model metadata; example: `engine.parameters_open()` before publishing rows.
    pub fn parameters_open(&self) -> bool {
        self.parameter_panel.is_open()
    }
    /// Return a widget result; example: `engine.acknowledge_parameter_action(&action, result)`.
    pub fn acknowledge_parameter_action(
        &mut self,
        action: &crate::GuiParameterAction,
        result: Result<(), String>,
    ) {
        self.parameter_panel.acknowledge_action(action, result);
    }
    /// Resolve the matching text edit without borrowing model state into widgets.
    /// Example: `engine.acknowledge_parameter_edit(&edit, result)` after the owner processes it.
    pub fn acknowledge_parameter_edit(
        &mut self,
        edit: &crate::GuiParameterTextEdit,
        result: Result<(), String>,
    ) -> crate::GuiParameterAck {
        self.parameter_panel.acknowledge(edit, result)
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
    parameters: &mut crate::parameter_panel::ParameterPanel,
    workspace: &mut workspace_shell::WorkspaceShellState,
    request: &GuiFrameRequest,
) -> Result<GuiFrameOutput, String> {
    prepare_frame_inputs(context, request);
    let ui = context.frame();
    let mut selected = Vec::new();
    workspace_shell::draw(
        ui,
        &request.commands,
        request.status.as_deref(),
        &mut selected,
        parameters,
        workspace,
    );
    let parameter_actions = parameters.draw(ui);
    let snapshot = capture_frame(context.render(), request.extent)?;
    Ok(GuiFrameOutput {
        draws: snapshot,
        actions: selected,
        capture: input_capture(context.io()),
        activity: frame_activity()?,
        parameter_actions,
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
