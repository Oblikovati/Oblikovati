// SPDX-License-Identifier: GPL-2.0-only
//! Synchronize the native AT-SPI tree with the actual application bar.
use super::NativeHeadWindow;
use crate::accessibility::frame::{FrameControl, FrameSemantics};
use oblikovati_viewport::ViewportEvent;

impl NativeHeadWindow {
    pub(super) fn update_accessibility(
        &mut self,
        event: Option<&ViewportEvent>,
        commands: &oblikovati_gui::GuiCommandState,
        update: &mut crate::HeadGuiUpdate,
    ) -> Result<(), String> {
        if matches!(event, Some(ViewportEvent::Resized(_))) {
            self.accessibility_bounds = self.view.window_screen_bounds()?;
        }
        let Some((controls, popup, focused)) =
            self.gui.as_ref().and_then(|gui| gui.frame_controls())
        else {
            return Ok(());
        };
        while let Some(control) = self.accessibility.next_request() {
            if !popup {
                dispatch_control(control, update);
            }
        }
        let semantics = FrameSemantics {
            title: self.accessibility_title.clone(),
            maximized: commands.window_maximized,
            enabled: !popup,
            bounds: controls.bounds.map(|bounds| bounds.map(f64::from)),
            focused: controls.focused.and_then(frame_control),
        };
        let [x1, y1, x2, y2] = self.accessibility_bounds.map(f64::from);
        self.accessibility
            .update(semantics, accesskit::Rect::new(x1, y1, x2, y2), focused);
        Ok(())
    }
}
fn frame_control(index: usize) -> Option<FrameControl> {
    match index {
        0 => Some(FrameControl::Minimize),
        1 => Some(FrameControl::Maximize),
        2 => Some(FrameControl::Close),
        _ => None,
    }
}
fn dispatch_control(control: FrameControl, update: &mut crate::HeadGuiUpdate) {
    match control {
        FrameControl::Minimize => {
            update.window_action = Some(oblikovati_gui::GuiWindowAction::Minimize)
        }
        FrameControl::Maximize => {
            update.window_action = Some(oblikovati_gui::GuiWindowAction::ToggleMaximize)
        }
        FrameControl::Close => update.actions.push(oblikovati_gui::GuiShellAction::Exit),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn control_actions_use_the_gui_close_and_window_paths() {
        let mut update = crate::HeadGuiUpdate::default();
        dispatch_control(FrameControl::Minimize, &mut update);
        assert_eq!(
            update.window_action,
            Some(oblikovati_gui::GuiWindowAction::Minimize)
        );
        dispatch_control(FrameControl::Maximize, &mut update);
        assert_eq!(
            update.window_action,
            Some(oblikovati_gui::GuiWindowAction::ToggleMaximize)
        );
        dispatch_control(FrameControl::Close, &mut update);
        assert_eq!(update.actions, vec![oblikovati_gui::GuiShellAction::Exit]);
        assert_eq!(frame_control(3), None);
    }
}
