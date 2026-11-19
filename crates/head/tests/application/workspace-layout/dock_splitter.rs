// SPDX-License-Identifier: GPL-2.0-only
//! Observe the real ImGui dock split beside a browser (#3897).
use imgui::sys;
use std::ffi::CString;

/// Read the visible horizontal splitter beside a docked window, in pixels.
/// Example: `right_splitter("Model", 359, 361)` for a left browser.
pub(super) fn right_splitter(
    title: &str,
    browser_right: i32,
    viewport_left: i32,
) -> Option<[i32; 2]> {
    let name = CString::new(title).ok()?;
    let window = unsafe { sys::igFindWindowByName(name.as_ptr()).as_ref() }?;
    if !window.Active || window.DockId == 0 {
        return None;
    }
    let scale = unsafe { sys::igGetIO().as_ref() }?
        .DisplayFramebufferScale
        .x;
    let mut node = unsafe { sys::igDockBuilderGetNode(window.DockId).as_ref() }?;
    for _ in 0..32 {
        let parent = unsafe { node.ParentNode.as_ref() }?;
        if let Some(bounds) = horizontal_split(parent, scale)
            && splitter_matches(bounds, browser_right, viewport_left)
        {
            return Some(bounds);
        }
        node = parent;
    }
    None
}

fn horizontal_split(parent: &sys::ImGuiDockNode, scale: f32) -> Option<[i32; 2]> {
    if parent.SplitAxis != sys::ImGuiAxis_X || !scale.is_finite() || scale <= 0. {
        return None;
    }
    let first = unsafe { parent.ChildNodes[0].as_ref() }?;
    let second = unsafe { parent.ChildNodes[1].as_ref() }?;
    Some([
        ((first.Pos.x + first.Size.x) * scale).round() as i32,
        (second.Pos.x * scale).round() as i32,
    ])
}

fn splitter_matches(bounds: [i32; 2], browser_right: i32, viewport_left: i32) -> bool {
    bounds[0] < bounds[1]
        && (bounds[0] - browser_right).abs() <= 1
        && (bounds[1] - viewport_left).abs() <= 1
}

#[test]
fn matching_requires_the_actual_splitter_to_join_both_regions() {
    assert!(splitter_matches([359, 361], 359, 361));
    assert!(!splitter_matches([359, 361], 359, 365));
    assert!(!splitter_matches([359, 359], 359, 359));
}
