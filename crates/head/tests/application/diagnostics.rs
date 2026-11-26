// SPDX-License-Identifier: GPL-2.0-only
//! Read the completed native GUI frame and its live windows.
pub(super) use super::drawn_windows::window_geometry;
use super::world::squeezed;
use imgui::sys;
use std::ffi::CString;

#[path = "../../../../test-utilities/rust/drawn_text.rs"]
mod drawn_text;
pub(super) use drawn_text::{
    RenderedFrame, RenderedRun, has_strike, rendered_frame, run_is_checked,
};

/// Locate the draw list belonging to the named active window.
pub(super) fn window_draw_list(title: &str) -> Option<usize> {
    let title = CString::new(title).unwrap();
    // The live frame owns the window and its draw list.
    let window = unsafe { sys::igFindWindowByName(title.as_ptr()).as_ref() }?;
    window
        .Active
        .then(|| rendered_draw_list_index(window.DrawList))?
}

pub(super) fn tooltip_draw_list() -> Option<usize> {
    (0..8).find_map(|index| window_draw_list(&format!("##Tooltip_{index:02}")))
}

#[cfg(target_os = "linux")]
pub(super) fn draw_list_window_bounds(index: usize) -> Option<[f32; 4]> {
    // Both the active windows and the draw lists belong to the completed frame.
    let context = unsafe { sys::igGetCurrentContext().as_ref() }?;
    let windows =
        unsafe { std::slice::from_raw_parts(context.Windows.Data, context.Windows.Size as usize) };
    let window = windows
        .iter()
        .filter_map(|window| unsafe { window.as_ref() })
        .find(|window| window.Active && rendered_draw_list_index(window.DrawList) == Some(index))?;
    Some([
        window.Pos.x,
        window.Pos.y,
        window.Pos.x + window.Size.x,
        window.Pos.y + window.Size.y,
    ])
}

/// Find the draw list of the active child at a point inside its bounds.
pub(super) fn child_draw_list_at(parent: &str, point: [f32; 2]) -> Option<usize> {
    let (draw_list, _) = active_child_at(parent, point);
    rendered_draw_list_index(draw_list)
}

/// Size of an active rendered child, measured from its actual outer bounds.
pub(super) fn child_size_at(parent: &str, point: [f32; 2]) -> [f32; 2] {
    active_child_at(parent, point).1
}

fn active_child_at(parent: &str, point: [f32; 2]) -> (*mut sys::ImDrawList, [f32; 2]) {
    let name = CString::new(parent).unwrap();
    // The window and draw-list pointers belong to the live context until the next frame.
    let owner = unsafe { sys::igFindWindowByName(name.as_ptr()).as_ref() }
        .unwrap_or_else(|| panic!("expected GUI window {parent:?}"));
    let children = &owner.DC.ChildWindows;
    assert!(children.Size > 0, "expected child windows in {parent:?}");
    let children = unsafe { std::slice::from_raw_parts(children.Data, children.Size as usize) };
    children
        .iter()
        .find_map(|window| {
            let window = unsafe { window.as_ref() }?;
            let bounds = window.OuterRectClipped;
            (window.Active
                && (bounds.Min.x..bounds.Max.x).contains(&point[0])
                && (bounds.Min.y..bounds.Max.y).contains(&point[1]))
            .then_some((window.DrawList, [window.Size.x, window.Size.y]))
        })
        .unwrap_or_else(|| panic!("expected active child at {point:?} in {parent:?}"))
}

#[derive(Clone, Copy)]
pub(super) struct ChildScroll {
    pub list: usize,
    pub origin: [f32; 2],
    pub size: [f32; 2],
    pub center: [f32; 2],
    pub content_right: f32,
    pub offset: f32,
    pub limit: f32,
    pub horizontal_scrollbar: bool,
    pub content_width: f32,
}

/// Read the scroll range and draw list of a named child in the completed frame.
pub(super) fn child_scroll(parent: &str, label: &str) -> Option<ChildScroll> {
    let name = CString::new(parent).ok()?;
    let label = CString::new(label).ok()?;
    // The GUI owner owns these pointers until the next frame. No reference escapes.
    let window = unsafe { sys::igFindWindowByName(name.as_ptr()) };
    let owner = unsafe { window.as_ref() }?;
    if !owner.Active || owner.DC.ChildWindows.Size <= 0 {
        return None;
    }
    let id = unsafe { sys::ImGuiWindow_GetID_Str(window, label.as_ptr(), std::ptr::null()) };
    let children = &owner.DC.ChildWindows;
    let children = unsafe { std::slice::from_raw_parts(children.Data, children.Size as usize) };
    let child = children
        .iter()
        .filter_map(|ptr| unsafe { ptr.as_ref() })
        .find(|child| child.Active && child.ChildId == id)?;
    let bounds = child.OuterRectClipped;
    Some(ChildScroll {
        list: rendered_draw_list_index(child.DrawList)?,
        origin: [child.Pos.x, child.Pos.y],
        size: [child.Size.x, child.Size.y],
        center: [
            (bounds.Min.x + bounds.Max.x) * 0.5,
            (bounds.Min.y + bounds.Max.y) * 0.5,
        ],
        content_right: child.InnerRect.Max.x,
        offset: child.Scroll.y,
        limit: child.ScrollMax.y,
        horizontal_scrollbar: child.ScrollbarX,
        content_width: child.ContentSize.x,
    })
}

