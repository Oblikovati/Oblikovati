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
    draws: Option<GuiFrame>,
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
    clipboard: Option<GuiClipboard>,
}
impl NativeGui {
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
        let compiler = GuiFrameCompiler::new(engine.atlas())?;
        Ok(Self::with_compiler(engine, compiler, clipboard))
    }
    fn with_compiler(
        engine: GuiFrameEngine,
        compiler: GuiFrameCompiler,
        clipboard: Option<GuiClipboard>,
    ) -> Self {
        Self {
            engine,
            compiler,
            draws: None,
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
        self.dirty |= self.draws.as_ref() != Some(&output.draws);
        self.draws = Some(output.draws);
        self.commands = Some(commands.clone());
        self.notice = notice.map(str::to_owned);
        self.last_frame = Some(now);
        self.parameter_changed = false;
        self.activity = output.activity;
        self.deadline = self
            .activity
            .needs_frame()
            .then(|| now.saturating_add(GUI_TICK));
        Ok(HeadGuiUpdate {
            consumed_input: self.pointer.consumed(event, output.capture),
            actions: output.actions,
            parameter_actions: output.parameter_actions,
            wake_after: self.wake_after(now),
            ..Default::default()
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
        if self.gui.is_none() {
            let clipboard =
                GuiClipboard::new(self.clipboard.clone(), super::clipboard::CLIPBOARD_BYTES)?;
            self.gui = Some(NativeGui::new(Some(clipboard))?);
            self.view.set_status_text("")?;
        }
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
    if let Err(cause) = view.submit_gui_frame(&gui.compiler, draws) {
        if view.drawable_extent()? == draws.extent {
            return Err(cause);
        }
        gui.deadline = Some(now);
        update.deferred = true;
        update.wake_after = Some(Duration::ZERO);
        return Ok(());
    }
    gui.dirty = false;
    update.changed = true;
    Ok(())
}

#[cfg(test)]
#[path = "../../../tests/native_clipboard_health.rs"]
mod clipboard_health_tests;
#[cfg(test)]
#[path = "../../../tests/native_input_method.rs"]
mod provider_tests;
#[cfg(test)]
#[path = "../../../tests/gui_loop_clock.rs"]
mod tests;
