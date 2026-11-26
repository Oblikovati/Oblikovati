// SPDX-License-Identifier: GPL-2.0-only
//! Native evidence for the quick-access icon wireframe (#4062).
use super::super::{diagnostics, quick_access_icons};
use super::*;
use cucumber::{given, then, when};
fn active_tooltip() -> Option<([f32; 2], [f32; 2])> {
    (0..8).find_map(|index| diagnostics::window_geometry(&format!("##Tooltip_{index:02}")))
}

fn icon(label: &str) -> [i32; 2] {
    diagnostics::pointer_pixels(
        quick_access_icons::standard_icon_center(label)
            .unwrap_or_else(|| panic!("toolbar icon {label:?}: expected a visible standard icon")),
    )
}

#[given(
    regex = r"^the (New|Open|Save|Undo|Redo|Home|Projects|Update|Select|Material Browser|Materials|Appearance Browser|Parameters) shortcut is visible on the quick access toolbar$"
)]
fn visible_shortcut(world: &mut ApplicationWorld, command: String) {
    icon(&command);
    world.observed_toolbar_command = Some(command);
}

#[then("it has a distinct command icon without a persistent text caption")]
fn distinct_icon(world: &mut ApplicationWorld) {
    let images = quick_access_icons::toolbar_images();
    let textures: std::collections::BTreeSet<_> =
        images.iter().map(|image| image.texture).collect();
    assert_eq!(images.len(), 13);
    assert_eq!(
        textures.len(),
        13,
        "each default command must use its own icon"
    );
    let command = world.observed_toolbar_command.as_deref().unwrap();
    let center = quick_access_icons::standard_icon_center(command).unwrap();
    assert!(
        !world
            .frame
            .runs
            .iter()
            .any(|run| run.text == command && (run.center()[1] - center[1]).abs() < 12.),
        "unexpected toolbar caption {command}"
    );
}

#[given("New, Open and Save are visible on the quick access toolbar")]
fn standard_icons_visible(_: &mut ApplicationWorld) {
    let images = quick_access_icons::toolbar_images();
    assert!(images.len() >= 3, "expected New, Open and Save icons");
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

#[when(
    regex = r"^the user hovers over the (New|Open|Save|Undo|Redo|Home|Projects|Update|Select|Material Browser|Materials|Appearance Browser|Parameters) icon until its tooltip appears$"
)]
fn hover_icon(world: &mut ApplicationWorld, command: String) {
    hover(world, icon(&command));
}

#[then(
    regex = r"^the tooltip identifies the command as (New|Open|Save|Undo|Redo|Home|Projects|Update|Select|Material Browser|Materials|Appearance Browser|Parameters)$"
)]
fn tooltip_identifies(world: &mut ApplicationWorld, command: String) {
    let (position, size) = active_tooltip().expect("expected a visible tooltip window");
    let expected = command.replace(' ', "");
    assert!(
        world.frame.runs.iter().any(|run| {
            let center = run.center();
            run.text.replace(' ', "") == expected
                && (0..2).all(|axis| {
                    center[axis] >= position[axis] && center[axis] < position[axis] + size[axis]
                })
        }),
        "tooltip for {command:?} was not rendered inside {position:?} / {size:?}"
    );
}

#[then("the part remains active without a command dialog")]
fn part_remains_without_dialog(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected the fixture Part");
    world.expect_active_by_id(part);
    assert!(
        !diagnostics::command_popup_open(),
        "unexpected command popup"
    );
}

#[when("the user moves the pointer to an empty area of the viewport")]
fn pointer_leaves(world: &mut ApplicationWorld) {
    hover(world, [1100, 650]);
}

fn hover(world: &mut ApplicationWorld, point: [i32; 2]) {
    #[cfg(target_os = "linux")]
    super::super::desktop_input::hover(world, point);
    #[cfg(not(target_os = "linux"))]
    world.pump(vec![super::super::pointer_motion(point)]);
}

#[then("the tooltip disappears")]
fn tooltip_disappears(_: &mut ApplicationWorld) {
    assert!(
        active_tooltip().is_none(),
        "tooltip remains visible after pointer leaves"
    );
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
    assert_eq!(icons.len(), 13);
    assert!(icons.iter().all(|icon| icon.center[1] < file.minimum[1]));
    icons_without_captions(&mut world);
    world.click_text("View");
    world.save_capture("qat-view-row-4062");
    world.close();
}
