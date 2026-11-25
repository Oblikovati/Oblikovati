// SPDX-License-Identifier: GPL-2.0-only
//! Keep exclusive part feature drafts consistent with the dock lifecycle.
use super::*;

#[derive(Clone, Copy, PartialEq, Eq)]
enum Draft {
    Extrude,
    Blend,
    Shell,
    Hole,
    MoveBody,
}

fn requested_draft(action: &GuiShellAction) -> Option<Draft> {
    Some(match action {
        GuiShellAction::OpenExtrude | GuiShellAction::OpenExtrudeEdit(_) => Draft::Extrude,
        GuiShellAction::OpenFillet
        | GuiShellAction::OpenChamfer
        | GuiShellAction::OpenFilletEdit(_)
        | GuiShellAction::OpenChamferEdit(_) => Draft::Blend,
        GuiShellAction::OpenShellFeature => Draft::Shell,
        GuiShellAction::OpenHole => Draft::Hole,
        GuiShellAction::OpenMoveBody => Draft::MoveBody,
        _ => return None,
    })
}

pub(super) fn open_drafts(
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    shell: &mut crate::shell_feature_dialog::ShellFeatureDialog,
    hole: &mut crate::hole_dialog::HoleDialog,
    motion: &mut crate::move_body_dialog::MoveBodyDialog,
    blend: &mut crate::blend_dialog::BlendDialog,
    extrude: &mut crate::extrude_dialog::ExtrudeDialog,
) {
    let requested = actions
        .iter()
        .rev()
        .find(|action| requested_draft(action).is_some());
    let last = requested
        .cloned()
        .or_else(|| extrude.requested.then_some(GuiShellAction::OpenExtrude));
    let Some(action) = last else { return };
    let Some(draft) = requested_draft(&action) else {
        return;
    };
    actions.retain(|action| {
        requested_draft(action).is_none() || *action == GuiShellAction::OpenExtrude
    });
    if draft != Draft::Extrude || matches!(action, GuiShellAction::OpenExtrudeEdit(_)) {
        extrude.close(actions);
    }
    if draft != Draft::Shell {
        shell.close(actions);
    }
    if draft != Draft::Hole {
        hole.close(actions);
    }
    if draft != Draft::MoveBody {
        motion.close(actions);
    }
    if blend.active() {
        blend.close(actions);
    }
    match action {
        GuiShellAction::OpenExtrude => extrude.requested = true,
        GuiShellAction::OpenExtrudeEdit(feature) => extrude.open_edit(state, feature),
        GuiShellAction::OpenFillet => blend.open(state),
        GuiShellAction::OpenChamfer => blend.open_chamfer(state),
        GuiShellAction::OpenFilletEdit(feature) => blend.open_edit(state, feature),
        GuiShellAction::OpenChamferEdit(feature) => blend.open_chamfer_edit(state, feature),
        GuiShellAction::OpenShellFeature => shell.open(state),
        GuiShellAction::OpenHole => hole.open(state),
        GuiShellAction::OpenMoveBody => motion.open(state),
        _ => {}
    }
}
