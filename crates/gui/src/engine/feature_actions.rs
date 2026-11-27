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

impl crate::GuiFrameEngine {
    /// Open a revolution draft for the active part. Example: `engine.request_revolve()`.
    pub fn request_revolve(&mut self) {
        self.revolve_dialog.request_open();
    }
}

pub(super) fn resolve_menu_extrude(commands: &GuiCommandState, actions: &mut [GuiShellAction]) {
    for action in actions {
        if *action == GuiShellAction::OpenExtrude {
            *action = crate::extrude_dialog::start_action(commands);
        }
    }
}

pub(super) fn finish_frame_capture(
    capture: &mut crate::GuiInputCapture,
    ribbon: &crate::ribbon::DocumentRibbon,
    events: &[crate::GuiInputEvent],
    (popup_open, dock_escape): (bool, bool),
    pointer_flags: (bool, bool, bool),
) {
    let has_text = events
        .iter()
        .any(|event| matches!(event, crate::GuiInputEvent::Text(_)));
    capture.text |= ribbon.tips.consumed || (ribbon.tips.active() && has_text);
    capture.popup = popup_open
        || dock_escape
        || ribbon.tips.consumed
        || ribbon.customization.escape_consumed
        || ribbon.display.temporary_escape_consumed;
    capture.pointer |= pointer_flags.0 || pointer_flags.1 || pointer_flags.2;
}

#[derive(Clone, Copy, PartialEq, Eq)]
enum Draft {
    Extrude,
    Revolve,
    Sweep,
    Loft,
    Blend,
    Shell,
    Hole,
    MoveBody,
}

fn requested_draft(action: &GuiShellAction) -> Option<Draft> {
    Some(match action {
        GuiShellAction::Revolve(
            crate::GuiRevolveAction::Open | crate::GuiRevolveAction::OpenEdit(_),
        ) => Draft::Revolve,
        GuiShellAction::Sweep(crate::GuiSweepAction::Open | crate::GuiSweepAction::OpenEdit(_)) => {
            Draft::Sweep
        }
        GuiShellAction::Loft(crate::GuiLoftAction::Open | crate::GuiLoftAction::OpenEdit(_)) => {
            Draft::Loft
        }
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
    (sweep, loft, revolve): (
        &mut crate::sweep_dialog::SweepDialog,
        &mut crate::loft_dialog::LoftDialog,
        &mut crate::revolve_dialog::RevolveDialog,
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
    if draft != Draft::Revolve {
        revolve.close(actions);
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
        GuiShellAction::Revolve(crate::GuiRevolveAction::Open) => revolve.open(state, None),
        GuiShellAction::Revolve(crate::GuiRevolveAction::OpenEdit(feature)) => {
            revolve.open(state, Some(feature))
        }
        GuiShellAction::Sweep(crate::GuiSweepAction::Open) => sweep.open(state, None),
        GuiShellAction::Sweep(crate::GuiSweepAction::OpenEdit(feature)) => {
            sweep.open(state, Some(feature))
        }
        GuiShellAction::Loft(crate::GuiLoftAction::Open) => loft.open(state, None, actions),
        GuiShellAction::Loft(crate::GuiLoftAction::OpenEdit(feature)) => {
            loft.open(state, Some(feature), actions)
        }
        GuiShellAction::OpenExtrude if state.authoring == crate::GuiAuthoringContext::Sketch => {
            extrude.close(actions);
            extrude.wait_for_sketch(state.document);
        }
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
