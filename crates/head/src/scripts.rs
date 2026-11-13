// SPDX-License-Identifier: GPL-2.0-only
use crate::{HeadWork, HeadWorkStatus};
use oblikovati_api::client::Caller;
use oblikovati_script::{IsolatedLuaEngine, ScriptLimits, console::ScriptConsoleController};
use std::time::Duration;
/// Borrow the canonical model caller only on its document owner.
pub trait ScriptDocumentOwner: HeadWork {
    type Host<'a>: Caller
    where
        Self: 'a;
    /// Borrow a caller for one script poll; example: `let host = owner.script_host();`.
    fn script_host(&mut self) -> Self::Host<'_>;
}
/// Desktop script execution and document dispatch share the same frame owner.
pub struct DesktopScripts<D> {
    owner: D,
    console: ScriptConsoleController,
}
impl<D: ScriptDocumentOwner> DesktopScripts<D> {
    /// Inject the owner and trusted child entry; example: `DesktopScripts::new(owner, engine, ScriptLimits::gui())`.
    pub fn new(owner: D, engine: IsolatedLuaEngine, limits: ScriptLimits) -> Self {
        Self::with_methods(
            owner,
            engine,
            limits,
            oblikovati_automation::file_host_methods(),
        )
    }
    /// Inject the configured host inventory; example: `DesktopScripts::with_methods(owner, engine, limits, methods)`.
    pub fn with_methods(
        owner: D,
        engine: IsolatedLuaEngine,
        limits: ScriptLimits,
        methods: Vec<String>,
    ) -> Self {
        Self {
            owner,
            console: ScriptConsoleController::new(engine, limits, methods),
        }
    }
    /// Borrow native console controls; example: `desktop.console_mut().run(source)?`.
    pub fn console_mut(&mut self) -> &mut ScriptConsoleController {
        &mut self.console
    }
    /// Read immutable console state; example: `desktop.console().snapshot()` for a panel.
    pub fn console(&self) -> &ScriptConsoleController {
        &self.console
    }
    /// Read document ownership after a script; example: `desktop.document_owner()` for scene publication.
    pub fn document_owner(&self) -> &D {
        &self.owner
    }
    /// Borrow document ownership for UI commands; example: `desktop.document_owner_mut().script_host()`.
    pub fn document_owner_mut(&mut self) -> &mut D {
        &mut self.owner
    }
    /// Retire the child before returning model ownership; example: `desktop.into_document_owner()` after a failed window run.
    pub fn into_document_owner(mut self) -> D {
        self.console.shutdown();
        self.owner
    }
}
impl<D: ScriptDocumentOwner> HeadWork for DesktopScripts<D> {
    fn accept_sketch_alias(
        &mut self,
        event: oblikovati_viewport::ViewportEvent,
    ) -> (bool, Option<oblikovati_gui::GuiShellAction>) {
        self.owner.accept_sketch_alias(event)
    }
    fn take_shortcut_action(&mut self) -> Option<oblikovati_gui::GuiShellAction> {
        self.owner.take_shortcut_action()
    }
    fn set_keymap_from_gui(
        &mut self,
        delta: &oblikovati_api::wire::KeymapExport,
    ) -> Result<bool, String> {
        self.owner.set_keymap_from_gui(delta)
    }
    fn resolve_shortcut(
        &mut self,
        event: oblikovati_viewport::ViewportEvent,
    ) -> oblikovati_viewport::ViewportEvent {
        self.owner.resolve_shortcut(event)
    }
    fn startup_ready(&self) -> bool {
        self.owner.startup_ready()
    }
    fn complete_save_path(
        &mut self,
        document: u64,
        selected: Option<&std::path::Path>,
    ) -> Result<bool, String> {
        self.owner.complete_save_path(document, selected)
    }
    fn select_color_scheme_from_gui(
        &mut self,
        request: &oblikovati_api::wire::SetColorSchemeArgs,
    ) -> Result<bool, String> {
        self.owner.select_color_scheme_from_gui(request)
    }
    fn edit_theme_color_from_gui(
        &mut self,
        name: &str,
        token: &oblikovati_api::types::ThemeToken,
        value: oblikovati_api::types::Rgba,
    ) -> Result<bool, String> {
        self.owner.edit_theme_color_from_gui(name, token, value)
    }
    fn duplicate_theme_from_gui(&mut self, base: &str, name: &str) -> Result<bool, String> {
        self.owner.duplicate_theme_from_gui(base, name)
    }
    fn delete_theme_from_gui(&mut self, name: &str) -> Result<bool, String> {
        self.owner.delete_theme_from_gui(name)
    }
    fn save_theme_from_gui(&mut self, name: &str) -> Result<bool, String> {
        self.owner.save_theme_from_gui(name)
    }
    fn set_prompt_frequency_from_gui(
        &mut self,
        prompt: oblikovati_model::application_prompts::ApplicationPrompt,
        frequency: oblikovati_model::application_prompts::PromptFrequency,
    ) -> Result<bool, String> {
        self.owner.set_prompt_frequency_from_gui(prompt, frequency)
    }
    fn set_sketch_options_from_gui(
        &mut self,
        options: oblikovati_model::application_options::SketchApplicationOptions,
    ) -> Result<bool, String> {
        self.owner.set_sketch_options_from_gui(options)
    }
    fn edit_dimension_value_from_gui(
        &mut self,
        value: &oblikovati_gui::GuiDimensionValue,
    ) -> Result<bool, String> {
        self.owner.edit_dimension_value_from_gui(value)
    }
    fn start_sketch_on_plane_from_gui(
        &mut self,
        plane: oblikovati_gui::GuiOriginPlaneKind,
    ) -> Result<bool, String> {
        self.owner.start_sketch_on_plane_from_gui(plane)
    }
    fn recent_from_gui(
        &mut self,
        action: &oblikovati_gui::GuiRecentAction,
    ) -> Result<bool, String> {
        self.owner.recent_from_gui(action)
    }
    fn view_slot(&self) -> u64 {
        self.owner.view_slot()
    }
    fn document_window_from_gui(&mut self, id: u64, close: bool) -> Result<bool, String> {
        self.owner.document_window_from_gui(id, close)
    }
    fn new_document_window_from_gui(&mut self) -> Result<bool, String> {
        self.owner.new_document_window_from_gui()
    }
    fn inspect_from_gui(
        &mut self,
        action: oblikovati_gui::GuiInspectAction,
    ) -> Result<bool, String> {
        self.owner.inspect_from_gui(action)
    }
    fn precise_point_from_gui(
        &mut self,
        point: &oblikovati_gui::GuiPrecisePoint,
    ) -> Result<bool, String> {
        self.owner.precise_point_from_gui(point)
    }
    fn sketch_constraint_marks(&self) -> Vec<crate::HeadConstraintMark> {
        self.owner.sketch_constraint_marks()
    }
    fn precise_triad_from_gui(
        &mut self,
        command: oblikovati_gui::GuiTriadCommand,
    ) -> Result<bool, String> {
        self.owner.precise_triad_from_gui(command)
    }
    fn precise_triad_position(&self) -> Option<(oblikovati_model::sketch::SketchPlane, [f64; 2])> {
        self.owner.precise_triad_position()
    }
    fn toggle_center_point_format_from_gui(&mut self) -> Result<bool, String> {
        self.owner.toggle_center_point_format_from_gui()
    }
    fn set_center_point_from_gui(&mut self, point: u64, center: bool) -> Result<bool, String> {
        self.owner.set_center_point_from_gui(point, center)
    }
    fn line_command_from_gui(
        &mut self,
        command: oblikovati_gui::GuiLineCommand,
    ) -> Result<bool, String> {
        self.owner.line_command_from_gui(command)
    }
    fn set_ui_scale_from_gui(
        &mut self,
        options: oblikovati_model::ui_scale::UiScaleOptions,
    ) -> Result<bool, String> {
        self.owner.set_ui_scale_from_gui(options)
    }
    fn select_theme_from_gui(&mut self, name: &str) -> Result<bool, String> {
        self.owner.select_theme_from_gui(name)
    }
    fn gui_command_state(&self) -> Result<oblikovati_gui::GuiCommandState, String> {
        self.owner.gui_command_state()
    }
    fn edit_parameter_from_gui(
        &mut self,
        mutation: oblikovati_gui::GuiParameterMutation,
        request: &oblikovati_api::wire::ParameterSetArgs,
    ) -> Result<bool, String> {
        self.owner.edit_parameter_from_gui(mutation, request)
    }
    fn process_inputs(
        &mut self,
        inputs: &[oblikovati_viewport::ViewportEvent],
    ) -> Result<bool, String> {
        self.owner.process_inputs(inputs)
    }

