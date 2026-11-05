// SPDX-License-Identifier: GPL-2.0-only
use super::{HeadWindow, HeadWork, ViewportEvent, synchronize_sketch_view};
use oblikovati_gui::GuiShellAction;
use oblikovati_viewport::{ViewportInput, ViewportKey};
use std::time::Duration;
#[path = "gui_authoring.rs"]
mod authoring;
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
        update.changed |= dispatch_gui_action(window, owner, action)?;
    }
    Ok(update)
}
fn dispatch_gui_action(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    action: &GuiShellAction,
) -> Result<bool, String> {
    if let Some(key) = action_key(action) {
        let event = ViewportEvent::Input(ViewportInput::Key { key, pressed: true });
        return dispatch_frame_input(window, owner, &event);
    }
    let changed = report_authoring_result(window, authoring::apply(owner, action))?;
    Ok(synchronize_sketch_view(window, owner)? || changed)
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
        GuiShellAction::EditThemeColor { .. }
        | GuiShellAction::DuplicateTheme { .. }
        | GuiShellAction::DeleteTheme { .. }
        | GuiShellAction::SaveTheme { .. }
        | GuiShellAction::SetColorScheme(_)
        | GuiShellAction::SetTheme(_)
        | GuiShellAction::Parameter { .. }
        | GuiShellAction::ActivateDocument(_)
        | GuiShellAction::OpenDocument(_)
        | GuiShellAction::CloseDocument(_)
        | GuiShellAction::SaveDocumentAs(_)
        | GuiShellAction::EditSketch { .. }
        | GuiShellAction::PlaceSketchTool(_)
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
