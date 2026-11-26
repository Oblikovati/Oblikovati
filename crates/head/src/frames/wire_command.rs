// SPDX-License-Identifier: GPL-2.0-only
use super::{
    HeadWindow, HeadWork, gui::dispatch_action, publish_owner_scene, synchronize_sketch_view,
};
use oblikovati_api::{
    Error,
    wire::{ExecuteCommandArgs, METHOD_COMMANDS_EXECUTE, OKResult},
};
use oblikovati_automation::{decode_wire_arguments, encode_wire_reply};
use oblikovati_gui::{GuiShellAction, GuiViewCommand};

pub(crate) fn execute_command(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let args: ExecuteCommandArgs = decode_wire_arguments(METHOD_COMMANDS_EXECUTE, request)?;
    let action = GuiShellAction::from_command_id(&args.id).ok_or_else(|| {
        Error(format!(
            "desktop command {:?}: expected a supported command ID",
            args.id
        ))
    })?;
    if !action.dispatchable(&owner.gui_command_state().map_err(Error)?) {
        return Err(Error(format!(
            "desktop command {:?}: disabled in the current context",
            args.id
        )));
    }
    if matches!(
        action,
        GuiShellAction::Home | GuiShellAction::ViewCommand(GuiViewCommand::FitAll)
    ) {
        // An add-in may have created geometry earlier in this owner poll.
        synchronize_sketch_view(window, owner).map_err(Error)?;
        publish_owner_scene(window, owner).map_err(Error)?;
    }
    dispatch_action(window, owner, &action, |_, result| result).map_err(Error)?;
    encode_wire_reply(&OKResult { ok: true })
}
