// SPDX-License-Identifier: GPL-2.0-only
use std::ffi::c_char;

use imgui::{ImString, Ui, WindowFlags};

unsafe extern "C" {
    fn oblikovati_gui_begin_scroll_region(identity: *const c_char, width: f32, height: f32)
    -> bool;
    fn oblikovati_gui_begin_child_region(
        identity: *const c_char,
        width: f32,
        height: f32,
        content_width: f32,
        flags: i32,
    ) -> bool;
}

/// A live child window is always ended, including clipped frames and unwinding.
pub(crate) struct ScrollingRegion<'ui> {
    pub visible: bool,
    _ui: &'ui Ui,
}

impl Drop for ScrollingRegion<'_> {
    fn drop(&mut self) {
        // SAFETY: this token retains the UI that opened exactly one child, including a false BeginChild result.
        unsafe { imgui::sys::igEndChild() };
    }
}

/// Open scalar-sized scrolling content; example: `let region = scroll_region(ui, "panels", [640.,80.]);`.
pub(crate) fn scroll_region<'ui>(
    ui: &'ui Ui,
    identity: &str,
    size: [f32; 2],
) -> ScrollingRegion<'ui> {
    let identity = ImString::new(identity);
    // SAFETY: the UI owns the context and C++ consumes the live ID and scalar dimensions synchronously (#3803, #3849).
    let visible =
        unsafe { oblikovati_gui_begin_scroll_region(identity.as_ptr(), size[0], size[1]) };
    ScrollingRegion { visible, _ui: ui }
}

/// Size a child and its contents independently, preserving the numeric child ID used by `Ui::child_window`.
pub(crate) fn child_region<'ui>(
    ui: &'ui Ui,
    identity: &str,
    size: [f32; 2],
    content_width: f32,
    flags: WindowFlags,
) -> ScrollingRegion<'ui> {
    let identity = ImString::new(identity);
    // SAFETY: the UI owns the context and C++ constructs these vectors without a by-value aggregate ABI (#3803).
    let visible = unsafe {
        oblikovati_gui_begin_child_region(
            identity.as_ptr(),
            size[0],
            size[1],
            content_width,
            flags.bits() as i32,
        )
    };
    ScrollingRegion { visible, _ui: ui }
}
