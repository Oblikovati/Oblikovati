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
        GuiShellAction::FilletV2(action) => action.enabled(state),
        GuiShellAction::ChamferV2(action) => action.enabled(state),
        _ => return None,
    })
}
