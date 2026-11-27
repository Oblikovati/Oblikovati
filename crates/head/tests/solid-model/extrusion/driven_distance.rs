// SPDX-License-Identifier: GPL-2.0-only
use super::super::{diagnostics, rendered_label_in_rect};
use super::acceptance::{feature_count, feature_distance, open_extrusion_editor};
use super::direction::preview_depth;
use super::*;

#[given("a fresh copy of `fixtures/cad/extrusion/profile-depth-4mm.opd` is open")]
fn open_driven_part(world: &mut ApplicationWorld) {
    let source = super::super::scenario_root().join("fixtures/cad/extrusion/profile-depth-4mm.opd");
    let copy = world.scratch_folder().join("profile-depth-4mm.opd");
    std::fs::copy(source, &copy).unwrap();
    world.open_fixture(&copy);
}

#[given("the user makes Sketch1 visible and selects its rectangular profile")]
fn choose_rectangular_profile(world: &mut ApplicationWorld) {
    let state = world.state();
    let browser = state.browser.expect("a part browser");
    let sketch = browser
        .sketches
        .iter()
        .find(|row| row.name == "Sketch1")
        .unwrap();
    if !sketch.visible {
        world.right_click(diagnostics::pointer_pixels(world.run("Sketch1").center()));
        world.click_text("Visibility");
    }
    world.click_text("Sketch1");
    world.pump(Vec::new());
    let browser = world.state().browser.unwrap();
    assert!(
        browser
            .sketches
            .iter()
            .any(|row| row.name == "Sketch1" && row.visible)
    );
}

#[when("the user starts a fixed-distance extrusion and enters Depth * 2 into Distance A")]
fn enter_expression(world: &mut ApplicationWorld) {
    world.click_text("Extrude");
    world.click(super::distance_a_input(world));
    world.key(ViewportPhysicalKey::A, true);
    world.type_text("Depth * 2");
    world.pump(Vec::new());
    let checked = world
        .state()
        .extrude_distance_validation
        .expect("validated expression");
    assert_eq!(checked.distance, "Depth * 2");
    assert!(world.window_text("Extrude").contains("Depth*2"));
}

#[then("the preview has a depth of 8 mm and OK is enabled")]
fn expression_preview(world: &mut ApplicationWorld) {
    assert!(world.state().extrude_preview_active);
    assert!(world.run_is_enabled("OK"));
    assert_eq!(preview_depth(world), [0., 0.8]);
    assert_eq!(feature_count(world), 0);
}

#[when("the user accepts the feature and changes Depth to 5 mm in the part parameters")]
fn update_driver(world: &mut ApplicationWorld) {
    world.click_text("OK");
    world.pump(Vec::new());
    assert_eq!(feature_count(world), 1);
    assert!((feature_distance(world) - 8.).abs() < 1e-8);

    world.click_text("Manage");
    let ribbon_parameters = rendered_label_in_rect("Parameters", [0.; 2], [2000., 150.]);
    world.click(ribbon_parameters);
    world.click(depth_equation());
    world.key(ViewportPhysicalKey::A, true);
    world.type_text("5 mm");
    world.click_text("Done");
    world.pump(Vec::new());
    assert!(
        world
            .state()
            .parameters
            .iter()
            .any(|row| row.name == "Depth" && row.expression == "5 mm")
    );
}

fn depth_equation() -> [i32; 2] {
    let (position, size) = diagnostics::window_geometry("Parameters").unwrap();
    let name = rendered_label_in_rect(
        "Depth",
        position,
        [position[0] + 200., position[1] + size[1]],
    );
    let equation_x = rendered_label_in_rect(
        "Equation",
        position,
        [position[0] + size[0], position[1] + size[1]],
    )[0];
    [equation_x, name[1]]
}

#[then("the extrusion updates to a depth of 10 mm")]
fn feature_tracks_driver(world: &mut ApplicationWorld) {
    assert_eq!(feature_count(world), 1);
    assert!((feature_distance(world) - 10.).abs() < 1e-8);
    assert_eq!(preview_depth(world), [0., 1.]);

    assert_owned_expression(world);
    open_extrusion_editor(world);
    world.pump(Vec::new());
    assert!(world.state().extrude_preview_active);
    assert_eq!(preview_depth(world), [0., 1.]);
    world.click_text("Cancel");
    world.pump(Vec::new());
    assert_eq!(preview_depth(world), [0., 1.]);
}

fn assert_owned_expression(world: &mut ApplicationWorld) {
    let document = world.active_id();
    let snapshot = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    let feature = snapshot
        .work_geometry
        .get(oblikovati_model::feature::FeatureId(1))
        .unwrap();
    let scalar = feature
        .owned_scalars()
        .iter()
        .find(|scalar| scalar.role == "distance")
        .unwrap();
    assert_eq!(
        snapshot.parameters.get(scalar.parameter).unwrap().source,
        "Depth * 2"
    );
}
