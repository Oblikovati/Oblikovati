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

/// Queue focus with the other input so Dear ImGui releases its held state in order.
/// Example: `queue_focus(io, false)` when the native window loses focus.
pub(crate) fn queue_focus(io: &mut imgui::Io, focused: bool) {
    use imgui::internal::RawCast;
    // Io implements RawCast for this exact pinned binding; the pointer is used only by this call.
    unsafe { imgui::sys::ImGuiIO_AddFocusEvent(io.raw_mut(), focused) };
}

/// Copy current activity after Render without exposing the context; example: `frame_activity()?`.
pub(crate) fn frame_activity() -> Result<crate::GuiFrameActivity, String> {
    // imgui-rs 0.12.0 omits the deferred input queue from its safe Io view.
    // The exclusive GUI owner holds the active context; no native pointers escape this read.
    let context = unsafe { imgui::sys::igGetCurrentContext().as_ref() }
        .ok_or("GUI activity: expected a live exclusive context after Render")?;
    Ok(crate::GuiFrameActivity {
        queued_input: context.InputEventsQueue.Size > 0,
        pending_layout: pending_window_layout(context)?,
        pending_navigation: context.NavMoveSubmitted
            || context.NavInitRequest
            || context.NavInitResultId != 0
            || context.NavNextActivateId != 0,
        interaction: interactive_context(context),
    })
}
fn interactive_context(context: &imgui::sys::ImGuiContext) -> bool {
    context.ActiveId != 0
        || context.HoveredId != 0
        || context.IO.WantTextInput
        || context.IO.MouseDown.iter().any(|down| *down)
        || !context.MovingWindow.is_null()
        || context.WheelingWindowReleaseTimer > 0.
}
fn pending_window_layout(context: &imgui::sys::ImGuiContext) -> Result<bool, String> {
    let windows = &context.Windows;
    if windows.Size < 0 || (windows.Size > 0 && windows.Data.is_null()) {
        return Err(format!(
            "GUI window count {}: expected a nonnegative owned window array",
            windows.Size
        ));
    }
    for index in 0..windows.Size as usize {
        // ImGui owns this array and every window while the exclusive owner reads it after Render.
        let window = unsafe { (*windows.Data.add(index)).as_ref() }
            .ok_or_else(|| format!("GUI window {index}: expected a live owned window"))?;
        if window_needs_layout(window) {
            return Ok(true);
        }
    }
    Ok(false)
}

fn window_needs_layout(window: &imgui::sys::ImGuiWindow) -> bool {
    window.Active
        && (window.AutoFitFramesX > 0
            || window.AutoFitFramesY > 0
            || window.HiddenFramesCannotSkipItems > 0)
}
#[cfg(test)]
#[path = "../tests/activity/native.rs"]
mod tests;

#[cfg(test)]
#[path = "../tests/clipboard/native.rs"]
mod clipboard_tests;
