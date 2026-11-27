// SPDX-License-Identifier: GPL-2.0-only
//! Place, annotate, and save a section through a checked native Loft edit.
use super::*;
use oblikovati_model::feature::part::PartFeatureDefinition;
use oblikovati_model::feature::sketch_loft::SketchLoftPlacedSection;

const DIMENSIONS: &str = "Section Dimensions";

#[given("the user creates Loft1 from the two circular profiles and the 20 mm center line")]
fn create_fixture_loft(world: &mut ApplicationWorld) {
    center_line::draft_on_guide(world, 2.);
    center_line::pick_guide(world, 1.);
    accept_loft(world);
    loft_body_key(world);
}

#[given("Loft1 is reopened for editing with its two sections and center line selected")]
fn reopen_with_guide(world: &mut ApplicationWorld) {
    edit::double_click_loft(world);
    let state = world.state();
    assert_eq!(state.loft_sections.len(), 2);
    assert_eq!(state.loft_center_line.as_deref(), Some("Sketch3"));
}

#[when("the user selects placed-sections mode and clicks the center line near its midpoint")]
fn place_one_section(world: &mut ApplicationWorld) {
    click_loft_control(world, "CenterLinewithPlacedSections");
    let midpoint =
        crate::HeadWindow::model_pixel(&world.app().window.native, [0., 0., 1.]).unwrap();
    world.click(midpoint);
    world.pump(Vec::new());
}

#[then("a Point1 entry and the Section Dimensions dialog appear")]
fn section_opens_dimensions(world: &mut ApplicationWorld) {
    assert!(world.state().loft_placed.unwrap().editing);
    world.run_in_window("Loft", "Point1");
    assert!(diagnostics::window_geometry(DIMENSIONS).is_some());
    world.run_in_window(DIMENSIONS, "AlternateSectionSize");
}

#[then("the preview shows an intermediate section with position and area annotations")]
fn preview_has_geometric_section_and_annotations(world: &mut ApplicationWorld) {
    let state = world.state();
    assert_eq!(state.loft_preview, Some(Ok(())));
    let view = state.loft_placed.unwrap();
    assert!((view.section.position - 0.5).abs() < 0.03);
    assert!((view.guide_length_mm - 20.).abs() < 1e-8);
    let radius_mm = 10. - 5. * view.section.position;
    assert!((view.area_mm2 - std::f64::consts::PI * radius_mm.powi(2)).abs() < 1e-8);
    world.expect_text(&format!(
        "Position {:.6} Area {:.3}",
        view.section.position, view.area_mm2
    ));
    assert!(
        world
            .app()
            .window
            .scene_triangles
            .iter()
            .any(|triangle| triangle.color == crate::part_scene::NEW_SOLID_PREVIEW)
    );
}

#[when("the user accepts the section dimensions and the Loft dialog")]
fn accept_section_and_loft(world: &mut ApplicationWorld) {
    let ok = diagnostics::pointer_pixels(world.run_in_window(DIMENSIONS, "OK").center());
    world.click(ok);
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry(DIMENSIONS).is_none());
    assert!(
        world
            .state()
            .loft_placed
            .is_some_and(|view| !view.editing && view.dirty)
    );
    accept_loft(world);
}

#[then("Loft1 remains in the browser and a solid is visible")]
fn committed_section_survives_reopen(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Loft").is_none());
    let key = loft_body_key(world);
    let section = committed_section(world);
    assert!((section.position - 0.5).abs() < 0.03);
    assert_eq!(section.area_scale, 1.);
    let path = world.scratch_folder().join("loft-with-section.opd");
    solid_extrusion::acceptance::save_and_reopen(world, &path);
    world.pump(Vec::new());
    assert_eq!(loft_body_key(world), key);
    assert!(world.state().browser.unwrap().bodies[0].visible);
    assert_eq!(committed_section(world), section);
}

fn committed_section(world: &mut ApplicationWorld) -> SketchLoftPlacedSection {
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
    let PartFeatureDefinition::SketchLoft(definition) =
        snapshot.work_geometry.get(feature).unwrap().definition()
    else {
        panic!("expected Loft1 to have a saved Loft definition");
    };
    definition.recipe().placed.unwrap()
}
