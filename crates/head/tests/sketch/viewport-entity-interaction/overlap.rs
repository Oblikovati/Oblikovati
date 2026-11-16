// SPDX-License-Identifier: GPL-2.0-only
//! Choose between overlapping sketch entities through the native viewport (#4041, #4042).
use super::super::super::ApplicationWorld;
use cucumber::{given, then, when};

#[given("an editable sketch contains a curve and a selectable point at the same screen location")]
fn point_on_line(world: &mut ApplicationWorld) {
    super::add_isolated_line(world);
    super::add_overlap_point(world);
    world.remember_edited_sketch(Vec::new());
}

#[when("the user clicks the overlapping location")]
fn click_overlap(world: &mut ApplicationWorld) {
    let position = world.sketch_pixel_of([0., 1.]);
    super::native_click(world, position);
}

#[then("the MCAD Software identifies the picked entity type")]
fn picked_point_is_named(world: &mut ApplicationWorld) {
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let selected = world.state().sketch_selection.entities;
    assert_eq!(selected.len(), 1);
    assert_eq!(selected[0].0, edit.sketch);
    assert!(world.edited_geometry().point(selected[0].1).is_ok());
    world.expect_text("1. Point (Sketch1)");
    world.save_capture("overlap-point-cue-4042");
}

#[then("any available pick-choice control identifies the alternatives")]
fn select_other_identifies_and_selects_line(world: &mut ApplicationWorld) {
    open_select_other(world);
    let point = submenu_candidate(world, "Point");
    let line = submenu_candidate(world, "Line");
    assert_ne!(point, line);
    world.save_capture("overlap-select-other-4041");
    super::native_click(world, line);
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let entity = world.edited_geometry().lines()[0].id;
    assert_eq!(
        world.state().sketch_selection.entities,
        [(edit.sketch, entity)]
    );
    world.expect_edited_sketch_retained();
}

fn open_select_other(world: &mut ApplicationWorld) {
    let position = world.sketch_pixel_of([0., 1.]);
    #[cfg(target_os = "linux")]
    super::super::super::x11_input::right_click(position);
    #[cfg(not(target_os = "linux"))]
    world.right_click(position);
    world.pump(Vec::new());
    let center = world.run("Select Other").center();
    super::native_click(
        world,
        super::super::super::diagnostics::pointer_pixels(center),
    );
    world.pump(Vec::new());
}

fn submenu_candidate(world: &ApplicationWorld, label: &str) -> [i32; 2] {
    let row = world.run("Select Other");
    let candidate = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == label && run.minimum[0] > row.maximum[0])
        .unwrap_or_else(|| {
            panic!(
                "candidate {label:?}: expected it in Select Other; got {:?}",
                world.text()
            )
        });
    super::super::super::diagnostics::pointer_pixels(candidate.center())
}
