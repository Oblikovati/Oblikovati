// SPDX-License-Identifier: GPL-2.0-only
use super::{HeadWindow, HeadWork, ViewportEvent, ViewportInput};

pub(super) fn update_tool_candidate(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    event: &ViewportEvent,
) -> Result<bool, String> {
    let ViewportEvent::Input(ViewportInput::Motion { position, .. }) = event else {
        return window.set_tool_feedback(None);
    };
    if window.navigation_active() {
        return window.set_tool_feedback(None);
    }
    let candidate = match owner.sketch_tool_feedback(*position, window) {
        Ok(candidate) => candidate,
        Err(cause) => {
            window.show_command_notice(crate::DesktopCommandNotice::SelectionFailed { cause })?;
            window.set_tool_feedback(None)?;
            return Ok(true);
        }
    };
    // #4051: a pick uses the owner generation; its display must use that same sketch.
    let updated = if candidate.is_some() {
        super::super::publish_owner_sketches(window, owner)?
    } else {
        false
    };
    Ok(window.set_tool_feedback(candidate.as_ref())? || updated)
}

pub(super) fn semantic_command_event(event: ViewportEvent) -> ViewportEvent {
    let ViewportEvent::Input(ViewportInput::PhysicalKey {
        command: Some(key),
        pressed,
        ..
    }) = event
    else {
        return event;
    };
    ViewportEvent::Input(ViewportInput::Key { key, pressed })
}
