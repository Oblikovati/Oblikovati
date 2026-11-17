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
        #[cfg(target_os = "linux")]
        if matches!(event, Some(ViewportEvent::Resized(_))) {
            self.accessibility_bounds = self.view.window_screen_bounds()?;
        }
        let Some((controls, popup, focused, prompt)) =
            self.gui.as_ref().and_then(|gui| gui.frame_controls())
        else {
            return Ok(());
        };
        #[cfg(target_os = "windows")]
        let Some(accessibility) = self.accessibility.as_mut() else {
            return Ok(());
        };
        #[cfg(target_os = "linux")]
        let accessibility = &mut self.accessibility;
        let mut answered = false;
        while let Some(control) = accessibility.next_request() {
            match control {
                FrameControl::Answer(choice)
                    if !answered
                        && prompt.as_ref().is_some_and(|dialog| {
                            dialog.buttons.iter().any(|button| button.choice == choice)
                        }) =>
                {
                    if let Some(gui) = self.gui.as_mut() {
                        gui.answer_close_prompt(choice);
                    }
                    answered = true;
                    update.changed = true;
                    update.wake_after = Some(std::time::Duration::ZERO);
                }
                FrameControl::Answer(_) => {}
                _ if !popup => dispatch_control(control, update),
                _ => {}
            }
        }
        let semantics = FrameSemantics {
            title: self.accessibility_title.clone(),
            maximized: commands.window_maximized,
            enabled: !popup,
            bounds: controls.bounds.map(|bounds| bounds.map(f64::from)),
            focused: controls.focused.and_then(frame_control),
            prompt,
        };
        #[cfg(target_os = "linux")]
        {
            let [x1, y1, x2, y2] = self.accessibility_bounds.map(f64::from);
            accessibility.update(semantics, accesskit::Rect::new(x1, y1, x2, y2), focused);
        }
        #[cfg(target_os = "windows")]
        {
            let _ = (event, focused);
            accessibility.update(semantics);
        }
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
        FrameControl::Answer(_) => {}
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
