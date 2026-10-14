// SPDX-License-Identifier: GPL-2.0-only
use super::{HeadWindow, HeadWork, ViewportEvent, synchronize_sketch_view};
use oblikovati_gui::GuiShellAction;
use oblikovati_viewport::{ViewportInput, ViewportKey};
use std::time::Duration;
/// GUI work joins the existing event and wake paths; lifecycle events remain owner-visible.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct HeadGuiUpdate {
    pub changed: bool,
    pub consumed_input: bool,
    pub deferred: bool,
    pub actions: Vec<GuiShellAction>,
    pub wake_after: Option<Duration>,
}
pub(super) fn advance_gui(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    event: Option<&ViewportEvent>,
) -> Result<HeadGuiUpdate, String> {
    let mut update = window.gui_frame(event, &owner.gui_command_state()?)?;
    for action in &update.actions {
        let event = ViewportEvent::Input(ViewportInput::Key {
            key: action_key(*action),
            pressed: true,
        });
        update.changed |= dispatch_frame_input(window, owner, &event)?;
    }
    Ok(update)
}
fn action_key(action: GuiShellAction) -> ViewportKey {
    match action {
        GuiShellAction::Save => ViewportKey::Save,
        GuiShellAction::Undo => ViewportKey::Undo,
        GuiShellAction::Redo => ViewportKey::Redo,
        GuiShellAction::Home => ViewportKey::Home,
    }
}
pub(super) fn dispatch_frame_input(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    event: &ViewportEvent,
) -> Result<bool, String> {
    let mut changed = window.process_view_input(event)?;
    changed |= owner.process_window_inputs(&[*event], window)?;
    changed |= synchronize_sketch_view(window, owner)?;
    Ok(changed)
}
pub(super) fn earliest_wake(left: Option<Duration>, right: Option<Duration>) -> Option<Duration> {
    left.into_iter().chain(right).min()
}
#[cfg(test)]
#[path = "../../tests/gui_frames.rs"]
mod tests;
