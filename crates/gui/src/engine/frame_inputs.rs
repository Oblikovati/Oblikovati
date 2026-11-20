// SPDX-License-Identifier: GPL-2.0-only
//! Inspect frame navigation input without changing keyboard capture.
use super::*;
pub(super) fn has_arrow_press(request: &GuiFrameRequest) -> bool {
    request.events.iter().any(|event| {
        matches!(
            event,
            GuiInputEvent::Key {
                key: crate::GuiKey::LeftArrow
                    | crate::GuiKey::RightArrow
                    | crate::GuiKey::UpArrow
                    | crate::GuiKey::DownArrow,
                pressed: true
            }
        )
    })
}

/// Select a frame-control focus request; example: `frame_focus(Some(GuiAccessibleFocus::Frame(1)))`.
pub(super) fn frame_focus(focus: Option<crate::GuiAccessibleFocus>) -> Option<usize> {
    match focus {
        Some(crate::GuiAccessibleFocus::Frame(index)) => Some(index),
        _ => None,
    }
}

#[cfg(test)]
mod focus_tests {
    #[test]
    fn selects_only_frame_controls() {
        assert_eq!(
            super::frame_focus(Some(crate::GuiAccessibleFocus::Frame(2))),
            Some(2)
        );
        assert_eq!(
            super::frame_focus(Some(crate::GuiAccessibleFocus::ClosePrompt(
                crate::GuiClosePromptChoice::Cancel
            ))),
            None
        );
        assert_eq!(super::frame_focus(None), None);
    }
}
