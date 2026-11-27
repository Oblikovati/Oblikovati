// SPDX-License-Identifier: GPL-2.0-only
//! Drive weighted circular Loft takeoff through the native Conditions tab.
use super::*;
use oblikovati_model::feature::{part::PartFeatureDefinition, sketch_loft::LoftTakeoffCondition};

#[given("a surface Loft made from two circular sections")]
fn create_pinned_surface(world: &mut ApplicationWorld) {
    surface::create_loft_surface_for_edit(world);
}

#[when("LoftSrf1 is opened for editing and the alternate first-section condition is selected")]
fn choose_angled_condition(world: &mut ApplicationWorld) {
    reopen_conditions(world);
    world.run_in_window("Loft", "0.000ul");
    click_loft_control(world, "Free");
    let option = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Angle" && world.frame.is_popup_row(run))
        .expect("expected Angle in the native section condition menu");
    world.click(diagnostics::pointer_pixels(option.center()));
    world.pump(Vec::new());
}

#[then("its weight changes from zero to one")]
fn weight_defaults_to_one(world: &mut ApplicationWorld) {
    world.run_in_window("Loft", "1.000ul");
    assert_eq!(world.state().loft_preview, Some(Ok(())));
    assert!(world.state().loft_takeoff_dirty);
    assert!(saved_takeoff(world).is_none());
}

#[when("the first-section weight is replaced with 2 and OK is clicked")]
fn save_weight_two(world: &mut ApplicationWorld) {
    edit_number(world, "1.000ul", "2");
    assert_eq!(world.state().loft_preview, Some(Ok(())));
    assert!(world.run_is_enabled("OK"));
    let key = surface::loft_surface_key(world);
    accept_loft(world);
    assert_eq!(surface::loft_surface_key(world), key);
    assert_eq!(
        saved_takeoff(world),
        LoftTakeoffCondition::new(90., 2.).ok()
    );
}

#[when("LoftSrf1 is reopened on the Conditions tab")]
#[when("the feature is reopened on Conditions")]
fn reopen_conditions(world: &mut ApplicationWorld) {
    let row = diagnostics::pointer_pixels(world.run("LoftSrf1").center());
    world.pump(crate::native::gui_loop_tests::pointer_click_events(row, 2));
    world.pump(Vec::new());
    assert_eq!(world.state().loft_preview, Some(Ok(())));
    click_loft_control(world, "Conditions");
}

#[then("the alternate condition and weight 2.000 ul are shown for Sketch1")]
fn weight_two_is_visible(world: &mut ApplicationWorld) {
    for label in ["Sketch1", "Angle", "2.000ul"] {
        world.run_in_window("Loft", label);
    }
}

#[given(
    "the user edits LoftSrf1, selects the alternate first-section condition, sets its weight to 2 and accepts"
)]
fn prepare_weight_two(world: &mut ApplicationWorld) {
    choose_angled_condition(world);
    save_weight_two(world);
}

#[given("LoftSrf1 has the alternate first-section condition and weight 2")]
fn prepare_pinned_weight_two(world: &mut ApplicationWorld) {
    surface::create_loft_surface_for_edit(world);
    prepare_weight_two(world);
}

#[when("its first-section angle is changed from 90 to 45 degrees and OK is clicked")]
fn save_angle_45(world: &mut ApplicationWorld) {
    reopen_conditions(world);
    assert!((surface_radius(world) - 1.).abs() < 0.005);
    let key = surface::loft_surface_key(world);
    edit_number(world, "90.00deg", "45");
    assert_eq!(world.state().loft_preview, Some(Ok(())));
    assert!(world.run_is_enabled("OK"));
    assert!(surface_radius(world) > 1.15);
    let capture = capture_framebuffer(&mut world.app().window.native);
    assert!(solid_visibility::visible_preview_fill(
        &capture,
        &diagnostics::window_occlusions()
    ));
    accept_loft(world);
    assert_eq!(surface::loft_surface_key(world), key);
    assert_eq!(
        saved_takeoff(world),
        LoftTakeoffCondition::new(45., 2.).ok()
    );
}

#[then("the resulting surface is visibly reshaped")]
fn committed_surface_changes_radius(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Loft").is_none());
    assert!(surface_radius(world) > 1.15);
    assert!(world.state().browser.unwrap().surface_bodies[0].visible);
    world.save_capture("weighted-loft-angle");
    let key = surface::loft_surface_key(world);
    let path = world.scratch_folder().join("conditioned-loft.opd");
    solid_extrusion::acceptance::save_and_reopen(world, &path);
    world.pump(Vec::new());
    assert_eq!(surface::loft_surface_key(world), key);
    assert!(surface_radius(world) > 1.15);
}

#[then("Sketch1 shows 45.00 degrees and weight 2.000 ul")]
fn saved_angle_and_weight_are_visible(world: &mut ApplicationWorld) {
    weight_two_is_visible(world);
    world.run_in_window("Loft", "45.00deg");
}

fn edit_number(world: &mut ApplicationWorld, previous: &str, replacement: &str) {
    let position = diagnostics::pointer_pixels(world.run_in_window("Loft", previous).center());
    world.click(position);
    world.key(oblikovati_viewport::ViewportPhysicalKey::A, true);
    world.type_text(replacement);
    world.key(oblikovati_viewport::ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}

fn saved_takeoff(world: &mut ApplicationWorld) -> Option<LoftTakeoffCondition> {
    let state = world.state();
    let document = state.document.unwrap();
    let feature = state.browser.unwrap().surface_bodies[0].feature;
    let snapshot = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    let PartFeatureDefinition::SketchLoft(definition) =
        snapshot.work_geometry.get(feature).unwrap().definition()
    else {
        panic!("expected LoftSrf1 to have a Loft recipe");
    };
    definition.recipe().takeoff
}

fn surface_radius(world: &mut ApplicationWorld) -> f32 {
    world
        .app()
        .window
        .scene_triangles
        .iter()
        .flat_map(|triangle| triangle.positions)
        .map(|[x, y, _]| x.hypot(y))
        .fold(0.0, f32::max)
}
