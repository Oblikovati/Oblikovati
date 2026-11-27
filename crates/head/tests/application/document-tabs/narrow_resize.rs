// SPDX-License-Identifier: GPL-2.0-only
//! Resize a populated native tab strip while keeping the selected Part in view.
use super::super::{desktop_input, diagnostics};
use super::*;

const SAVED_PART_PREFIX: &str = "docking-visible-solid-current";

#[when(
    "the user narrows the main application window so the tab strip cannot fit all tab labels at full width"
)]
fn narrow_populated_strip(world: &mut ApplicationWorld) {
    let saved = world.active_id();
    world.previous_document = Some(saved);
    for _ in 0..4 {
        world.add_part();
    }
    world.click_text("Documents");
    let name = world.document_name(saved);
    world.click_text(&name);
    world.expect_active(saved);
    let full_width = world.expect_tab(SAVED_PART_PREFIX);
    assert!(!full_width.text.contains("..."));
    world.observed_rows = Some(full_width.text);
    let [width, height] = world.extent().map(|value| value as i32);
    desktop_input::drag(world, [width - 2, height - 2], [660, 550]);
    world.pump(Vec::new());
    let [width, height] = world.extent();
    assert!(
        width.abs_diff(660) <= 4 && height.abs_diff(550) <= 4,
        "resized to {width}x{height}"
    );
}

#[then("the active document tab remains visible in the tab strip using middle truncation")]
fn active_tab_is_shortened(world: &mut ApplicationWorld) {
    let saved = world.previous_document.unwrap();
    assert_eq!(world.active_id(), saved);
    let label = world.expect_tab("docking");
    let full_name = world.observed_rows.as_deref().unwrap();
    let (start, end) = label
        .text
        .split_once("...")
        .unwrap_or_else(|| panic!("expected middle truncation, saw {:?}", label.text));
    assert!(
        !start.is_empty() && full_name.starts_with(start),
        "{label:?}"
    );
    assert!(!end.is_empty() && full_name.ends_with(end), "{label:?}");
    assert!(
        tab_has_active_underline(world, &label.text),
        "expected the active document underline"
    );
}

/// Read the actual accent under a visible tab, for example `tab_has_active_underline(world, "Home")`.
pub(super) fn tab_has_active_underline(world: &mut ApplicationWorld, name: &str) -> bool {
    let label = world.expect_tab(name);
    let (position, size) = window_geometry(TAB_STRIP).unwrap();
    let [x, y] = diagnostics::pointer_pixels([label.center()[0], position[1] + size[1] - 2.])
        .map(|value| value as usize);
    let underline = world.pixel_at([x, y]);
    let accent =
        unsafe { imgui::sys::igGetColorU32_Col(imgui::sys::ImGuiCol_NavHighlight as i32, 1.) }
            .to_le_bytes();
    underline[..3] == accent[..3]
}

#[then("tab overflow navigation controls remain reachable alongside the Documents button")]
fn overflow_controls_are_reachable(world: &mut ApplicationWorld) {
    for name in ["<", ">", "Documents"] {
        assert!(world.run_is_enabled(name), "expected {name} enabled");
    }
    let before = world.expect_tab("docking").text;
    world.save_capture("narrow-active-document-tab");
    world.click_text(">");
    world.pump(Vec::new());
    assert!(
        world.tab_run(&before).is_none(),
        "right arrow should scroll away from the active tab"
    );
    world.click_text("<");
    world.pump(Vec::new());
    world.expect_tab(&before);
    world.click_text("Documents");
    world.expect_text("Close Tabs");
    assert_eq!(Some(world.active_id()), world.previous_document);
}
