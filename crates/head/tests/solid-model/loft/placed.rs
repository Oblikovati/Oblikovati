// SPDX-License-Identifier: GPL-2.0-only
//! Place, annotate, and save a section through a checked native Loft edit.
use super::*;
use oblikovati_model::feature::part::PartFeatureDefinition;
use oblikovati_model::feature::sketch_loft::SketchLoftPlacedSection;

const DIMENSIONS: &str = "Section Dimensions";

#[given("an editable loft has two circular sections and a selected center line")]
fn open_editable_fixture_loft(world: &mut ApplicationWorld) {
    create_fixture_loft(world);
    reopen_with_guide(world);
}

#[given("an existing loft has a placed intermediate section")]
fn create_fixture_loft_with_placed_section(world: &mut ApplicationWorld) {
    create_fixture_loft(world);
    accept_midpoint_section(world);
}

#[given("the existing two-circle loft has a straight center line and a placed section")]
fn open_fixture_loft_with_placed_section(world: &mut ApplicationWorld) {
    create_fixture_loft_with_placed_section(world);
    reopen_with_the_saved_placed_section(world);
}

#[given("the user creates Loft1 from the two circular profiles and the 20 mm center line")]
fn create_fixture_loft(world: &mut ApplicationWorld) {
    fixtures::short_center_line_sections_if_needed(world);
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
    accept_section_dimensions(world);
    assert!(
        world
            .state()
            .loft_placed
            .is_some_and(|view| !view.editing && view.dirty)
    );
    accept_loft(world);
}

fn accept_section_dimensions(world: &mut ApplicationWorld) {
    let ok = diagnostics::pointer_pixels(world.run_in_window(DIMENSIONS, "OK").center());
    world.click(ok);
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry(DIMENSIONS).is_none());
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

#[given(
    "the user edits Loft1, selects placed-sections mode, adds an intermediate section near the center-line midpoint and accepts it with alternate section size 1"
)]
fn accept_midpoint_section(world: &mut ApplicationWorld) {
    reopen_with_guide(world);
    place_one_section(world);
    section_opens_dimensions(world);
    accept_section_and_loft(world);
    assert_eq!(committed_section(world).area_scale, 1.);
}

#[when("the user edits Loft1 and double-clicks the intermediate point in the preview")]
fn edit_the_section_at_its_viewport_point(world: &mut ApplicationWorld) {
    edit::double_click_loft(world);
    double_click_placed_point(world);
}

fn double_click_placed_point(world: &mut ApplicationWorld) {
    let view = world.state().loft_placed.unwrap();
    let z = view.section.position * view.guide_length_mm / 10.;
    let pixel = crate::HeadWindow::model_pixel(&world.app().window.native, [0., 0., z]).unwrap();
    world.pump(crate::native::gui_loop_tests::pointer_click_events(
        pixel, 2,
    ));
    world.pump(Vec::new());
}

#[then("Section Dimensions opens with the saved section settings")]
fn saved_section_dimensions_are_visible(world: &mut ApplicationWorld) {
    section_opens_dimensions(world);
    assert_eq!(
        world.state().loft_placed.unwrap().section,
        committed_section(world)
    );
    world.run_in_window(DIMENSIONS, "1.000");
}

#[when("the user changes the alternate section size from 1 to 2 and accepts it")]
fn double_the_section_area(world: &mut ApplicationWorld) {
    replace_section_dimension(world, "1.000", "2");
    accept_section_dimensions(world);
}

#[then("the displayed section area doubles and the loft preview bulges")]
fn doubled_area_changes_the_solid_preview(world: &mut ApplicationWorld) {
    let view = world.state().loft_placed.unwrap();
    let original = committed_section(world);
    assert_eq!(view.section.area_scale, 2.);
    assert!((view.section.position - original.position).abs() < 0.0001);
    let starting_radius = 10. - 5. * view.section.position;
    let before = std::f64::consts::PI * starting_radius.powi(2);
    assert!((view.area_mm2 / before - 2.).abs() < 1e-8);
    world.expect_text(&format!("Area {:.3}", view.area_mm2));
    let maximum_radius = world
        .app()
        .window
        .scene_triangles
        .iter()
        .filter(|triangle| triangle.color == crate::part_scene::NEW_SOLID_PREVIEW)
        .flat_map(|triangle| triangle.positions)
        .map(|point| point[0].hypot(point[1]))
        .fold(0.0_f32, f32::max);
    assert!(maximum_radius > 1.02, "preview radius {maximum_radius}");
}

