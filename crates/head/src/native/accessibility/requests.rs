// SPDX-License-Identifier: GPL-2.0-only
//! Apply assistive requests on the GUI owner thread.
use super::{FrameControl, FrameRequest, dispatch_control, focus_target};
use crate::{DesktopCommandNotice, HeadGuiUpdate};
use oblikovati_gui::GuiClosePrompt;

pub(super) struct RequestContext<'a> {
    pub view: &'a mut oblikovati_viewport::Viewport,
    pub gui: &'a mut Option<super::super::gui_loop::NativeGui>,
    pub notice: &'a mut Option<DesktopCommandNotice>,
    pub update: &'a mut HeadGuiUpdate,
    pub popup: bool,
    pub focused: bool,
    pub prompt: Option<&'a GuiClosePrompt>,
    pub answered: bool,
}
impl RequestContext<'_> {
    pub fn apply(&mut self, request: FrameRequest) {
        let control = match request {
            FrameRequest::Click(control) | FrameRequest::Focus(control) => control,
        };
        if !allowed(control, self.popup, self.prompt) {
            return;
        }
        match request {
            FrameRequest::Focus(_) => self.focus(control),
            FrameRequest::Click(FrameControl::Answer(choice)) if !self.answered => {
                if let Some(gui) = self.gui.as_mut() {
                    gui.answer_close_prompt(choice);
                }
                self.answered = true;
                self.wake();
            }
            FrameRequest::Click(FrameControl::Answer(_)) => {}
            FrameRequest::Click(_) => dispatch_control(control, self.update),
        }
    }
    fn focus(&mut self, control: FrameControl) {
        if !self.focused
            && let Err(cause) = self.view.request_accessible_focus()
        {
            *self.notice = Some(DesktopCommandNotice::AuthoringFailed { cause });
            self.update.changed = true;
            return;
        }
        if let Some(gui) = self.gui.as_mut() {
            gui.focus_accessible(focus_target(control));
        }
        self.wake();
    }
    fn wake(&mut self) {
        self.update.changed = true;
        self.update.wake_after = Some(std::time::Duration::ZERO);
    }
}

fn allowed(control: FrameControl, popup: bool, prompt: Option<&GuiClosePrompt>) -> bool {
    match control {
        FrameControl::Answer(choice) => {
            prompt.is_some_and(|dialog| dialog.buttons.iter().any(|button| button.choice == choice))
        }
        _ => !popup,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn only_current_modal_choices_are_allowed_during_a_prompt() {
        let prompt = GuiClosePrompt {
            title: "Save changes",
            message: "Save?".into(),
            bounds: [0.; 4],
            buttons: vec![oblikovati_gui::GuiClosePromptButton {
                label: "Cancel",
                choice: oblikovati_gui::GuiClosePromptChoice::Cancel,
                bounds: [0.; 4],
                focused: false,
            }],
        };
        assert!(!allowed(FrameControl::Close, true, Some(&prompt)));
        assert!(allowed(FrameControl::Close, false, None));
        assert!(allowed(
            FrameControl::Answer(oblikovati_gui::GuiClosePromptChoice::Cancel),
            true,
            Some(&prompt)
        ));
        assert!(!allowed(
            FrameControl::Answer(oblikovati_gui::GuiClosePromptChoice::Save),
            true,
            Some(&prompt)
        ));
        assert!(!allowed(
            FrameControl::Answer(oblikovati_gui::GuiClosePromptChoice::Cancel),
            false,
            None
        ));
    }
}
