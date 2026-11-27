// SPDX-License-Identifier: GPL-2.0-only
use crate::{DesktopDispatchOptions, HeadWork, HeadWorkStatus, ScriptDocumentOwner};
use oblikovati_addin_host::{NativeAddInRuntime, NativeCompletion};
use oblikovati_automation::dispatch::DispatchWakeup;
use std::path::{Path, PathBuf};
mod startup;
use startup::NativeStartupPlan;
/// Native services borrow the desktop owner; the foreign thread contains no model or window.
pub struct NativeAddInDesktop<'a, D: ScriptDocumentOwner, W: DispatchWakeup> {
    owner: &'a mut D,
    runtime: NativeAddInRuntime<W>,
    completion: Option<Result<NativeCompletion, String>>,
    stopped: bool,
    startup: Option<NativeStartupPlan>,
}
impl<'a, D: ScriptDocumentOwner, W: DispatchWakeup + 'static> NativeAddInDesktop<'a, D, W> {
    /// Install one process binding with injected wake; example: `NativeAddInDesktop::new(&mut owner, options, wake)?`.
    pub fn new(owner: &'a mut D, options: DesktopDispatchOptions, wake: W) -> Result<Self, String> {
        let runtime = NativeAddInRuntime::new(options.capacity, options.timeout, wake)
            .map_err(|cause| cause.to_string())?;
        Ok(Self {
            owner,
            runtime,
            completion: None,
            stopped: false,
            startup: None,
        })
    }
    /// Queue trusted startup libraries; example: `unsafe { desktop.begin_startup(paths) }?`.
    /// # Safety
    /// Every library must satisfy the same trust, affinity, and retirement obligations as `start_load`.
    pub unsafe fn begin_startup(&mut self, paths: Vec<PathBuf>) -> Result<(), String> {
        self.require_consumed_completion()?;
        if self.startup.is_some() {
            return Err("native startup already selected: expected one startup plan".into());
        }
        let mut plan = NativeStartupPlan::new(paths);
        plan.next(&mut self.runtime)?;
        self.startup = Some(plan);
        Ok(())
    }
    /// Consume one worker result before starting another control; example: `desktop.take_native_completion()` after poll.
    pub fn take_native_completion(&mut self) -> Option<Result<NativeCompletion, String>> {
        self.completion.take()
    }
    /// Start a library on the native worker; example: `unsafe { desktop.start_load(path) }?`.
    /// # Safety
    /// Trust constructors, canonical ABI exports, reply allocators, and the assigned worker affinity.
    /// Retain and drain this desktop until successful shutdown; do not create a lifecycle alias.
    pub unsafe fn start_load(&mut self, path: &Path) -> Result<(), String> {
        self.require_consumed_completion()?;
        unsafe { self.runtime.start_load(path) }.map_err(|cause| cause.to_string())
    }
    /// Start activation while native frames continue; example: `unsafe { desktop.start_activate(id) }?`.
    /// # Safety
    /// The admitted library must obey the ABI and join all callback work before deactivation succeeds.
    pub unsafe fn start_activate(&mut self, id: &str) -> Result<(), String> {
        self.require_consumed_completion()?;
        unsafe { self.runtime.start_activate(id) }.map_err(|cause| cause.to_string())
    }
    fn require_consumed_completion(&self) -> Result<(), String> {
        if self.stopped || self.completion.is_some() || !self.startup_ready() {
            return Err(format!(
                "native desktop stopped={}, completion={:?}: expected a live desktop and consumed control result",
                self.stopped, self.completion
            ));
        }
        Ok(())
    }
    fn poll_native(&mut self, budget: usize) -> Result<bool, String> {
        let outcome = {
            let host = self.owner.script_host();
            self.runtime.poll(budget, &host)
        }
        .map_err(|cause| cause.to_string())?;
        self.record_native_outcome(outcome)
    }
    fn poll_native_window(
        &mut self,
        budget: usize,
        window: &mut dyn crate::HeadWindow,
    ) -> Result<bool, String> {
        let (outcome, window_changed) = {
            let slots = self.owner.document_camera_slots();
            let host = crate::window_caller::WindowCaller::new(self.owner, window, slots);
            let outcome = self.runtime.poll(budget, &host);
            (outcome, host.window_changed())
        };
        let outcome = outcome.map_err(|cause| cause.to_string())?;
        Ok(self.record_native_outcome(outcome)? || window_changed)
    }
    fn record_native_outcome(
        &mut self,
        outcome: Option<Result<NativeCompletion, oblikovati_api::Error>>,
    ) -> Result<bool, String> {
        let Some(outcome) = outcome else {
            return Ok(false);
        };
        let outcome = outcome.map_err(|cause| cause.to_string());
        if let Some(plan) = &mut self.startup
            && !plan.ready()
        {
            plan.completed(&mut self.runtime, outcome)?;
        } else {
            self.completion = Some(outcome);
        }
        Ok(true)
    }
    fn pending_failure(&mut self) -> Option<String> {
        if !self.runtime.control_pending() {
            return self
                .completion
                .as_ref()
                .and_then(|outcome| outcome.as_ref().err())
                .cloned();
        }
        let host = self.owner.script_host();
        self.runtime
            .wait(&host)
            .err()
            .map(|cause| cause.to_string())
    }
    fn finish_native(&mut self) -> Result<(), String> {
        let prior = self.pending_failure().map_or(Ok(()), Err);
        crate::frames::combine_retirement(prior, self.shutdown_foreign())
    }
    fn shutdown_foreign(&mut self) -> Result<(), String> {
        let host = self.owner.script_host();
        unsafe { self.runtime.start_shutdown() }.map_err(|cause| cause.to_string())?;
        self.runtime
            .wait(&host)
            .map_err(|cause| cause.to_string())?;
        drop(host);
        self.stopped = true;
        self.require_retired_replies()
    }
    fn require_retired_replies(&self) -> Result<(), String> {
        let outstanding = self.runtime.last_buffer_counts().outstanding;
        if outstanding != 0 {
            return Err(format!(
                "native host buffers {outstanding}: expected zero outstanding replies after shutdown"
            ));
        }
        Ok(())
    }
}
impl<D: ScriptDocumentOwner, W: DispatchWakeup + 'static> HeadWork
    for NativeAddInDesktop<'_, D, W>
{
    fn set_marking_menu_from_gui(
        &mut self,
        menus: &[oblikovati_api::wire::MarkingMenuView],
        display: Option<oblikovati_model::marking_menu::MarkingMenuDisplay>,
    ) -> Result<bool, String> {
        self.owner.set_marking_menu_from_gui(menus, display)
    }
    fn take_shortcut_action(&mut self) -> Option<oblikovati_gui::GuiShellAction> {
        self.owner.take_shortcut_action()
    }
    fn set_keymap_from_gui(
        &mut self,
        preferences: &oblikovati_model::keymap::KeymapPreferences,
    ) -> Result<bool, String> {
        self.owner.set_keymap_from_gui(preferences)
    }
    fn resolve_shortcut(
        &mut self,
        event: oblikovati_viewport::ViewportEvent,
    ) -> oblikovati_viewport::ViewportEvent {
        self.owner.resolve_shortcut(event)
    }
    fn startup_ready(&self) -> bool {
        self.owner.startup_ready() && self.startup.as_ref().is_none_or(NativeStartupPlan::ready)
    }
    fn saved_document_path(&self, document: u64) -> Result<&Path, String> {
        self.owner.saved_document_path(document)
    }
    fn complete_save_path(
        &mut self,
        document: u64,
        selected: Option<&Path>,
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
    fn recent_from_gui(
        &mut self,
        action: &oblikovati_gui::GuiRecentAction,
    ) -> Result<bool, String> {
        self.owner.recent_from_gui(action)
    }
    fn view_slot(&self) -> u64 {
        self.owner.view_slot()
    }
    fn document_camera_slots(&self) -> std::collections::BTreeMap<u64, u64> {
        self.owner.document_camera_slots()
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
    fn measure_selected_face_from_gui(&mut self, document: u64, key: &str) -> Result<bool, String> {
        self.owner.measure_selected_face_from_gui(document, key)
    }
    fn open_face_properties_from_gui(&mut self, document: u64, key: &str) -> Result<bool, String> {
        self.owner.open_face_properties_from_gui(document, key)
    }
    fn close_face_properties_from_gui(&mut self) -> Result<bool, String> {
        self.owner.close_face_properties_from_gui()
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
    fn gui_command_state(&mut self) -> Result<oblikovati_gui::GuiCommandState, String> {
        self.owner.gui_command_state()
    }
    fn edit_parameter_from_gui(
        &mut self,
        mutation: oblikovati_gui::GuiParameterMutation,
        request: &oblikovati_api::wire::ParameterSetArgs,
    ) -> Result<bool, String> {
        self.owner.edit_parameter_from_gui(mutation, request)
    }
    fn clear_selected_face_appearances_from_gui(
        &mut self,
        document: u64,
        keys: &[String],
    ) -> Result<bool, String> {
        self.owner
            .clear_selected_face_appearances_from_gui(document, keys)
    }
    fn clear_selected_mixed_appearance_from_gui(
        &mut self,
        document: u64,
        bodies: &[String],
        faces: &[String],
    ) -> Result<bool, String> {
        self.owner
            .clear_selected_mixed_appearance_from_gui(document, bodies, faces)
    }
    fn clear_selected_body_appearances_from_gui(
        &mut self,
        document: u64,
        keys: &[String],
    ) -> Result<bool, String> {
        self.owner
            .clear_selected_body_appearances_from_gui(document, keys)
    }
    fn set_model_state_table_from_gui(
        &mut self,
        document: u64,
        request_id: u64,
        request: &oblikovati_api::wire::SetModelStateTableArgs,
    ) -> Result<bool, String> {
        self.owner
            .set_model_state_table_from_gui(document, request_id, request)
    }
    fn set_document_property_from_gui(
        &mut self,
        request: &oblikovati_api::wire::SetPropertyArgs,
    ) -> Result<bool, String> {
        self.owner.set_document_property_from_gui(request)
    }
    fn gui_parameter_snapshot(
        &self,
    ) -> Result<Option<oblikovati_gui::GuiParameterSnapshot>, String> {
        self.owner.gui_parameter_snapshot()
    }
    fn apply_gui_parameter(
        &mut self,
        action: &oblikovati_gui::GuiParameterAction,
    ) -> Result<bool, String> {
        self.owner.apply_gui_parameter(action)
    }
    fn apply_gui_shell_action(
        &mut self,
        action: oblikovati_gui::GuiShellAction,
    ) -> Result<bool, String> {
        self.owner.apply_gui_shell_action(action)
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
    fn process_live_window_inputs(
        &mut self,
        inputs: &[oblikovati_viewport::ViewportEvent],
        window: &mut dyn crate::HeadWindow,
    ) -> Result<bool, String> {
        self.owner.process_live_window_inputs(inputs, window)
    }
    fn take_command_notice(&mut self) -> Option<crate::DesktopCommandNotice> {
        self.startup
            .as_mut()
            .and_then(|plan| plan.notices.pop_front())
            .or_else(|| self.owner.take_command_notice())
    }

    fn poll(&mut self, budget: usize) -> Result<HeadWorkStatus, String> {
        if self.stopped {
            return Err("native desktop stopped=true: expected live owner polling".into());
        }
        if budget < 2 {
            return Err(format!(
                "native desktop budget {budget}: expected at least two owner request slots"
            ));
        }
        let changed = self.poll_native(budget / 2)?;
        let mut activity = self.owner.poll(budget - budget / 2)?;
        activity.changed |= changed;
        activity.pending |= self
            .runtime
            .queued_calls_pending()
            .map_err(|cause| cause.to_string())?;
        Ok(activity)
    }
    fn poll_window(
        &mut self,
        budget: usize,
        window: &mut dyn crate::HeadWindow,
    ) -> Result<HeadWorkStatus, String> {
        if self.stopped {
            return Err("native desktop stopped=true: expected live owner polling".into());
        }
        let changed = self.poll_native_window(budget / 2, window)?;
        let mut activity = self.owner.poll_window(budget - budget / 2, window)?;
        activity.changed |= changed;
        activity.pending |= self
            .runtime
            .queued_calls_pending()
            .map_err(|cause| cause.to_string())?;
        Ok(activity)
    }
    fn close_allowed(&mut self) -> Result<bool, String> {
        if !self.startup_ready() || self.runtime.control_pending() {
            return Ok(false);
        }
        self.owner.close_allowed()
    }
    fn finish(&mut self) -> Result<(), String> {
        let native = if self.stopped {
            Ok(())
        } else {
            self.finish_native()
        };
        if !self.stopped {
            return native;
        }
        crate::frames::combine_retirement(native, self.owner.finish())
    }
}
impl<D: ScriptDocumentOwner, W: DispatchWakeup + 'static> ScriptDocumentOwner
    for NativeAddInDesktop<'_, D, W>
{
    type Host<'a>
        = oblikovati_addin_host::NativeCatalogCaller<D::Host<'a>>
    where
        Self: 'a;
    fn script_host(&mut self) -> Self::Host<'_> {
        self.runtime.catalog_caller(self.owner.script_host())
    }
}

impl<D: ScriptDocumentOwner, W: DispatchWakeup + 'static> crate::HeadSketchWork
    for NativeAddInDesktop<'_, D, W>
{
    fn set_part_options_from_gui(
        &mut self,
        options: oblikovati_model::application_options::PartApplicationOptions,
    ) -> Result<bool, String> {
        self.owner.set_part_options_from_gui(options)
    }
    fn accept_sketch_alias(
        &mut self,
        event: oblikovati_viewport::ViewportEvent,
    ) -> (bool, Option<oblikovati_gui::GuiShellAction>) {
        self.owner.accept_sketch_alias(event)
    }
    fn set_document_sketch_settings_from_gui(
        &mut self,
        document: u64,
        expected: &oblikovati_api::types::SketchSettings,
        settings: &oblikovati_api::types::SketchSettings,
    ) -> Result<bool, String> {
        self.owner
            .set_document_sketch_settings_from_gui(document, expected, settings)
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
    fn start_sketch_on_work_plane_from_gui(
        &mut self,
        plane: oblikovati_model::feature::FeatureId,
    ) -> Result<bool, String> {
        self.owner.start_sketch_on_work_plane_from_gui(plane)
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
    crate::work_sketch_forwarding::forward_owner_sketch_work!(owner);
}
