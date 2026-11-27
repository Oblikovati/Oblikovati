// SPDX-License-Identifier: GPL-2.0-only
//! Create a tapered open shell through the published circular sections.
use super::*;

#[when("Loft is opened and surface output is selected")]
fn open_loft_for_surface(world: &mut ApplicationWorld) {
    open_loft_with_fitted_view(world);
    click_loft_control(world, "Surface");
}

#[when("both circular edges are selected as sections")]
fn pick_surface_sections(world: &mut ApplicationWorld) {
    pick_model_points(world, [[1., 0., 0.], [0.5, 0., 3.]]);
}

#[then("a tapered surface preview appears and OK is enabled")]
fn surface_loft_preview(world: &mut ApplicationWorld) {
    assert_eq!(world.state().loft_preview, Some(Ok(())));
    assert!(world.run_is_enabled("OK"));
    let mesh: Vec<_> = world
        .app()
        .window
        .scene_triangles
        .iter()
        .filter(|triangle| triangle.color == crate::part_scene::NEW_SOLID_PREVIEW)
        .collect();
    assert!(!mesh.is_empty());
    assert!(
        mesh.iter().all(|triangle| {
            let [a, b, c] = triangle.positions.map(|point| point[2]);
            a.max(b).max(c) - a.min(b).min(c) > 0.001
        }),
        "the preview shell must have no horizontal caps"
    );
    for (z, radius) in [(0., 1.), (3., 0.5)] {
        let preview_radius = mesh
            .iter()
            .flat_map(|triangle| triangle.positions)
            .filter(|point| (point[2] - z).abs() < 0.001)
            .map(|point| point[0].hypot(point[1]))
            .fold(0.0_f32, f32::max);
        assert!((preview_radius - radius).abs() < 0.005);
    }
    let capture = capture_framebuffer(&mut world.app().window.native);
    assert!(solid_visibility::visible_preview_fill(
        &capture,
        &diagnostics::window_occlusions()
    ));
    world.save_capture("surface-loft-preview");
}

#[then("LoftSrf1 and Surface Bodies(1) appear in the browser")]
fn surface_loft_is_committed_and_reopened(world: &mut ApplicationWorld) {
    let key = loft_surface_key(world);
    let path = world.scratch_folder().join("accepted-loft-surface.opd");
    solid_extrusion::acceptance::save_and_reopen(world, &path);
    world.pump(Vec::new());
    assert_eq!(loft_surface_key(world), key);
}

fn loft_surface_key(world: &mut ApplicationWorld) -> String {
    let browser = world.state().browser.unwrap();
    assert!(browser.bodies.is_empty());
    let [surface] = browser.surface_bodies.as_slice() else {
        panic!(
            "expected one surface body, got {:?}",
            browser.surface_bodies
        );
    };
    assert_eq!(surface.faces, 1);
    assert!(
        browser
            .features
            .iter()
            .any(|feature| feature.name == "LoftSrf1" && feature.id == surface.feature)
    );
    world.expect_text("LoftSrf1");
    world.expect_text("Surface Bodies (1)");
    surface.key.clone()
}

#[given(
    "the user creates LoftSrf1 with surface output by selecting both circular sketches as sections"
)]
fn create_loft_surface_for_edit(world: &mut ApplicationWorld) {
    open_loft_for_surface(world);
    pick_surface_sections(world);
    assert_eq!(world.state().loft_preview, Some(Ok(())));
    accept_loft(world);
    loft_surface_key(world);
}

#[given("LoftSrf1 is open on Transition with Automatic Mapping enabled")]
fn open_surface_transition(world: &mut ApplicationWorld) {
    let row = diagnostics::pointer_pixels(world.run("LoftSrf1").center());
    world.pump(crate::native::gui_loop_tests::pointer_click_events(row, 2));
    world.pump(Vec::new());
    assert_eq!(world.state().loft_preview, Some(Ok(())));
    click_loft_control(world, "Transition");
    assert!(world.checked_run(world.run_in_window("Loft", "AutomaticMapping")));
}

#[then("sets 1 and map-point rows for Sketch1 and Sketch2 are displayed")]
fn default_mapping_shows_both_point_rows(world: &mut ApplicationWorld) {
    assert!(!world.checked_run(world.run_in_window("Loft", "AutomaticMapping")));
    assert!(loft_panel_has_text(
        world,
        "Set1Sketch1MapPointSketch2MapPoint"
    ));
}

#[when("Click to add is selected")]
fn add_an_incomplete_point_set(world: &mut ApplicationWorld) {
    click_loft_control(world, "Clicktoadd");
}

#[then("sets 2 appears with starred map-point rows and OK is disabled")]
fn new_point_rows_are_incomplete(world: &mut ApplicationWorld) {
    assert!(loft_panel_has_text(
        world,
        "Set2Sketch1MapPoint*Sketch2MapPoint*"
    ));
    assert!(!world.run_is_enabled("OK"));
    assert_eq!(world.state().loft_preview, Some(Ok(())));
}

#[when("the incomplete edit is cancelled")]
fn cancel_incomplete_mapping(world: &mut ApplicationWorld) {
    let before = world.state().document_generation;
    let key = loft_surface_key(world);
    click_loft_control(world, "Cancel");
    assert_eq!(world.state().document_generation, before);
    assert_eq!(loft_surface_key(world), key);
}

#[then("the existing surface Loft remains visible")]
fn surface_loft_stays_visible(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Loft").is_none());
    assert!(world.state().browser.unwrap().surface_bodies[0].visible);
    assert!(!world.app().window.scene_triangles.is_empty());
    assert!(
        world
            .app()
            .window
            .scene_triangles
            .iter()
            .all(|triangle| { triangle.color != crate::part_scene::NEW_SOLID_PREVIEW })
    );
}
