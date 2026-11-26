// SPDX-License-Identifier: GPL-2.0-only
//! Keep exclusive part feature drafts consistent with the dock lifecycle.
use super::*;

impl GuiFrameEngine {
    /// Attach Sweep presets to the active desktop preference profile.
    /// Example: `engine.set_sweep_preset_store(store)?`.
    pub fn set_sweep_preset_store(
        &mut self,
        store: Box<dyn crate::GuiSweepPresetStore>,
    ) -> Result<(), String> {
        self.sweep_dialog.set_preset_store(store)
    }
}

#[derive(Clone, Copy, PartialEq, Eq)]
enum Draft {
    Extrude,
    Sweep,
    Loft,
    Blend,
    Shell,
    Hole,
    MoveBody,
}

fn requested_draft(action: &GuiShellAction) -> Option<Draft> {
    Some(match action {
        GuiShellAction::Sweep(crate::GuiSweepAction::Open | crate::GuiSweepAction::OpenEdit(_)) => {
            Draft::Sweep
        }
        GuiShellAction::Loft(crate::GuiLoftAction::Open) => Draft::Loft,
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
    (shell, hole): (
        &mut crate::shell_feature_dialog::ShellFeatureDialog,
        &mut crate::hole_dialog::HoleDialog,
    ),
    (motion, blend): (
        &mut crate::move_body_dialog::MoveBodyDialog,
        &mut crate::blend_dialog::BlendDialog,
    ),
    (sweep, loft): (
        &mut crate::sweep_dialog::SweepDialog,
        &mut crate::loft_dialog::LoftDialog,
    ),
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
    if draft != Draft::Sweep {
        sweep.close(actions);
    }
    if draft != Draft::Loft {
        loft.close(actions);
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
        GuiShellAction::Sweep(crate::GuiSweepAction::Open) => sweep.open(state, None),
        GuiShellAction::Sweep(crate::GuiSweepAction::OpenEdit(feature)) => {
            sweep.open(state, Some(feature))
        }
        GuiShellAction::Loft(crate::GuiLoftAction::Open) => loft.open(state, actions),
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
