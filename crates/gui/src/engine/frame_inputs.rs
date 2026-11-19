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
