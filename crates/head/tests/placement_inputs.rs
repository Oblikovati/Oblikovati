// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_viewport::{ViewportButton, ViewportEvent, ViewportInput, ViewportKey};
pub(crate) fn button(pixel: [i32; 2], pressed: bool) -> ViewportEvent {
    ViewportEvent::Input(ViewportInput::Button {
        button: ViewportButton::Left,
        pressed,
        position: pixel,
        modifiers: Default::default(),
    })
}
pub(crate) fn motion(pixel: [i32; 2]) -> ViewportEvent {
    ViewportEvent::Input(ViewportInput::Motion {
        position: pixel,
        modifiers: Default::default(),
    })
}
pub(crate) fn key(key: ViewportKey, pressed: bool) -> ViewportEvent {
    ViewportEvent::Input(ViewportInput::Key { key, pressed })
}

pub(crate) struct ProjectionlessWindow;
impl crate::HeadWindow for ProjectionlessWindow {
    fn events(&mut self, _: usize) -> Result<Vec<ViewportEvent>, String> {
        Ok(Vec::new())
    }
    fn present(&mut self) -> Result<crate::HeadPresentation, String> {
        Ok(crate::HeadPresentation::Completed)
    }
    fn wait(&mut self, _: Option<std::time::Duration>) -> Result<bool, String> {
        Ok(false)
    }
    fn finish(&mut self) -> Result<(), String> {
        Ok(())
    }
}
