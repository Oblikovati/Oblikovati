// SPDX-License-Identifier: GPL-2.0-only
//! Select and replay the fixture's actual straight Center Line.
use super::*;
use oblikovati_model::feature::{part::PartFeatureDefinition, sketch_loft::SketchLoftCenterLine};

#[given("the two circular sections and straight sketch between their centers are visible")]
fn center_line_inputs_are_visible(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    let names: Vec<_> = browser
        .sketches
        .iter()
        .filter(|sketch| sketch.visible)
        .map(|sketch| sketch.name.as_str())
        .collect();
    assert_eq!(names, ["Sketch1", "Sketch2", "Sketch3"]);
}

#[when("both circular profiles are selected in Loft and Center Line mode is chosen")]
fn draft_a_center_line_loft(world: &mut ApplicationWorld) {
    draft_on_guide(world, 3.);
}

pub(super) fn draft_on_guide(world: &mut ApplicationWorld, height: f64) {
    open_loft_with_fitted_view(world);
    pick_model_points(world, [[1., 0., 0.], [0.5, 0., height]]);
    click_loft_control(world, "CenterLine");
}

#[then("the dialog requests a center-line sketch and OK is disabled")]
fn asks_for_the_center_line(world: &mut ApplicationWorld) {
    assert!(loft_panel_has_text(world, "SelectaCenterLinesketch"));
    assert!(world.state().loft_center_line.is_none());
    assert!(!world.run_is_enabled("OK"));
}

#[when("the Center Line selector is activated and Sketch3 is shown as the selected center line")]
fn pick_the_straight_center_line(world: &mut ApplicationWorld) {
    pick_guide(world, 1.5);
}

pub(super) fn pick_guide(world: &mut ApplicationWorld, z: f64) {
    click_loft_control(world, "SelectCenterLine");
    let pixel = crate::HeadWindow::model_pixel(&world.app().window.native, [0., 0., z]).unwrap();
    world.click(pixel);
    world.pump(Vec::new());
    assert_eq!(world.state().loft_center_line.as_deref(), Some("Sketch3"));
    world.run_in_window("Loft", "Sketch3");
}

#[then("a solid preview appears and OK is enabled")]
fn guided_solid_preview_is_ready(world: &mut ApplicationWorld) {
    assert_eq!(world.state().loft_preview, Some(Ok(())));
    assert!(world.run_is_enabled("OK"));
    let capture = capture_framebuffer(&mut world.app().window.native);
    assert!(solid_visibility::visible_preview_fill(
        &capture,
        &diagnostics::window_occlusions()
    ));
}

#[then("Loft1 and Solid Bodies(1) appear in the browser")]
fn guided_loft_replays_its_authored_line(world: &mut ApplicationWorld) {
    let old_key = loft_body_key(world);
    world.expect_text("Solid Bodies (1)");
    assert_center_line_recipe(world);
    let path = world.scratch_folder().join("center-line-loft.opd");
    solid_extrusion::acceptance::save_and_reopen(world, &path);
    world.pump(Vec::new());
    assert_eq!(loft_body_key(world), old_key);
    assert_center_line_recipe(world);
}

fn assert_center_line_recipe(world: &mut ApplicationWorld) {
    let document = world.active_id();
    let feature = world.state().browser.unwrap().bodies[0].feature;
    let app = world.app();
    let snapshot = app
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    let authored = snapshot.work_geometry.get(feature).unwrap();
    let PartFeatureDefinition::SketchLoft(definition) = authored.definition() else {
        panic!("expected the authored Loft");
    };
    assert_eq!(
        definition.recipe().center_line,
        Some(SketchLoftCenterLine {
            sketch: DocumentSketchId(3),
            line: SketchEntityId(10)
        })
    );
}