    crate::work_forwarding::forward_owner_view_work!(owner);

    fn process_window_inputs(
        &mut self,
        inputs: &[oblikovati_viewport::ViewportEvent],
        window: &dyn crate::HeadWindow,
    ) -> Result<bool, String> {
        self.owner.process_window_inputs(inputs, window)
    }
    fn take_command_notice(&mut self) -> Option<crate::DesktopCommandNotice> {
        self.owner.take_command_notice()
    }

    fn poll(&mut self, budget: usize) -> Result<HeadWorkStatus, String> {
        let native_budget = owner_batch_budget(budget, self.console.is_active())?;
        let mut activity = self.owner.poll(native_budget)?;
        let host = self.owner.script_host();
        let console = self.console.poll_activity(&host);
        activity.changed |= console.changed;
        // A running VM may emit no stream event. Timed waits service deadlines without idle GPU redraws.
        if console.active {
            let interval = Duration::from_secs(1) / 60;
            activity.wake_after = Some(
                activity
                    .wake_after
                    .map_or(interval, |owner_wait| owner_wait.min(interval)),
            );
        }
        Ok(activity)
    }

    fn close_allowed(&mut self) -> Result<bool, String> {
        if self.console.is_active() {
            return Ok(false);
        }
        self.owner.close_allowed()
    }
    fn finish(&mut self) -> Result<(), String> {
        self.console.shutdown();
        self.owner.finish()
    }
}

impl<D: ScriptDocumentOwner> ScriptDocumentOwner for DesktopScripts<D> {
    type Host<'a>
        = D::Host<'a>
    where
        Self: 'a;
    fn script_host(&mut self) -> Self::Host<'_> {
        self.owner.script_host()
    }
}
fn owner_batch_budget(budget: usize, active: bool) -> Result<usize, String> {
    let reserved = usize::from(active);
    budget.checked_sub(reserved).filter(|remaining| *remaining > 0)
        .ok_or_else(|| format!("desktop frame budget {budget}, script active={active}: expected capacity for owner work and one active script event"))
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn script_event_reservation_keeps_the_total_frame_budget_bounded() {
        assert_eq!(owner_batch_budget(32, true).unwrap(), 31);
        assert_eq!(owner_batch_budget(32, false).unwrap(), 32);
        assert!(owner_batch_budget(1, true).is_err());
        assert!(owner_batch_budget(0, false).is_err());
    }
}
