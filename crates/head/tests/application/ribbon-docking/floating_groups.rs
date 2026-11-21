// SPDX-License-Identifier: GPL-2.0-only
//! Floating groups scroll and change with the selected tab.
use super::*;
use cucumber::{given, then, when};
const PANE: &str = "##ribbon-temporary-panel";

fn floating_label(world: &ApplicationWorld, label: &str) -> [i32; 2] {
    let run = world.run_in_window(FLOATING, &label.replace(' ', ""));
    super::super::diagnostics::pointer_pixels(run.center())
}

fn floating_group(world: &ApplicationWorld, label: &str) -> [i32; 2] {
    let (position, size) = window_geometry(FLOATING).expect("expected floating ribbon");
    let name = label.replace(' ', "");
    let run = world
        .frame
        .runs
        .iter()
        .find(|run| {
            let [x, y] = run.center();
            run.text == name
                && (position[0] + size[0] * 0.5..position[0] + size[0]).contains(&x)
                && (position[1]..position[1] + size[1]).contains(&y)
        })
        .unwrap_or_else(|| panic!("{label}: expected a visible floating group"));
    super::super::diagnostics::pointer_pixels(run.center())
}

fn click_floating_label(world: &mut ApplicationWorld, label: &str) {
    let at = floating_label(world, label);
    #[cfg(target_os = "linux")]
    super::super::x11_input::click(at);
    #[cfg(not(target_os = "linux"))]
    world.click(at);
    world.pump(Vec::new());
}

#[given("the floating 3D Model ribbon shows Start 2D Sketch above the lower groups")]
fn floating_starts_with_sketch(world: &mut ApplicationWorld) {
    undock_ribbon(world);
    expect_floating(world);
    let start = floating_label(world, "Start 2D Sketch");
    let modify = floating_label(world, "Modify");
    assert!(start[1] < modify[1]);
    assert!(world.run_is_enabled("Start 2D Sketch"));
}

#[when("the user scrolls down in the floating ribbon group list")]
fn scroll_floating_groups(world: &mut ApplicationWorld) {
    let at = floating_label(world, "Modify");
    #[cfg(target_os = "linux")]
    super::super::x11_input::scroll(at, -2);
    #[cfg(not(target_os = "linux"))]
    world.pump(vec![oblikovati_viewport::ViewportEvent::Input(
        oblikovati_viewport::ViewportInput::Wheel {
            notches: -2,
            position: at,
        },
    )]);
    world.pump(Vec::new());
}

#[then("Start 2D Sketch scrolls out of view")]
fn sketch_is_clipped(world: &mut ApplicationWorld) {
    world.expect_no_text("Start 2D Sketch");
}

#[then("Shape Generator, Work Features, Pattern, and Create Freeform are visible")]
fn lower_groups_are_visible(world: &mut ApplicationWorld) {
    let labels = [
        "Shape Generator",
        "Work Features",
        "Pattern",
        "Create Freeform",
    ];
    let positions = labels.map(|label| floating_group(world, label));
    assert!(positions.windows(2).all(|pair| pair[0][1] < pair[1][1]));
    for label in ["Shape Generator", "Create Freeform"] {
        assert!(!world.run_is_enabled(label));
    }
}

#[given("a part is active and the ribbon is floating with 3D Model selected")]
fn floating_model_part(world: &mut ApplicationWorld) {
    part_active(world);
    undock_ribbon(world);
    floating_label(world, "Modify");
}

#[when("the user selects the Annotate tab on the floating ribbon")]
#[when("the user selects Annotate")]
fn select_annotate(world: &mut ApplicationWorld) {
    click_floating_label(world, "Annotate");
}

#[then(
    "the ribbon displays Geometric Annotation, General Annotation, Notes, Manage, and Tolerance Analysis groups"
)]
#[then("the ribbon displays its annotation groups")]
fn annotation_groups(world: &mut ApplicationWorld) {
    let labels = [
        "Geometric Annotation",
        "General Annotation",
        "Notes",
        "Manage",
        "Tolerance Analysis",
    ];
    let positions = labels.map(|label| floating_group(world, label));
    assert!(positions.windows(2).all(|pair| pair[0][1] < pair[1][1]));
    world.expect_no_text("Modify");
}

#[when("the user selects the Notes group")]
#[when("the user selects Notes")]
fn select_notes(world: &mut ApplicationWorld) {
    click_floating_label(world, "Notes");
}

#[then("an adjacent panel displays Leader Text, General Note, and General Profile Note commands")]
#[then("the adjacent panel lists Leader Text, General Note and General Profile Note")]
fn note_entries(world: &mut ApplicationWorld) {
    let (ribbon, width) = window_geometry(FLOATING).expect("expected floating ribbon");
    let (pane, _) = window_geometry(PANE).expect("expected Notes pane");
    assert!(pane[0] >= ribbon[0] + width[0]);
    let labels = ["LeaderText", "GeneralNote", "GeneralProfileNote"];
    for label in labels {
        let run = world.run_in_window(PANE, label);
        assert!(
            run.alpha < u8::MAX,
            "{label}: expected an unavailable command"
        );
    }
    let notes = floating_label(world, "Notes");
    let [red, green, blue, _] = world.pixel_at([notes[0] as usize, (notes[1] - 12) as usize]);
    assert!(blue > green && green > red, "expected a blue Notes button");
}

#[when("the user selects 3D Model")]
fn restore_model_tab(world: &mut ApplicationWorld) {
    click_floating_label(world, "3D Model");
}

#[then("the modeling groups return")]
fn floating_model_groups(world: &mut ApplicationWorld) {
    model_groups_again(world);
    same_part(world);
}

#[cfg(target_os = "linux")]
#[test]
fn x11_floating_start_sketch_selects_a_plane_directly() {
    let mut world = ApplicationWorld::new();
    part_active(&mut world);
    floating_starts_with_sketch(&mut world);
    click_floating_label(&mut world, "Start 2D Sketch");
    assert!(window_geometry(PANE).is_none());
    world.click_text("XY Plane");
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    same_part(&mut world);
    world.close();
}
