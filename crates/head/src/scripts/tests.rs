// SPDX-License-Identifier: GPL-2.0-only
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
    assert!(desktop.set_orientation_lock_from_gui(true).unwrap());
    assert_eq!(
        desktop.orientation_selection_pivot().unwrap(),
        Some([1., 2., 3.])
    );
}

struct WindowAwareOwner {
    window_called: Rc<Cell<bool>>,
    action_called: Rc<Cell<bool>>,
}

impl crate::HeadSketchWork for WindowAwareOwner {}
impl HeadWork for WindowAwareOwner {
    fn set_orientation_lock_from_gui(&mut self, locked: bool) -> Result<bool, String> {
        Ok(locked)
    }
    fn orientation_selection_pivot(&mut self) -> Result<Option<[f64; 3]>, String> {
        Ok(Some([1., 2., 3.]))
    }
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
