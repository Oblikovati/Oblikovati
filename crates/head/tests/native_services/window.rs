// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) struct CloseWhenReadyWindow<'a> {
    pub window: &'a mut NativeHeadWindow,
    pub done: Rc<Cell<bool>>,
    pub deadline: std::time::Instant,
}
impl HeadWindow for CloseWhenReadyWindow<'_> {
    fn events(&mut self, limit: usize) -> Result<Vec<ViewportEvent>, String> {
        if std::time::Instant::now() >= self.deadline {
            return Err("native fixture startup: expected completion within twenty seconds".into());
        }
        let mut events = self.window.events(limit)?;
        if self.done.get() {
            events.push(ViewportEvent::CloseRequested);
        }
        Ok(events)
    }
    fn submit_scene(&mut self, update: crate::HeadSceneUpdate) -> Result<(), String> {
        self.window.submit_scene(update)
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        self.window.present()
    }
    fn wait(&mut self, _: Option<std::time::Duration>) -> Result<bool, String> {
        self.window.wait(Some(std::time::Duration::from_millis(20)))
    }
    fn finish(&mut self) -> Result<(), String> {
        self.window.finish()
    }
}
