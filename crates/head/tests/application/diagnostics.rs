// SPDX-License-Identifier: GPL-2.0-only
//! Read the completed native GUI frame and its live windows.
pub(super) use super::drawn_windows::window_geometry;
use imgui::sys;
use std::ffi::CString;

#[path = "../../../../test-utilities/rust/drawn_text.rs"]
mod drawn_text;
pub(super) use drawn_text::{
    RenderedFrame, RenderedRun, has_strike, rendered_frame, run_is_checked,
};

/// Check for a visible image beside a caption in the same draw list.
#[cfg(target_os = "linux")]
pub(super) fn has_caption_icon(run: &RenderedRun) -> bool {
    use imgui::{DrawCmd, internal::RawCast};
    let Some(raw) = (unsafe { sys::igGetDrawData().as_ref() }) else {
        return false;
    };
    let draws = unsafe { imgui::DrawData::from_raw(raw) };
    let Some(list) = draws.draw_lists().nth(run.list) else {
        return false;
    };
    list.commands().any(|command| {
        let DrawCmd::Elements {
            count: 6,
            cmd_params,
        } = command
        else {
            return false;
        };
        let indices = &list.idx_buffer()[cmd_params.idx_offset..];
        let vertices = &list.vtx_buffer()[cmd_params.vtx_offset..];
        let [first, opposite] = [indices[0], indices[2]].map(|index| vertices[index as usize]);
        first.uv == [0., 0.]
            && opposite.uv == [1., 1.]
            && drawn_text::inside_clip(first, opposite, cmd_params.clip_rect)
            && (0. ..10.).contains(&(run.minimum[0] - opposite.pos[0]))
            && (first.pos[1]..opposite.pos[1]).contains(&run.center()[1])
    })
}

#[cfg(test)]
use drawn_text::{
    current_font_size, frame_of_draw_data, glyph_for_uv, glyph_lookup, inside_clip, quad_glyph,
};
#[cfg(test)]
use imgui::DrawVert;
#[cfg(test)]
use std::collections::BTreeMap;

pub(super) fn window_popup_open(window: &str, popup: &str) -> bool {
    let window = CString::new(window).expect("window: expected no NUL");
    let popup = CString::new(popup).expect("popup: expected no NUL");
    let Some(owner) = (unsafe { sys::igFindWindowByName(window.as_ptr()).as_ref() }) else {
        return false;
    };
    let Some(context) = (unsafe { sys::igGetCurrentContext().as_ref() }) else {
        return false;
    };
    let stack = &context.OpenPopupStack;
    if stack.Size <= 0 {
        return false;
    }
    let id = unsafe { sys::igImHashStr(popup.as_ptr(), 0, owner.ID) };
    unsafe { std::slice::from_raw_parts(stack.Data, stack.Size as usize) }
        .iter()
        .any(|entry| entry.PopupId == id)
}
/// Read a window's dock node; example: compare two visible tool windows after a drag.
pub(super) fn window_dock_id(title: &str) -> Option<u32> {
    let title = CString::new(title).ok()?;
    let window = unsafe { sys::igFindWindowByName(title.as_ptr()).as_ref() }?;
    window.Active.then_some(window.DockId)
}
/// Read the active state of a docked tab; example: `window_dock_tab_visible("Parameters")`.
pub(super) fn window_dock_tab_visible(title: &str) -> Option<bool> {
    let title = CString::new(title).ok()?;
    let window = unsafe { sys::igFindWindowByName(title.as_ptr()).as_ref() }?;
    (window.DockId != 0).then(|| window.DockTabIsVisible())
}
/// Read a docked tab's bounds; example: click its close control without closing the group.
pub(super) fn window_dock_tab_bounds(title: &str) -> Option<([f32; 2], [f32; 2])> {
    let title = CString::new(title).ok()?;
    let window = unsafe { sys::igFindWindowByName(title.as_ptr()).as_ref() }?;
    let rect = window.DockTabItemRect;
    (window.Active && window.DockId != 0)
        .then_some(([rect.Min.x, rect.Min.y], [rect.Max.x, rect.Max.y]))
}
/// Framebuffer bounds for a named GUI window, including display density.
/// Example: `window_pixel_bounds("##navigation-bar")` for screenshot exclusion.
pub(super) fn window_pixel_bounds(title: &str) -> Option<[f32; 4]> {
    let (position, size) = window_geometry(title)?;
    let scale = unsafe { sys::igGetIO().as_ref() }?.DisplayFramebufferScale;
    Some([
        position[0] * scale.x,
        position[1] * scale.y,
        (position[0] + size[0]) * scale.x,
        (position[1] + size[1]) * scale.y,
    ])
}
/// Map drawn text positions to native pointer pixels, including the current UI scale.
pub(super) fn pointer_pixels(position: [f32; 2]) -> [i32; 2] {
    let io = unsafe { sys::igGetIO().as_ref() }.expect("expected a live GUI context");
    [
        (position[0] * io.DisplayFramebufferScale.x) as i32,
        (position[1] * io.DisplayFramebufferScale.y) as i32,
    ]
}
/// Screen rectangles that belong to GUI windows, not to the model beneath them.
pub(super) fn window_occlusions() -> Vec<[f32; 4]> {
    let context =
        unsafe { sys::igGetCurrentContext().as_ref() }.expect("expected a live native GUI");
    let windows =
        unsafe { std::slice::from_raw_parts(context.Windows.Data, context.Windows.Size as usize) };
    let scale = context.IO.DisplayFramebufferScale;
    let dock_host = CString::new("##workspace-docking-host").unwrap();
    let dock_host = unsafe { sys::igFindWindowByName(dock_host.as_ptr()) };
    windows
        .iter()
        .flat_map(|window_ptr| {
            if *window_ptr == dock_host {
                return Vec::new();
            }
            let Some(window) = (unsafe { window_ptr.as_ref() }) else {
                return Vec::new();
            };
            if !window.Active || window.Hidden {
                return Vec::new();
            }
            let name = unsafe { std::ffi::CStr::from_ptr(window.Name) }.to_bytes();
            // Its children are transparent dockspace hosts; docked tools retain their own names.
            if name == b"##workspace-outer-host"
                || name.starts_with(b"##workspace-outer-host/")
                || name.starts_with(b"##workspace-docking-host/DockSpace_")
            {
                return Vec::new();
            }
            let bounds = window.OuterRectClipped;
            let mut rects = vec![[
                bounds.Min.x * scale.x,
                bounds.Min.y * scale.y,
                bounds.Max.x * scale.x,
                bounds.Max.y * scale.y,
            ]];
            if let Some(node) = unsafe { window.DockNode.as_ref() }
                && node.VisibleWindow == *window_ptr
                && !node.TabBar.is_null()
                && let Some(header) = dock_header_bounds(node, bounds.Min.y, scale)
            {
                rects.push(header);
            }
            rects
        })
        .collect()
}

fn dock_header_bounds(
    node: &sys::ImGuiDockNode,
    content_top: f32,
    scale: sys::ImVec2,
) -> Option<[f32; 4]> {
    let top = node.Pos.y;
    let bottom = content_top.min(top + node.Size.y);
    if node.Size.x <= 0. || bottom <= top {
        return None;
    }
    Some([
        node.Pos.x * scale.x,
        top * scale.y,
        (node.Pos.x + node.Size.x) * scale.x,
        bottom * scale.y,
    ])
}
#[cfg(test)]
#[path = "diagnostics_tests.rs"]
mod tests;
