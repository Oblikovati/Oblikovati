// SPDX-License-Identifier: GPL-2.0-only
use super::NativeHeadWindow;
use crate::HeadGuiUpdate;
use oblikovati_gui::{
    GuiClipboard, GuiCommandState, GuiFrameActivity, GuiFrameEngine, GuiFrameOutput,
    GuiFrameRequest, GuiInputEvent,
};
use oblikovati_viewport::{GuiFrame, GuiFrameCompiler, ViewportEvent};
use std::time::Duration;
mod input;
// Go head/cmd/oblikovati-head/main.go uses animationTickSeconds = 1/60.
const GUI_TICK: Duration = Duration::from_nanos(1_000_000_000 / 60);
pub(super) struct NativeGui {
    engine: GuiFrameEngine,
    compiler: GuiFrameCompiler,
    textures: oblikovati_gui::GuiTextureResources,
    draws: Option<GuiFrame>,
    document_area: Option<([u32; 2], Option<[u32; 4]>)>,
    commands: Option<GuiCommandState>,
    notice: Option<String>,
    last_frame: Option<Duration>,
    deadline: Option<Duration>,
    activity: GuiFrameActivity,
    dirty: bool,
    parameter_changed: bool,
    focused: bool,
    mapped: bool,
    pointer: input::PointerOwnership,
    last_capture: oblikovati_gui::GuiInputCapture,
    window_controls: Option<oblikovati_gui::GuiWindowControls>,
    close_prompt: Option<oblikovati_gui::GuiClosePrompt>,
    accessibility_popup_active: bool,
    accessibility_focus_frames: u8,
    clipboard: Option<GuiClipboard>,
}
impl NativeGui {
    pub(super) fn frame_controls(
        &self,
    ) -> Option<(
        oblikovati_gui::GuiWindowControls,
        bool,
        bool,
        Option<oblikovati_gui::GuiClosePrompt>,
    )> {
        Some((
            self.window_controls.clone()?,
            self.accessibility_popup_active,
            self.focused,
            self.close_prompt.clone(),
        ))
    }
    pub(super) fn focus_accessible(&mut self, target: oblikovati_gui::GuiAccessibleFocus) {
        self.engine.focus_accessible(target);
        self.parameter_changed = true;
        self.accessibility_focus_frames = 2;
        self.deadline = Some(Duration::ZERO);
    }
    pub(super) fn answer_close_prompt(&mut self, choice: oblikovati_gui::GuiClosePromptChoice) {
        self.engine.answer_close_prompt(choice);
        self.parameter_changed = true;
        self.deadline = Some(Duration::ZERO);
    }
    pub(super) fn request_application_close(&mut self) {
        self.engine.request_application_close();
        self.parameter_changed = true;
    }

