// SPDX-License-Identifier: GPL-2.0-only
//! Native semantics for the prompts that protect an application close.
use super::frame::{FrameControl, control_id, frame_button};
use accesskit::{Node, NodeId, Rect, Role};

pub(super) const DIALOG: NodeId = NodeId(5);
pub(super) fn dialog_nodes(prompt: &oblikovati_gui::GuiClosePrompt) -> Vec<(NodeId, Node)> {
    let mut dialog = Node::new(Role::Dialog);
    dialog.set_label(prompt.title);
    dialog.set_description(&prompt.message);
    dialog.set_modal();
    let bounds = prompt.bounds.map(f64::from);
    dialog.set_bounds(Rect::new(bounds[0], bounds[1], bounds[2], bounds[3]));
    dialog.set_children(
        prompt
            .buttons
            .iter()
            .map(|button| control_id(FrameControl::Answer(button.choice)))
            .collect::<Vec<_>>(),
    );
    let buttons = prompt.buttons.iter().map(|button| {
        let id = control_id(FrameControl::Answer(button.choice));
        (
            id,
            frame_button(button.label, button.bounds.map(f64::from), true),
        )
    });
    std::iter::once((DIALOG, dialog)).chain(buttons).collect()
}
pub(super) fn focused_dialog_control(prompt: &oblikovati_gui::GuiClosePrompt) -> NodeId {
    prompt
        .buttons
        .iter()
        .find(|button| button.focused)
        .map_or(DIALOG, |button| {
            control_id(FrameControl::Answer(button.choice))
        })
}
