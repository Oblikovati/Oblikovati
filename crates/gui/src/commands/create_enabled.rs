// SPDX-License-Identifier: GPL-2.0-only
//! Enablement checks for Part Create feature dialog actions.
use crate::{GuiCommandState, GuiShellAction};

pub(crate) fn enabled(action: &GuiShellAction, state: &GuiCommandState) -> Option<bool> {
    Some(match action {
        GuiShellAction::Revolve(action) => action.enabled(state),
        GuiShellAction::Sweep(action) => action.enabled(state),
        GuiShellAction::Loft(action) => action.enabled(state),
        GuiShellAction::Coil(action) => action.enabled(state),
        GuiShellAction::Emboss(action) => action.enabled(state),
        GuiShellAction::Derive(action) => action.enabled(state),
        GuiShellAction::Rib(action) => action.enabled(state),
        GuiShellAction::Decal(action) => action.enabled(state),
        GuiShellAction::Import(action) => action.enabled(state),
        GuiShellAction::Unwrap(action) => action.enabled(state),
        GuiShellAction::Hole(action) => action.enabled(state),
        GuiShellAction::Slot(action) => action.enabled(state),
        GuiShellAction::Combine(action) => action.enabled(state),
        GuiShellAction::Draft(action) => action.enabled(state),
        GuiShellAction::Thread(action) => action.enabled(state),
        GuiShellAction::DeleteFace(action) => action.enabled(state),
        GuiShellAction::ThickenOffset(action) => action.enabled(state),
        GuiShellAction::DirectEdit(action) => action.enabled(state),
        GuiShellAction::Split(action) => action.enabled(state),
        GuiShellAction::Mark(action) => action.enabled(state),
        GuiShellAction::Finish(action) => action.enabled(state),
        GuiShellAction::RuledSurface(action) => action.enabled(state),
        GuiShellAction::WorkAxis(action) => action.enabled(state),
        GuiShellAction::Ucs(action) => action.enabled(state),
        GuiShellAction::WorkPoint(action) => action.enabled(state),
        GuiShellAction::WorkPlane(action) => action.enabled(state),
        GuiShellAction::ExtendSurface(action) => action.enabled(state),
        GuiShellAction::BoundaryPatch(action) => action.enabled(state),
        GuiShellAction::FilletV2(action) => action.enabled(state),
        GuiShellAction::ChamferV2(action) => action.enabled(state),
        GuiShellAction::ShellV2(action) => action.enabled(state),
        _ => return None,
    })
}

pub(crate) fn jump_history_enabled(state: &GuiCommandState, document: u64, position: i64) -> bool {
    !state.transaction_open
        && state.histories.iter().any(|history| {
            history.document == document
                && position >= 0
                && usize::try_from(position)
                    .ok()
                    .is_some_and(|pos| pos <= history.entries.as_ref().map_or(0, Vec::len))
        })
}
