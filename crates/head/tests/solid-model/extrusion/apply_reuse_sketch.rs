// SPDX-License-Identifier: GPL-2.0-only
use super::super::distance_editor::dimension_pick::{
    open_dimension_package, open_rectangle_preview,
};
use super::super::viewport_interaction::click_at;
use super::apply_next::click_settled;
use super::{ApplicationWorld, click_in_extrude, feature_count};
use cucumber::{given, then, when};

const FIXTURE: &str = "fixtures/cad/extrusion/extrusion-dimension-pick.opd";

#[given("a fresh copy of `fixtures/cad/extrusion/extrusion-dimension-pick.opd` is open")]
fn open_two_profile_sketch(world: &mut ApplicationWorld) {
    let source = super::super::super::scenario_root().join(FIXTURE);
    open_dimension_package(world, &std::fs::read(source).unwrap());
}

#[given("the user opens Extrude with the rectangular profile selected and a valid preview")]
fn rectangular_draft(world: &mut ApplicationWorld) {
    open_rectangle_preview(world);
    assert!(world.run_is_enabled("OK"));
    assert_eq!(world.state().extrude_draft_preview.unwrap().profile, 0);
}

#[given("a part-feature draft uses one profile in a sketch with another eligible profile")]
fn open_multi_profile_draft(world: &mut ApplicationWorld) {
    open_two_profile_sketch(world);
    rectangular_draft(world);
}

#[given("Keep sketches visible on (+) is checked in Advanced Settings")]
fn keep_sketch_checked(world: &mut ApplicationWorld) {
    super::preset_visibility::open_settings(world);
    assert!(world.checked_run(super::preset_visibility::menu_row(
        world,
        "Keepsketchesvisibleon(+)"
    )));
    world.escape();
    world.pump(Vec::new());
}

#[when("the user clicks Apply and create new")]
fn apply_first_extrusion(world: &mut ApplicationWorld) {
    click_settled(world, "+");
}

#[then("the first feature is committed")]
#[then("the first extrusion is committed")]
fn one_extrusion_committed(world: &mut ApplicationWorld) {
    assert_eq!(feature_count(world), 1);
    world.expect_text("Extrusion1");
}

#[then("the sketch remains visible for selecting a profile for the next feature")]
#[then(
    "the same sketch remains visible with the separate circular profile available for selection"
)]
fn awaiting_new_profile(world: &mut ApplicationWorld) {
    let state = world.state();
    let browser = state.browser.unwrap();
    assert_eq!(browser.sketches.len(), 1);
    assert!(browser.sketches[0].visible);
    assert_eq!(state.undo_labels.len(), 1);
    assert_eq!(browser.bodies[0].faces, 6);
    assert!(!state.extrude_preview_active);
    assert!(!world.run_is_enabled("OK"));
    world.run_in_window("Extrude", "SelectProfile");
    world.run("3mm");
}

#[when("the user selects the circular profile for the next draft")]
fn pick_circular_region(world: &mut ApplicationWorld) {
    let state_before = world.state();
    for (world_point, expected_preview) in [([2., 0., 0.], false), ([3., 0., 0.], true)] {
        let pixel = world
            .app()
            .window
            .native
            .view
            .model_pixel(world_point)
            .unwrap();
        world.hover(pixel);
        click_at(world, pixel);
        world.pump(Vec::new());
        let state = world.state();
        assert_eq!(
            state.extrude_preview_active,
            expected_preview,
            "{world_point:?}: {:?}",
            world.app().window.native.command_notice
        );
        assert_eq!(state.document_generation, state_before.document_generation);
        assert_eq!(state.undo_labels, state_before.undo_labels);
        let browser = state.browser.unwrap();
        let before = state_before.browser.as_ref().unwrap();
        assert_eq!(browser.features, before.features);
        assert_eq!(browser.bodies, before.bodies);
    }
}

#[then("the second draft uses that profile without changing the first extrusion")]
fn circular_draft(world: &mut ApplicationWorld) {
    let state = world.state();
    let draft = state.extrude_draft_preview.expect("circular preview");
    assert_eq!(draft.profile, 1);
    let browser = state.browser.unwrap();
    assert_eq!(draft.sketch, browser.sketches[0].id);
    assert_eq!(browser.bodies.len(), 1);
    assert!(world.run_is_enabled("OK"));
    world.run_in_window("Extrude", "1Profile");
    one_extrusion_committed(world);
    world.save_capture("extrude-apply-reused-sketch");
}

#[test]
fn native_extrude_reuses_a_visible_sketch_after_apply() {
    let mut world = ApplicationWorld::new();
    open_dimension_package(
        &mut world,
        include_bytes!(
            "../../../../../test-utilities/fixtures/extrusion/extrusion-dimension-pick.opd"
        ),
    );
    rectangular_draft(&mut world);
    keep_sketch_checked(&mut world);
    apply_first_extrusion(&mut world);
    one_extrusion_committed(&mut world);
    awaiting_new_profile(&mut world);
    pick_circular_region(&mut world);
    circular_draft(&mut world);
}