    pub(super) fn request_extrusion(&mut self) {
        self.engine.request_extrusion();
        self.deadline = Some(Duration::ZERO);
    }
    pub(super) fn reject_dimension_edit(
        &mut self,
        value: &oblikovati_gui::GuiDimensionValue,
        cause: &str,
    ) {
        self.engine.reject_dimension_edit(value, cause);
        self.deadline = Some(Duration::ZERO);
    }
    pub(super) fn reject_extrusion(
        &mut self,
        request: &oblikovati_gui::GuiExtrudePreview,
        cause: &str,
    ) {
        self.engine.reject_extrusion(request, cause);
        self.deadline = Some(Duration::ZERO);
    }
    pub(super) fn reject_offset_plane(
        &mut self,
        document: u64,
        base: oblikovati_gui::GuiOffsetPlaneBase,
        offset: &str,
        cause: &str,
    ) {
        self.engine
            .reject_offset_plane(document, base, offset, cause);
        self.deadline = Some(Duration::ZERO);
    }
    pub(super) fn reject_extrusion_edit(
        &mut self,
        request: &oblikovati_gui::GuiExtrudeEditRequest,
        cause: &str,
    ) {
        self.engine.reject_extrusion_edit(request, cause);
        self.deadline = Some(Duration::ZERO);
    }
    pub(super) fn request_save_path(&mut self, document: u64) -> Result<(), String> {
        self.engine.request_save_path(document)?;
        self.deadline = Some(Duration::ZERO);
        Ok(())
    }
    pub(super) fn note_presented_frame(&mut self, extent: [u32; 2], rgba: &[u8]) {
        self.engine.note_presented_frame(extent, rgba);
    }
    pub(super) fn receive_window_preview(&mut self, image: oblikovati_gui::GuiPreviewImage) {
        self.engine.receive_window_preview(image);
        self.deadline = Some(Duration::ZERO);
    }
    pub(super) fn receive_isometric_preview(
        &mut self,
        document: u64,
        image: oblikovati_gui::GuiPreviewImage,
    ) {
        self.engine.receive_isometric_preview(document, image);
        self.deadline = Some(Duration::ZERO);
    }
    pub(super) fn reject_isometric_preview(&mut self, document: u64, cause: String) {
        self.engine.reject_isometric_preview(document, cause);
        self.deadline = Some(Duration::ZERO);
    }
    pub(super) fn parameters_open(&self) -> bool {
        self.engine.parameters_open()
    }
    pub(super) fn set_parameters(
        &mut self,
        snapshot: Option<oblikovati_gui::GuiParameterSnapshot>,
    ) -> Result<bool, String> {
        let changed = self.engine.set_parameter_snapshot(snapshot)?;
        self.parameter_changed |= changed;
        Ok(changed)
    }
    pub(super) fn report_parameter_issue(&mut self, issue: String) -> Result<bool, String> {
        let changed = self.engine.report_parameter_view_issue(issue)?;
        self.parameter_changed |= changed;
        Ok(changed)
    }
    pub(super) fn acknowledge_parameter(
        &mut self,
        action: &oblikovati_gui::GuiParameterAction,
        result: Result<(), String>,
    ) {
        self.engine.acknowledge_parameter_action(action, result);
        self.parameter_changed = true;
        self.deadline = Some(self.last_frame.unwrap_or_default());
    }
    pub(super) fn require_publication(&mut self) {
        self.dirty = true;
    }
    fn new(clipboard: Option<GuiClipboard>) -> Result<Self, String> {
        let engine = match &clipboard {
            Some(clipboard) => GuiFrameEngine::with_clipboard(clipboard.clone())?,
            None => GuiFrameEngine::new()?,
        };
        let compiler = GuiFrameCompiler::with_resources(engine.resources())?;
        Ok(Self::with_compiler(engine, compiler, clipboard))
    }
    fn with_compiler(
        engine: GuiFrameEngine,
        compiler: GuiFrameCompiler,
        clipboard: Option<GuiClipboard>,
    ) -> Self {
        let textures = engine.resources().clone();
        Self {
            engine,
            textures,
            compiler,
            draws: None,
            document_area: None,
            commands: None,
            notice: None,
            last_frame: None,
            deadline: None,
            activity: Default::default(),
            dirty: false,
            parameter_changed: false,
            focused: true,
            mapped: true,
            pointer: Default::default(),
            last_capture: Default::default(),
            window_controls: None,
            close_prompt: None,
            accessibility_popup_active: false,
            accessibility_focus_frames: 0,
            clipboard,
        }
    }
    fn step(
        &mut self,
        now: Duration,
        extent: [u32; 2],
        event: Option<&ViewportEvent>,
        commands: &GuiCommandState,
        notice: Option<&str>,
    ) -> Result<HeadGuiUpdate, String> {
        self.observe_lifecycle(event);
        if !self.frame_due(now, extent, event, commands, notice) {
            return Ok(HeadGuiUpdate {
                wake_after: self.wake_after(now),
                ..Default::default()
            });
        }
        let request = self.request(now, extent, event, commands, notice)?;
        let output = self.frame_input(&request)?;
        self.refresh_gui_resources()?;
        self.dirty |= self.draws.as_ref() != Some(&output.draws);
        self.draws = Some(output.draws);
        self.document_area = Some((extent, output.document_area));
        self.parameter_changed = false;
        self.commands = Some(commands.clone());
        self.notice = notice.map(str::to_owned);
        self.last_frame = Some(now);
        self.activity = output.activity;
        self.last_capture = output.capture;
        self.window_controls = Some(output.window_controls);
        self.close_prompt = output.close_prompt;
        self.accessibility_popup_active = output.accessibility_popup_active;
        self.accessibility_focus_frames = self.accessibility_focus_frames.saturating_sub(1);
        self.deadline = (self.activity.needs_frame() || self.accessibility_focus_frames > 0)
            .then(|| now.saturating_add(GUI_TICK));
        Ok(HeadGuiUpdate {
            window_action: output.window_action,
            consumed_input: self.pointer.consumed(event, output.capture),
            actions: output.actions,
            parameter_actions: output.parameter_actions,
            constraint_hover: Some(output.constraint_hover),
            wake_after: self.wake_after(now),
            ..Default::default()
        })
    }
    pub(super) fn allows_ready_sketch_motion(&self, [x, y]: [i32; 2]) -> bool {
        if !self.focused
            || !self.mapped
            || self.dirty
            || self.activity.needs_frame()
            || !self.pointer.idle()
            || self.last_capture.pointer
            || self.last_capture.popup
            || self.last_capture.text
            || self.last_capture.keyboard
        {
            return false;
        }
        let Some((_, Some([left, top, right, bottom]))) = self.document_area else {
            return false;
        };
        let Ok(x) = u32::try_from(x) else {
            return false;
        };
        let Ok(y) = u32::try_from(y) else {
            return false;
        };
        (left..right).contains(&x) && (top..bottom).contains(&y)
    }
    pub(super) fn document_area(&self, extent: [u32; 2]) -> Result<[u32; 4], String> {
        let (observed, area) = self
            .document_area
            .ok_or("document area: expected a completed GUI layout")?;
        if observed != extent {
            return Err(format!(
                "document area extent {observed:?}: expected current drawable {extent:?}"
            ));
        }
        area.ok_or_else(|| {
            format!("document area {area:?}: expected a nonempty rectangle in {extent:?}")
        })
    }
    fn frame_input(&mut self, request: &GuiFrameRequest) -> Result<GuiFrameOutput, String> {
        let [GuiInputEvent::PointerMoved(_), GuiInputEvent::Wheel(_)] = request.events.as_slice()
        else {
            return self.engine.frame(request);
        };
        // #3630: ImGui can defer a wheel's new pointer position; keep click-queue trickling intact.
        let mut moved = request.clone();
        moved.events.truncate(1);
        let mut first = self.engine.frame(&moved)?;
        let mut wheeled = request.clone();
        wheeled.events.remove(0);
        wheeled.delta_seconds = Duration::from_nanos(1).as_secs_f32();
        let mut final_frame = self.engine.frame(&wheeled)?;
        first.actions.append(&mut final_frame.actions);
        final_frame.actions = first.actions;
        first
            .parameter_actions
            .append(&mut final_frame.parameter_actions);
        final_frame.parameter_actions = first.parameter_actions;
        Ok(final_frame)
    }
    fn observe_lifecycle(&mut self, event: Option<&ViewportEvent>) {
        match event {
            Some(ViewportEvent::Focus(focused)) => self.focused = *focused,
            Some(ViewportEvent::Mapped(mapped)) => self.mapped = *mapped,
            _ => {}
        }
    }
    fn frame_due(
        &self,
        now: Duration,
        extent: [u32; 2],
        event: Option<&ViewportEvent>,
        commands: &GuiCommandState,
        notice: Option<&str>,
    ) -> bool {
        self.parameter_changed
            || event.is_some()
            || self
                .draws
                .as_ref()
                .is_none_or(|draws| draws.extent != extent)
            || self.commands.as_ref() != Some(commands)
            || self.notice.as_deref() != notice
            || self.wake_after(now) == Some(Duration::ZERO)
    }
    fn wake_after(&self, now: Duration) -> Option<Duration> {
        if !self.mapped || !self.focused {
            return None;
        }
        self.deadline.map(|deadline| deadline.saturating_sub(now))
    }
    fn request(
        &self,
        now: Duration,
        extent: [u32; 2],
        event: Option<&ViewportEvent>,
        commands: &GuiCommandState,
        notice: Option<&str>,
    ) -> Result<GuiFrameRequest, String> {
        let elapsed = match self.last_frame {
            Some(previous) => now.checked_sub(previous).ok_or_else(|| {
                format!("GUI time {now:?}: expected monotonic time after {previous:?}")
            })?,
            None => GUI_TICK,
        };
        // Several native events can share a timestamp; Dear ImGui requires positive elapsed time.
        let delta_seconds = elapsed.max(Duration::from_nanos(1)).as_secs_f32();
        Ok(GuiFrameRequest {
            extent,
            events: input::copied_gui_events(event),
            commands: commands.clone(),
            status: notice.map(str::to_owned),
            delta_seconds,
            framebuffer_scale: [1.; 2],
        })
    }
}
impl NativeHeadWindow {
    fn ensure_native_gui(&mut self) -> Result<(), String> {
        if self.gui.is_some() {
            return Ok(());
        }
        let clipboard =
            GuiClipboard::new(self.clipboard.clone(), super::clipboard::CLIPBOARD_BYTES)?;
        let mut gui = NativeGui::new(Some(clipboard))?;
        if let Some(directory) = &self.ribbon_settings_directory {
            gui.engine.set_ribbon_settings_store(Box::new(
                super::preferences::NativeRibbonSettings::new(directory),
            ))?;
        }
        if let Some(crate::DesktopCommandNotice::SavePathRequired { document }) =
            &self.command_notice
        {
            gui.request_save_path(*document)?;
            self.command_notice = None;
        }
        self.gui = Some(gui);
        self.view.set_status_text("")?;
        self.refresh_title()?;
        Ok(())
    }
    fn native_gui_notice(&self) -> Result<Option<String>, String> {
        let mut messages = self
            .command_notice
            .as_ref()
            .map(crate::DesktopCommandNotice::message)
            .into_iter()
            .collect::<Vec<_>>();
        if let Some(issue) = self.view.text_input_issue()? {
            messages.push(format!("Text input unavailable: {issue}"));
        }
        if let Some(issue) = self.native_clipboard_issue() {
            messages.push(format!("Clipboard unavailable: {issue}"));
        }
        Ok((!messages.is_empty()).then(|| messages.join("\n")))
    }
    fn native_clipboard_issue(&self) -> Option<String> {
        self.clipboard.issue().or_else(|| {
            self.gui
                .as_ref()
                .and_then(|gui| gui.clipboard.as_ref())
                .and_then(GuiClipboard::issue)
        })
    }
    pub(super) fn advance_native_gui(
        &mut self,
        event: Option<&ViewportEvent>,
        commands: &GuiCommandState,
    ) -> Result<HeadGuiUpdate, String> {
        if self.gui.is_none() && event == Some(&ViewportEvent::CloseRequested) {
            return Ok(Default::default());
        }
        let extent = self.view.drawable_extent()?;
        self.ensure_native_gui()?;
        let now = self.gui_epoch.elapsed();
        let notice = self.native_gui_notice()?;
        let gui = self.gui.as_mut().expect("initialized GUI");
        let mut update = gui.step(now, extent, event, commands, notice.as_deref())?;
        if gui.dirty {
            publish_gui(&mut self.view, gui, now, &mut update)?;
        }
        Ok(update)
    }
}
fn publish_gui(
    view: &mut oblikovati_viewport::Viewport,
    gui: &mut NativeGui,
    now: Duration,
    update: &mut HeadGuiUpdate,
) -> Result<(), String> {
    let draws = gui.draws.as_ref().expect("dirty GUI has owned draws");
    match view.submit_gui_frame_checked(&gui.compiler, draws)? {
        oblikovati_viewport::ViewportGuiSubmission::Accepted => {
            gui.dirty = false;
            update.changed = true;
        }
        oblikovati_viewport::ViewportGuiSubmission::SurfaceChanged(_) => {
            gui.deadline = Some(now);
            update.deferred = true;
            update.wake_after = Some(Duration::ZERO);
        }
    }
    Ok(())
}

#[cfg(all(test, target_os = "linux"))]
#[path = "../../../tests/native_clipboard_health.rs"]
mod clipboard_health_tests;
#[cfg(test)]
#[path = "../../../tests/native_input_method.rs"]
mod provider_tests;
#[cfg(test)]
#[path = "../../../tests/gui_loop_clock.rs"]
mod tests;

impl NativeGui {
    fn refresh_gui_resources(&mut self) -> Result<(), String> {
        let resources = self.engine.resources();
        if &self.textures == resources {
            return Ok(());
        }
        self.compiler = GuiFrameCompiler::with_resources(resources)?;
        self.textures = resources.clone();
        self.dirty = true;
        Ok(())
    }
}
