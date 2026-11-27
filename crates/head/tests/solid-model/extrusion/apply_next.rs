// SPDX-License-Identifier: GPL-2.0-only
use super::super::super::diagnostics;
use super::super::{
    direction::preview_depth, replace_extrusion_distance, viewport_interaction::click_at,
};
use super::{ApplicationWorld, click_in_extrude, feature_count, native_fixture_preview};
use cucumber::{given, then, when};
use oblikovati_gui::GuiExtrudeDirection;

#[given("a feature Properties panel has a valid preview in a disposable part")]
fn start_extrusion(world: &mut ApplicationWorld) {
    native_fixture_preview(world);
    replace_extrusion_distance(world, "8");
    click_in_extrude(world, "Flipped");
    click_in_extrude(world, "Direction");
    let row = world
        .frame
        .runs
        .iter()
        .find(|row| row.text == "Dropdown" && world.frame.is_popup_row(row))
        .unwrap();
    click_at(world, diagnostics::pointer_pixels(row.center()));
    world.pump(Vec::new());
    assert_eq!(preview_depth(world), [-0.8, 0.]);
    assert_eq!(feature_count(world), 0);
    assert!(world.run_is_enabled("OK"));
    assert_eq!(
        world.state().extrude_draft_preview.unwrap().direction,
        GuiExtrudeDirection::Negative
    );
}

#[when("the user clicks Apply and create new in the footer")]
fn apply_in_footer(world: &mut ApplicationWorld) {
    click_settled(world, "+");
}

#[then("the current feature is committed once and appears in the model browser")]
fn first_extrusion_committed(world: &mut ApplicationWorld) {
    assert_eq!(feature_count(world), 1);
    world.expect_text("Extrusion1");
    assert_eq!(preview_depth(world), [-0.8, 0.]);
}

#[then(
    "Properties stays open for another feature with the carried configuration and fresh input requirements"
)]
fn configuration_carried(world: &mut ApplicationWorld) {
    world.run_in_window("Extrude", "Properties");
    world.run_in_window("Extrude", "Chooseasketch");
    world.run_in_window("Extrude", "SelectProfile");
    world.run_in_window("Extrude", "8");
    world.run_in_window("Extrude", "Flipped");
    assert!(!world.run_is_enabled("OK"));
    assert!(!world.state().extrude_preview_active);
    world.save_capture("extrude-next-draft");
}

#[when("the user cancels the second feature")]
fn cancel_next_draft(world: &mut ApplicationWorld) {
    click_settled(world, "Cancel");
}

#[then("the first feature remains committed and no second feature is added")]
fn first_feature_retained(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Extrude").is_none());
    first_extrusion_committed(world);
    world.expect_no_text("Extrusion2");
}

pub(super) fn click_settled(world: &mut ApplicationWorld, label: &str) {
    let point = |world: &ApplicationWorld| {
        diagnostics::pointer_pixels(world.run_in_window("Extrude", label).center())
    };
    world.hover(point(world));
    click_at(world, point(world));
    world.pump(Vec::new());
}