#[when("the user accepts and reopens the loft and section dimensions")]
fn reopen_after_accepting_doubled_area(world: &mut ApplicationWorld) {
    accept_loft(world);
    edit_the_section_at_its_viewport_point(world);
}

#[then("the section size remains 2")]
fn saved_size_is_two(world: &mut ApplicationWorld) {
    section_opens_dimensions(world);
    assert_eq!(committed_section(world).area_scale, 2.);
    world.run_in_window(DIMENSIONS, "2.000");
}

#[when("the user sets the section position distance to 5 mm on the 20 mm center line")]
fn move_section_five_millimeters_from_the_start(world: &mut ApplicationWorld) {
    let view = world.state().loft_placed.unwrap();
    replace_section_dimension(
        world,
        &format!("{:.3}", view.section.position * view.guide_length_mm),
        "5",
    );
    accept_section_dimensions(world);
}

#[then("the section moves and its position annotation reads 0.250000")]
fn section_repositions_in_the_preview(world: &mut ApplicationWorld) {
    let view = world.state().loft_placed.unwrap();
    assert_eq!(view.section.position, 0.25);
    assert!((committed_section(world).position - 0.5).abs() < 0.03);
    world.expect_text("Position 0.250000");
    assert_eq!(world.state().loft_preview, Some(Ok(())));
}

fn replace_section_dimension(world: &mut ApplicationWorld, previous: &str, replacement: &str) {
    let position = diagnostics::pointer_pixels(world.run_in_window(DIMENSIONS, previous).center());
    world.click(position);
    world.key(oblikovati_viewport::ViewportPhysicalKey::A, true);
    world.type_text(replacement);
}

const EDIT_WARNING: &str = "Edit Loft Feature";

#[given("Loft1 is reopened for editing")]
fn reopen_with_the_saved_placed_section(world: &mut ApplicationWorld) {
    edit::double_click_loft(world);
    assert_eq!(
        world.state().loft_placed.unwrap().section,
        committed_section(world)
    );
}

#[when("the user enables Closed Loop and accepts the edit")]
fn try_closing_the_straight_guide(world: &mut ApplicationWorld) {
    let key = loft_body_key(world);
    let section = committed_section(world);
    let generation = world.state().document_generation;
    click_loft_control(world, "ClosedLoop");
    assert!(world.checked_run(world.run_in_window("Loft", "ClosedLoop")));
    accept_loft(world);
    assert_eq!(world.state().document_generation, generation);
    assert_eq!(loft_body_key(world), key);
    assert_eq!(committed_section(world), section);
}

#[then("the MCAD Software reports that the loft feature edit failed")]
fn report_the_invalid_closed_guide(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(EDIT_WARNING).is_some());
    world.expect_text("The Loft feature edit failed.");
    let details =
        diagnostics::pointer_pixels(world.run_in_window(EDIT_WARNING, "Details").center());
    world.click(details);
    world.pump(Vec::new());
    assert!(
        world
            .state()
            .loft_edit_warning
            .unwrap()
            .contains("periodic Center Line")
    );
    world.expect_text("periodic Center Line");
}

#[when("the user cancels the failure dialog")]
fn cancel_the_failed_closed_edit(world: &mut ApplicationWorld) {
    if diagnostics::window_geometry("Edit Thread Feature Failed").is_some() {
        let cancel = diagnostics::pointer_pixels(
            world
                .run_in_window("Edit Thread Feature Failed", "Cancel")
                .center(),
        );
        world.click(cancel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        return;
    }
    let key = loft_body_key(world);
    let generation = world.state().document_generation;
    let position =
        diagnostics::pointer_pixels(world.run_in_window(EDIT_WARNING, "Cancel").center());
    world.click(position);
    world.pump(Vec::new());
    assert_eq!(world.state().document_generation, generation);
    assert_eq!(loft_body_key(world), key);
}

#[then("the previously accepted solid is visible again")]
fn previous_solid_is_back_after_the_decline(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(EDIT_WARNING).is_none());
    assert!(diagnostics::window_geometry("Loft").is_none());
    assert!(world.state().browser.unwrap().bodies[0].visible);
    assert_eq!(committed_section(world).area_scale, 1.);
    assert!(!world.app().window.scene_triangles.is_empty());
    world.expect_no_text("Point1 Position");
    assert!(
        world
            .app()
            .window
            .scene_triangles
            .iter()
            .all(|triangle| { triangle.color != crate::part_scene::NEW_SOLID_PREVIEW })
    );
}
