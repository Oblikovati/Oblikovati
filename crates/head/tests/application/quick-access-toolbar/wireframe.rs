// SPDX-License-Identifier: GPL-2.0-only
//! Native evidence for the quick-access icon wireframe (#4062).
use super::super::{diagnostics, quick_access_icons};
use super::*;
use cucumber::{given, then, when};

fn icon(label: &str) -> [i32; 2] {
    diagnostics::pointer_pixels(
        quick_access_icons::standard_icon_center(label)
            .unwrap_or_else(|| panic!("toolbar icon {label:?}: expected the three default icons")),
    )
}

#[given("New, Open and Save are visible on the quick access toolbar")]
fn standard_icons_visible(_: &mut ApplicationWorld) {
    let images = quick_access_icons::toolbar_images();
    assert_eq!(images.len(), 3, "expected New, Open and Save icons");
    assert_ne!(images[0].texture, images[1].texture);
    assert_ne!(images[0].texture, images[2].texture);
    assert_ne!(images[1].texture, images[2].texture);
}

#[then("New, Open and Save have distinct command icons without persistent captions")]
fn icons_without_captions(world: &mut ApplicationWorld) {
    standard_icons_visible(world);
    for label in ["New", "Open", "Save"] {
        assert!(
            !world.frame.runs.iter().any(|run| run.text == label),
            "unexpected persistent {label} caption"
        );
    }
}

#[when(regex = r"^the user hovers over the (New|Open|Save) icon until its tooltip appears$")]
fn hover_icon(world: &mut ApplicationWorld, command: String) {
    #[cfg(target_os = "linux")]
    super::super::x11_input::move_pointer(icon(&command));
    #[cfg(not(target_os = "linux"))]
    world.pump(vec![super::super::pointer_motion(icon(&command))]);
    world.pump(Vec::new());
}

#[then(regex = r"^the tooltip identifies the command as (New|Open|Save)$")]
fn tooltip_identifies(world: &mut ApplicationWorld, command: String) {
    world.expect_text(&command);
}

#[then("the part remains active without a command dialog")]
fn part_remains_without_dialog(world: &mut ApplicationWorld) {
    assert!(world.state().document.is_some());
    for title in ["Create New File", "Open", "Save As"] {
        assert!(
            diagnostics::window_geometry(title).is_none(),
            "unexpected {title} dialog"
        );
    }
}

#[when("the user moves the pointer to an empty area of the viewport")]
fn pointer_leaves(world: &mut ApplicationWorld) {
    #[cfg(target_os = "linux")]
    super::super::x11_input::move_pointer([1100, 650]);
    #[cfg(not(target_os = "linux"))]
    world.pump(vec![super::super::pointer_motion([1100, 650])]);
    world.pump(Vec::new());
}

#[then("the tooltip disappears")]
fn tooltip_disappears(world: &mut ApplicationWorld) {
    for label in ["New", "Open", "Save"] {
        assert!(
            !world.frame.runs.iter().any(|run| run.text == label),
            "tooltip {label} remains visible"
        );
    }
}

#[test]
fn native_application_rows_show_icon_commands_above_ribbon_tabs() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.save_capture("qat-wireframe-4062");
    let file = world.run("File");
    let tab = world.run("3D Model");
    assert!((file.center()[1] - tab.center()[1]).abs() < 16.);
    let icons = quick_access_icons::toolbar_images();
    assert_eq!(icons.len(), 3);
    assert!(icons.iter().all(|icon| icon.center[1] < file.minimum[1]));
    icons_without_captions(&mut world);
    world.click_text("View");
    world.save_capture("qat-view-row-4062");
    world.close();
}
