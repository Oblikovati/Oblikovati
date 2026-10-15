// SPDX-License-Identifier: GPL-2.0-only
use super::{HeadWindow, HeadWork, ViewportEvent, synchronize_sketch_view};
use oblikovati_gui::GuiShellAction;
use oblikovati_viewport::{ViewportInput, ViewportKey};
use std::time::Duration;
/// GUI work joins the existing event and wake paths; lifecycle events remain owner-visible.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct HeadGuiUpdate {
    pub changed: bool,
    pub consumed_input: bool,
    pub deferred: bool,
    pub actions: Vec<GuiShellAction>,
    pub parameter_actions: Vec<oblikovati_gui::GuiParameterAction>,
    pub wake_after: Option<Duration>,
}
pub(super) fn advance_gui(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    event: Option<&ViewportEvent>,
) -> Result<HeadGuiUpdate, String> {
    let metadata_changed = window.gui_parameters_open() && publish_parameter_view(window, owner)?;
    let mut update = window.gui_frame(event, &owner.gui_command_state()?)?;
    update.changed |= metadata_changed;
    for action in &update.parameter_actions {
        let result = owner.apply_gui_parameter(action);
        update.changed |= result.as_ref().copied().unwrap_or(false);
        window.acknowledge_gui_parameter(action, result.map(|_| ()))?;
    }
    for action in &update.actions {
        update.changed |= owner.apply_gui_shell_action(*action)?;
        if let Some(event) = action_event(*action) {
            update.changed |= dispatch_frame_input(window, owner, &event)?;
        }
    }
    Ok(update)
}
fn publish_parameter_view(
    window: &mut dyn HeadWindow,
    owner: &dyn HeadWork,
) -> Result<bool, String> {
    match owner.gui_parameter_snapshot() {
        Ok(snapshot) => window
            .set_gui_parameter_snapshot(snapshot)
            .or_else(|issue| window.show_gui_parameter_issue(issue)),
        Err(issue) => window.show_gui_parameter_issue(issue),
    }
}
fn action_event(action: GuiShellAction) -> Option<ViewportEvent> {
    let key = match action {
        GuiShellAction::Save => ViewportKey::Save,
        GuiShellAction::Undo => ViewportKey::Undo,
        GuiShellAction::Redo => ViewportKey::Redo,
        GuiShellAction::Home => ViewportKey::Home,
        GuiShellAction::NewPart
        | GuiShellAction::SketchXY
        | GuiShellAction::Rectangle
        | GuiShellAction::Extrude => return None,
    };
    Some(ViewportEvent::Input(ViewportInput::Key {
        key,
        pressed: true,
    }))
}
pub(super) fn dispatch_frame_input(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    event: &ViewportEvent,
) -> Result<bool, String> {
    let event = semantic_command_event(*event);
    let event = &event;
    let mut changed = window.process_view_input(event)?;
    changed |= owner.process_window_inputs(&[*event], window)?;
    changed |= synchronize_sketch_view(window, owner)?;
    Ok(changed)
}
fn semantic_command_event(event: ViewportEvent) -> ViewportEvent {
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
pub(super) fn earliest_wake(left: Option<Duration>, right: Option<Duration>) -> Option<Duration> {
    left.into_iter().chain(right).min()
}
#[cfg(test)]
#[path = "../../tests/gui_frames.rs"]
mod tests;
