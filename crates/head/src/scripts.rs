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
    fn poll_console(&mut self, activity: &mut HeadWorkStatus) {
        let host = self.owner.script_host();
        let console = self.console.poll_activity(&host);
        activity.changed |= console.changed;
        if console.active {
            let interval = Duration::from_secs(1) / 60;
            activity.wake_after = Some(
                activity
                    .wake_after
                    .map_or(interval, |owner_wait| owner_wait.min(interval)),
            );
        }
    }
}
impl<D: ScriptDocumentOwner> HeadWork for DesktopScripts<D> {
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
    fn gui_command_state(&mut self) -> Result<oblikovati_gui::GuiCommandState, String> {
        self.owner.gui_command_state()
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

    crate::work::forward_sketch_work!(owner);

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
        self.poll_console(&mut activity);
        Ok(activity)
    }

    fn poll_window(
        &mut self,
        budget: usize,
        window: &mut dyn crate::HeadWindow,
    ) -> Result<HeadWorkStatus, String> {
        let native_budget = owner_batch_budget(budget, self.console.is_active())?;
        let mut activity = self.owner.poll_window(native_budget, window)?;
        self.poll_console(&mut activity);
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
    use oblikovati_api::{Error, client::Caller};
    use std::{cell::Cell, rc::Rc};

    #[test]
    fn script_event_reservation_keeps_the_total_frame_budget_bounded() {
        assert_eq!(owner_batch_budget(32, true).unwrap(), 31);
        assert_eq!(owner_batch_budget(32, false).unwrap(), 32);
        assert!(owner_batch_budget(1, true).is_err());
        assert!(owner_batch_budget(0, false).is_err());
    }

    #[test]
    fn script_desktop_delegates_window_aware_polling() {
        let window_called = Rc::new(Cell::new(false));
        let action_called = Rc::new(Cell::new(false));
        let mut desktop = DesktopScripts::new(
            WindowAwareOwner {
                window_called: Rc::clone(&window_called),
                action_called: Rc::clone(&action_called),
            },
            IsolatedLuaEngine::new(std::path::Path::new("/usr/bin/true"), Vec::new()),
            ScriptLimits::gui(),
        );
        desktop.poll_window(2, &mut EmptyWindow).unwrap();
        assert!(window_called.get());
        assert!(
            desktop
                .apply_gui_shell_action(oblikovati_gui::GuiShellAction::NewPart)
                .unwrap()
        );
        assert!(action_called.get());
    }

    struct WindowAwareOwner {
        window_called: Rc<Cell<bool>>,
        action_called: Rc<Cell<bool>>,
    }

    impl HeadWork for WindowAwareOwner {
        fn poll(&mut self, _: usize) -> Result<HeadWorkStatus, String> {
            Ok(Default::default())
        }
        fn poll_window(
            &mut self,
            _: usize,
            _: &mut dyn crate::HeadWindow,
        ) -> Result<HeadWorkStatus, String> {
            self.window_called.set(true);
            Ok(Default::default())
        }
        fn apply_gui_shell_action(
            &mut self,
            _: oblikovati_gui::GuiShellAction,
        ) -> Result<bool, String> {
            self.action_called.set(true);
            Ok(true)
        }
        fn close_allowed(&mut self) -> Result<bool, String> {
            Ok(true)
        }
        fn finish(&mut self) -> Result<(), String> {
            Ok(())
        }
    }

    impl ScriptDocumentOwner for WindowAwareOwner {
        type Host<'a> = NoopCaller;
        fn script_host(&mut self) -> Self::Host<'_> {
            NoopCaller
        }
    }

    struct NoopCaller;

    impl Caller for NoopCaller {
        fn call(&self, method: &str, _: &[u8]) -> Result<Vec<u8>, Error> {
            Err(Error(format!(
                "test method {method:?}: expected no console request"
            )))
        }
    }

    struct EmptyWindow;

    impl crate::HeadWindow for EmptyWindow {
        fn events(&mut self, _: usize) -> Result<Vec<oblikovati_viewport::ViewportEvent>, String> {
            Ok(Vec::new())
        }
        fn present(&mut self) -> Result<crate::HeadPresentation, String> {
            Ok(crate::HeadPresentation::Completed)
        }
        fn wait(&mut self, _: Option<Duration>) -> Result<bool, String> {
            Ok(false)
        }
        fn finish(&mut self) -> Result<(), String> {
            Ok(())
        }
    }
}
