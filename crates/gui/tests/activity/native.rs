// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[derive(Default)]
struct FakeActivityWindows {
    context: imgui::sys::ImGuiContext,
    window: imgui::sys::ImGuiWindow,
}
#[test]
fn native_activity_reads_require_owned_context_and_window_entries() {
    assert!(
        frame_activity()
            .unwrap_err()
            .contains("live exclusive context")
    );
    let mut fixture = FakeActivityWindows::default();
    assert!(!pending_window_layout(&fixture.context).unwrap());
    fixture.context.Windows.Size = -1;
    assert!(pending_window_layout(&fixture.context).is_err());
    fixture.context.Windows.Size = 1;
    assert!(pending_window_layout(&fixture.context).is_err());
    let mut entry = std::ptr::null_mut();
    fixture.context.Windows.Data = &mut entry;
    assert!(pending_window_layout(&fixture.context).is_err());
    entry = &mut fixture.window;
    assert!(!entry.is_null());
    fixture.window.Active = true;
    fixture.window.HiddenFramesCanSkipItems = 1;
    assert!(!pending_window_layout(&fixture.context).unwrap());
    fixture.window.HiddenFramesCannotSkipItems = 1;
    assert!(pending_window_layout(&fixture.context).unwrap());
    fixture.window.Active = false;
    assert!(!pending_window_layout(&fixture.context).unwrap());
}
#[test]
fn native_activity_includes_each_interactive_timer_source() {
    let mut fixture = FakeActivityWindows::default();
    assert!(!interactive_context(&fixture.context));
    fixture.context.ActiveId = 1;
    assert!(interactive_context(&fixture.context));
    fixture.context.ActiveId = 0;
    fixture.context.HoveredId = 1;
    assert!(interactive_context(&fixture.context));
    fixture.context.HoveredId = 0;
    fixture.context.IO.WantTextInput = true;
    assert!(interactive_context(&fixture.context));
    fixture.context.IO.WantTextInput = false;
    fixture.context.IO.MouseDown[0] = true;
    assert!(interactive_context(&fixture.context));
    fixture.context.IO.MouseDown[0] = false;
    fixture.context.MovingWindow = &mut fixture.window;
    assert!(interactive_context(&fixture.context));
    fixture.context.MovingWindow = std::ptr::null_mut();
    fixture.context.WheelingWindowReleaseTimer = 0.1;
    assert!(interactive_context(&fixture.context));
}

#[test]
fn only_active_windows_with_pending_layout_request_another_frame() {
    let mut fixture = FakeActivityWindows::default();
    fixture.window.Active = true;
    assert!(!window_needs_layout(&fixture.window));
    fixture.window.AutoFitFramesX = 1;
    assert!(window_needs_layout(&fixture.window));
    fixture.window.AutoFitFramesX = 0;
    fixture.window.AutoFitFramesY = 1;
    assert!(window_needs_layout(&fixture.window));
    fixture.window.AutoFitFramesY = 0;
    fixture.window.HiddenFramesCannotSkipItems = 1;
    assert!(window_needs_layout(&fixture.window));
    fixture.window.Active = false;
    assert!(!window_needs_layout(&fixture.window));
}
