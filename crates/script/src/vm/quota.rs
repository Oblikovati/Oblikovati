// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use mlua::{HookTriggers, VmState, debug::DebugEvent};
pub(super) fn install(lua: &Lua, limits: &crate::ScriptLimits) -> mlua::Result<Rc<Cell<u64>>> {
    let accounted = Rc::new(Cell::new(0_u64));
    let count = accounted.clone();
    let limits = limits.clone();
    lua.set_hook(
        HookTriggers {
            on_calls: true,
            every_nth_instruction: Some(1_000),
            ..Default::default()
        },
        move |lua, event| {
            if event.event() == DebugEvent::Count {
                count.set(count.get().saturating_add(1_000));
            }
            check_quota(lua, &limits, count.get())?;
            Ok(VmState::Continue)
        },
    )?;
    Ok(accounted)
}
fn check_quota(lua: &Lua, limits: &crate::ScriptLimits, instructions: u64) -> mlua::Result<()> {
    if instructions > limits.instructions {
        return Err(failure(format!(
            "instructions {instructions}: expected at most {} (1,000-instruction accounting)",
            limits.instructions
        )));
    }
    if lua.inspect_stack(limits.stack_frames, |_| ()).is_some() {
        return Err(failure(format!(
            "script stack: expected fewer than {} frames",
            limits.stack_frames
        )));
    }
    Ok(())
}
