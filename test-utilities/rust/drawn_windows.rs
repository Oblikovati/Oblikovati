// SPDX-License-Identifier: GPL-2.0-only
use imgui::sys;
use std::ffi::CString;

/// Read a visible window's origin and size. Example: `window_geometry("##quick-access-below")`.
pub(crate) fn window_geometry(title: &str) -> Option<([f32; 2], [f32; 2])> {
    let title =
        CString::new(title).unwrap_or_else(|_| panic!("window title {title:?}: expected no NUL"));
    // SAFETY: the frame owner calls this while its ImGui context owns the named window.
    let window = unsafe { sys::igFindWindowByName(title.as_ptr()).as_ref() }?;
    window
        .Active
        .then_some(([window.Pos.x, window.Pos.y], [window.Size.x, window.Size.y]))
}
