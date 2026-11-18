// SPDX-License-Identifier: GPL-2.0-only
//! Open existing blend drafts before drawing their dockable window.
use super::*;
pub(super) fn open_blend_edits(
    dialog: &mut crate::blend_dialog::BlendDialog,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
) {
    for action in actions.iter() {
        match action {
            GuiShellAction::OpenFilletEdit(feature) => dialog.open_edit(state, *feature),
            GuiShellAction::OpenChamferEdit(feature) => dialog.open_chamfer_edit(state, *feature),
            _ => {}
        }
    }
    actions.retain(|action| {
        !matches!(
            action,
            GuiShellAction::OpenFilletEdit(_) | GuiShellAction::OpenChamferEdit(_)
        )
    });
}
