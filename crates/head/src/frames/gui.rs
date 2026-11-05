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
    pub wake_after: Option<Duration>,
}
pub(super) fn advance_gui(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    event: Option<&ViewportEvent>,
) -> Result<HeadGuiUpdate, String> {
    let mut update = window.gui_frame(event, &owner.gui_command_state()?)?;
    for action in &update.actions {
        if let GuiShellAction::EditSketch { document, sketch } = action {
            update.changed |= report_authoring_result(
                window,
                owner.edit_existing_sketch_from_gui(*document, *sketch),
            )?;
            update.changed |= synchronize_sketch_view(window, owner)?;
            continue;
        }
        if let GuiShellAction::Parameter { mutation, request } = action {
            update.changed |=
                report_authoring_result(window, owner.edit_parameter_from_gui(*mutation, request))?;
            continue;
        }
        if *action == GuiShellAction::Rectangle {
            update.changed |= report_authoring_result(window, owner.place_rectangle_from_gui())?;
            continue;
        }
        if matches!(
            action,
            GuiShellAction::StartSketchXY | GuiShellAction::FinishSketch
        ) {
            update.changed |= report_authoring_result(
                window,
                owner.edit_sketch_from_gui(*action == GuiShellAction::StartSketchXY),
            )?;
            update.changed |= synchronize_sketch_view(window, owner)?;
            continue;
        }
        if *action == GuiShellAction::NewPart {
            update.changed |= report_authoring_result(window, owner.create_part_document())?;
            continue;
        }
        let event = ViewportEvent::Input(ViewportInput::Key {
            key: action_key(action).ok_or("GUI action: expected a keyboard command")?,
            pressed: true,
        });
        update.changed |= dispatch_frame_input(window, owner, &event)?;
    }
    Ok(update)
}
fn report_authoring_result(
    window: &mut dyn HeadWindow,
    result: Result<bool, String>,
) -> Result<bool, String> {
    match result {
        Ok(changed) => Ok(changed),
        Err(cause) => {
            window.show_command_notice(crate::DesktopCommandNotice::AuthoringFailed { cause })?;
            Ok(true)
        }
    }
}
fn action_key(action: &GuiShellAction) -> Option<ViewportKey> {
    Some(match action {
        GuiShellAction::Parameter { .. }
        | GuiShellAction::EditSketch { .. }
        | GuiShellAction::Rectangle
        | GuiShellAction::NewPart
        | GuiShellAction::StartSketchXY
        | GuiShellAction::FinishSketch => {
            return None;
        }
        GuiShellAction::Save => ViewportKey::Save,
        GuiShellAction::Undo => ViewportKey::Undo,
        GuiShellAction::Redo => ViewportKey::Redo,
        GuiShellAction::Home => ViewportKey::Home,
    })
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
