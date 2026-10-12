// SPDX-License-Identifier: GPL-2.0-only
/// The UI owner checks occupancy before allocating a context that could end another frame.
/// Example: `require_vacant_context()?` before `Context::create()` on the GUI thread.
pub(crate) fn require_vacant_context() -> Result<(), String> {
    // imgui-rs e246a334 Context::drop ends the current frame even for an inactive context.
    // Only the GUI owner thread accesses this binding; the raw address never escapes or changes.
    if !unsafe { imgui::sys::igGetCurrentContext() }.is_null() {
        return Err(
            "GUI context occupied: expected exclusive owner activation before creation".into(),
        );
    }
    Ok(())
}