fn rendered_draw_list_index(target: *mut sys::ImDrawList) -> Option<usize> {
    let frame = unsafe { sys::igGetDrawData().as_ref() }?;
    if frame.CmdListsCount < 1 {
        return None;
    }
    let lists = unsafe { std::slice::from_raw_parts(frame.CmdLists, frame.CmdListsCount as usize) };
    // ImGui can omit an empty child list when the parent draws its border.
    lists.iter().position(|&list| list == target)
}

pub(super) fn radio_mark_rendered(run: &RenderedRun) -> bool {
    use imgui::internal::RawCast;
    // The completed draw data and current style belong to the live test window.
    let Some(raw) = (unsafe { sys::igGetDrawData().as_ref() }) else {
        return false;
    };
    let draws = unsafe { imgui::DrawData::from_raw(raw) };
    let accent =
        unsafe { sys::igGetColorU32_Col(sys::ImGuiCol_CheckMark as i32, 1.) }.to_le_bytes();
    let height = drawn_text::current_font_size();
    draws.draw_lists().nth(run.list).is_some_and(|list| {
        list.vtx_buffer().iter().any(|vertex| {
            vertex.col == accent
                && (0. ..height * 2.5).contains(&(run.minimum[0] - vertex.pos[0]))
                && (run.center()[1] - vertex.pos[1]).abs() < height * 0.5
        })
    })
}

/// Check for a visible image beside a caption in the same draw list.
#[cfg(target_os = "linux")]
pub(super) fn has_caption_icon(run: &RenderedRun) -> bool {
    caption_icon_left(run).is_some()
}

pub(super) fn caption_icon_left(run: &RenderedRun) -> Option<f32> {
    use imgui::{DrawCmd, internal::RawCast};
    let raw = unsafe { sys::igGetDrawData().as_ref() }?;
    let draws = unsafe { imgui::DrawData::from_raw(raw) };
    let list = draws.draw_lists().nth(run.list)?;
    list.commands().find_map(|command| {
        let DrawCmd::Elements {
            count: 6,
            cmd_params,
        } = command
        else {
            return None;
        };
        let indices = &list.idx_buffer()[cmd_params.idx_offset..];
        let vertices = &list.vtx_buffer()[cmd_params.vtx_offset..];
        let [first, opposite] = [indices[0], indices[2]].map(|index| vertices[index as usize]);
        (first.uv == [0., 0.]
            && opposite.uv == [1., 1.]
            && drawn_text::inside_clip(first, opposite, cmd_params.clip_rect)
            && (0. ..10.).contains(&(run.minimum[0] - opposite.pos[0]))
            && (first.pos[1]..opposite.pos[1]).contains(&run.center()[1]))
        .then_some(first.pos[0])
    })
}

#[cfg(test)]
use drawn_text::{current_font_size, frame_of_draw_data, inside_clip, quad_glyph};
#[cfg(test)]
pub(super) use drawn_text::{glyph_for_uv, glyph_lookup};
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
pub(super) fn command_popup_open() -> bool {
    let context =
        unsafe { sys::igGetCurrentContext().as_ref() }.expect("expected a live GUI context");
    context.OpenPopupStack.Size > 0
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

/// Resolve an origin plane label inside the active picker when its browser row has the same text.
pub(super) fn plane_choice_center(frame: &RenderedFrame, text: &str) -> Option<[f32; 2]> {
    if !["XY Plane", "XZ Plane", "YZ Plane"].contains(&text) {
        return None;
    }
    let (position, size) = window_geometry("##plane-selection")?;
    let wanted = squeezed(text);
    frame.runs.iter().find_map(|run| {
        let center = run.center();
        let on_plane_picker = (0..2)
            .all(|axis| (position[axis]..position[axis] + size[axis]).contains(&center[axis]));
        (on_plane_picker && squeezed(&run.text) == wanted).then_some(center)
    })
}

/// Group captions use uppercase lettering with an optional separate arrow glyph.
pub(super) fn panel_caption_matches(rendered: &str, expected: &str) -> bool {
    let caption = rendered.strip_suffix(['v', '▼']).unwrap_or(rendered);
    let expected = squeezed(expected);
    caption == expected || caption == expected.to_ascii_uppercase()
}

pub(super) fn panel_caption_run<'a>(
    frame: &'a RenderedFrame,
    expected: &str,
) -> Option<&'a RenderedRun> {
    frame
        .runs
        .iter()
        .rev()
        .find(|run| panel_caption_matches(&run.text, expected))
}

pub(super) fn shows_run(frame: &RenderedFrame, expected: &str) -> bool {
    frame.runs.iter().any(|run| {
        squeezed(&run.text).contains(expected) || panel_caption_matches(&run.text, expected)
    })
}

pub(super) fn annotation_run_matches(rendered: &str, expected: &str) -> bool {
    rendered == expected
        || rendered
            .strip_prefix(expected)
            .is_some_and(|unit| matches!(unit, "mm" | "cm" | "in" | "ft" | "m" | "µm" | "km" | "°"))
}

pub(super) fn annotation_shown(frame: &RenderedFrame, expected: &str) -> bool {
    frame
        .runs
        .iter()
        .any(|run| annotation_run_matches(&run.text, expected))
}
