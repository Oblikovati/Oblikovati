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
        let host = self.owner.script_host();
        let outcome = self
            .runtime
            .poll(budget, &host)
            .map_err(|cause| cause.to_string())?;
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
    fn startup_ready(&self) -> bool {
        self.owner.startup_ready() && self.startup.as_ref().is_none_or(NativeStartupPlan::ready)
    }
    fn complete_save_path(
        &mut self,
        document: u64,
        selected: Option<&Path>,
    ) -> Result<bool, String> {
        self.owner.complete_save_path(document, selected)
    }
    fn gui_command_state(&self) -> Result<oblikovati_gui::GuiCommandState, String> {
        self.owner.gui_command_state()
    }
    fn process_inputs(
        &mut self,
        inputs: &[oblikovati_viewport::ViewportEvent],
    ) -> Result<bool, String> {
        self.owner.process_inputs(inputs)
    }
    fn take_command_notice(&mut self) -> Option<crate::DesktopCommandNotice> {
        self.startup
            .as_mut()
            .and_then(|plan| plan.notices.pop_front())
            .or_else(|| self.owner.take_command_notice())
    }
    fn take_scene(&mut self) -> Result<Option<crate::HeadSceneUpdate>, String> {
        self.owner.take_scene()
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
