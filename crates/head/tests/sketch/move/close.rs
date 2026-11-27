// SPDX-License-Identifier: GPL-2.0-only
//! Check closing Precise Input independently of the Move option.
use super::{ApplicationWorld, diagnostics, move_pin, move_precise};
use cucumber::{given, then, when};
use oblikovati_gui::GuiSketchMoveStage;

#[given("the precise toolbar is visible in Move Base Point mode")]
fn precise_move(world: &mut ApplicationWorld) {
    world.remember_edited_sketch(vec![move_precise::EDGE]);
    move_precise::select_ellipse(world);
    assert!(diagnostics::window_geometry("Precise Input").is_some());
}

#[when("the user opens its right menu")]
fn open_menu(world: &mut ApplicationWorld) {
    move_pin::open_settings(world);
}

#[then("Pin Mini-Toolbar Position, Auto Fade and closes are listed")]
fn menu_items(world: &mut ApplicationWorld) {
    for text in ["Pin Mini-Toolbar Position", "Auto Fade", "Close"] {
        world.expect_text(text);
    }
}

#[then("the toolbar and triad disappear while Move remains open")]
fn toolbar_hidden(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Precise Input").is_none());
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Base
    );
    let origin = world.sketch_pixel_of([0., 0.]);
    world.expect_segment_absent(origin, [origin[0] + 24, origin[1]]);
}

#[then("Precise Input remains checked")]
fn option_remains_enabled(world: &mut ApplicationWorld) {
    let label = world.run_in_window("Move", "PreciseInput");
    assert!(
        world.checked_run(label),
        "Move Precise Input must stay checked"
    );
}

#[then("Move closes without accepting placement")]
fn no_placement(world: &mut ApplicationWorld) {
    assert!(world.state().sketch_move.is_none());
    assert!(crate::HeadSketchWork::sketch_preview(&world.app().owner).is_none());
    world.expect_edited_sketch_retained();
}

#[test]
fn closed_toolbar_reopens_through_move_checkbox() {
    let mut world = ApplicationWorld::new();
    move_precise::editable_ellipse(&mut world);
    precise_move(&mut world);
    move_pin::open_settings(&mut world);
    world.click_text("Close");
    toolbar_hidden(&mut world);
    option_remains_enabled(&mut world);
    toggle_move_precise(&mut world);
    assert!(!world.checked_run(world.run_in_window("Move", "PreciseInput")));
    toggle_move_precise(&mut world);
    option_remains_enabled(&mut world);
    assert!(diagnostics::window_geometry("Precise Input").is_some());
    world.escape();
    no_placement(&mut world);
    world.close();
}

fn toggle_move_precise(world: &mut ApplicationWorld) {
    let check = world.run_in_window("Move", "PreciseInput").center();
    world.click(diagnostics::pointer_pixels(check));
}

#[test]
fn ribbon_reopens_toolbar_and_point_tool_keeps_its_behavior() {
    let mut world = ApplicationWorld::new();
    move_precise::editable_ellipse(&mut world);
    precise_move(&mut world);
    move_pin::open_settings(&mut world);
    world.click_text("Close");
    option_remains_enabled(&mut world);
    world.open_precise_input();
    assert!(diagnostics::window_geometry("Precise Input").is_some());
    option_remains_enabled(&mut world);
    world.escape();
    no_placement(&mut world);
    world.select_point_tool();
    assert!(diagnostics::window_geometry("Precise Input").is_some());
    world.close_precise_input();
    world.open_precise_input();
    assert!(diagnostics::window_geometry("Precise Input").is_some());
    world.escape();
    world.expect_edited_sketch_retained();
    world.close();
}
