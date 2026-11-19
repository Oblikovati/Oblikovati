// SPDX-License-Identifier: GPL-2.0-only
//! Keep exclusive part feature drafts consistent with the dock lifecycle.
use super::*;
pub(super) fn open_drafts(
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    shell: &mut crate::shell_feature_dialog::ShellFeatureDialog,
    hole: &mut crate::hole_dialog::HoleDialog,
    motion: &mut crate::move_body_dialog::MoveBodyDialog,
    blend: &mut crate::blend_dialog::BlendDialog,
) {
    if actions.contains(&GuiShellAction::OpenShellFeature) {
        motion.close(actions);
        hole.close(actions);
        blend.close(actions);
        shell.open(state);
        actions.retain(|action| !matches!(action, GuiShellAction::OpenShellFeature));
    }
    if actions.contains(&GuiShellAction::OpenMoveBody) {
        blend.close(actions);
        hole.close(actions);
        shell.close(actions);
        motion.open(state);
        actions.retain(|action| !matches!(action, GuiShellAction::OpenMoveBody));
    }
    if actions.contains(&GuiShellAction::OpenHole) {
        motion.close(actions);
        blend.close(actions);
        shell.close(actions);
        hole.open(state);
        actions.retain(|action| !matches!(action, GuiShellAction::OpenHole));
    }
    if actions.contains(&GuiShellAction::OpenFillet) {
        motion.close(actions);
        hole.close(actions);
        shell.close(actions);
        if blend.active() {
            blend.close(actions);
        }
        blend.open(state);
        actions.retain(|action| !matches!(action, GuiShellAction::OpenFillet));
    }
    if actions.contains(&GuiShellAction::OpenChamfer) {
        motion.close(actions);
        hole.close(actions);
        shell.close(actions);
        if blend.active() {
            blend.close(actions);
        }
        blend.open_chamfer(state);
        actions.retain(|action| !matches!(action, GuiShellAction::OpenChamfer));
    }
}
