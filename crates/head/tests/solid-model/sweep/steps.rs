// SPDX-License-Identifier: GPL-2.0-only
//! Exercise the published Sweep geometry and history through the desktop.
use super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiSweepPreview;
use oblikovati_viewport::{ViewportNavigation, ViewportPhysicalKey};

#[given(expr = "a disposable copy of the native fixture {string} is open")]
fn open_sweep_fixture(world: &mut ApplicationWorld, relative: String) {
    let source = scenario_root().join(&relative);
    let copy = world.scratch_folder().join(source.file_name().unwrap());
    std::fs::copy(source, &copy).unwrap();
    world.open_fixture(&copy);
}

#[given("the part contains no sketches or solid bodies")]
fn empty_part(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert!(browser.sketches.is_empty());
    assert!(browser.bodies.is_empty());
}

#[given("its circle profile and straight path sketches are visible")]
fn source_sketches_visible(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.sketches.len(), 2);
    assert!(browser.sketches.iter().all(|sketch| sketch.visible));
    assert!(browser.bodies.is_empty());
}

#[when("Sweep is clicked")]
#[when("Sweep is invoked")]
fn invoke_sweep(world: &mut ApplicationWorld) {
    world.click_text("Sweep");
    world.pump(Vec::new());
}

#[then(
    "the MCAD Software displays a tip describing the geometry combinations required to create a sweep"
)]
fn geometry_tip(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().sweep_preview.map(|preview| preview.result),
        Some(GuiSweepPreview::MissingInputs)
    );
    assert!(super::world::squeezed(&world.window_text("Sweep")).contains("closedprofilesketch"));
}

#[then("OK and Help are available")]
fn help_and_ok(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("OK"));
    assert!(world.run_is_enabled("Help"));
    click_panel(world, "Help");
    assert!(super::world::squeezed(&world.window_text("Sweep")).contains("Startthepath"));
}

#[then("the panel shows 1 Profile and 1 Curve")]
fn chosen_sources(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().sweep_preview.map(|preview| preview.result),
        Some(GuiSweepPreview::Ready)
    );
    let panel = super::world::squeezed(&world.window_text("Sweep"));
    assert!(panel.contains("1Profile"), "{panel}");
    assert!(panel.contains("1Curve"), "{panel}");
}

#[then("a solid preview and enabled OK are displayed")]
fn shaded_preview(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("OK"));
    assert!(!world.app().window.scene_triangles.is_empty());
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.25 },
    )
    .unwrap();
    world.pump(Vec::new());
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    assert!(
        super::solid_visibility::visible_preview_fill(
            &capture,
            &super::diagnostics::window_occlusions()
        ),
        "expected a visible shaded Sweep preview"
    );
    world.save_capture("sweep-preview");
}

#[when("OK is clicked")]
fn accept(world: &mut ApplicationWorld) {
    click_panel(world, "OK");
}

#[then("Sweep1 and Solid Bodies(1) appear")]
#[then("Sweep1 and its solid return")]
fn accepted_sweep(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 1);
    let feature = browser
        .features
        .iter()
        .find(|feature| feature.kind == "Sweep")
        .unwrap();
    assert_eq!(feature.name, "Sweep1");
    assert_eq!(browser.bodies[0].feature, feature.id);
    assert_eq!(browser.bodies[0].faces, 3);
    assert!(browser.sketches.iter().all(|sketch| !sketch.visible));
    world.expect_text("Sweep1");
    world.expect_text("Solid Bodies");
}

#[when("Undo is invoked")]
fn undo(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Z, true);
    world.pump(Vec::new());
}

#[then("the two sketches return without the solid")]
fn sweep_was_undone(world: &mut ApplicationWorld) {
    source_sketches_visible(world);
    assert!(
        !world
            .state()
            .browser
            .unwrap()
            .features
            .iter()
            .any(|feature| feature.kind == "Sweep")
    );
}

#[when("Redo is invoked")]
fn redo(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Y, true);
    world.pump(Vec::new());
}

fn click_panel(world: &mut ApplicationWorld, label: &str) {
    let position = super::diagnostics::pointer_pixels(world.run_in_window("Sweep", label).center());
    world.click(position);
    world.pump(Vec::new());
}

#[given(
    "the user creates Sweep1 from its circle profile and straight path with Follow Path orientation and 5 degrees taper"
)]
fn create_tapered_sweep(world: &mut ApplicationWorld) {
    invoke_sweep(world);
    replace_taper(world, "5 deg");
    let draft = world.state().sweep_preview.unwrap();
    assert_eq!(draft.result, GuiSweepPreview::Ready);
    assert_eq!(draft.request.taper, "5 deg");
    assert!(world.window_text("Sweep").contains("Follow"));
    assert!((preview_width(world) - 1.525).abs() < 0.03);
    accept(world);
    accepted_sweep(world);
}

#[when("Sweep1 is reopened for editing")]
fn reopen_sweep(world: &mut ApplicationWorld) {
    let row = world.run("Sweep1").center();
    world.right_click(super::diagnostics::pointer_pixels(row));
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.expect_text("Show Dimensions");
    world.click(super::solid_extrusion::acceptance::edit_menu_position(
        world,
    ));
    world.pump(Vec::new());
}

#[then("1 Profile 1 Curve and taper 5 deg are retained")]
fn taper_is_retained(world: &mut ApplicationWorld) {
    chosen_sources(world);
    let panel = super::world::squeezed(&world.window_text("Sweep"));
    assert!(panel.contains("Taper5deg"), "{panel}");
    let draft = world.state().sweep_preview.unwrap().request;
    assert!(draft.feature.is_some());
    assert_eq!(draft.taper, "5 deg");
}

#[when("taper is replaced with 0")]
fn clear_taper(world: &mut ApplicationWorld) {
    replace_taper(world, "0");
}

#[then("the preview changes to straight sides and OK is enabled")]
fn preview_is_straight(world: &mut ApplicationWorld) {
    let draft = world.state().sweep_preview.unwrap();
    assert_eq!(draft.result, GuiSweepPreview::Ready);
    assert_eq!(draft.request.taper, "0");
    assert!((preview_width(world) - 1.).abs() < 0.03);
    assert!(world.run_is_enabled("OK"));
}

#[then("Sweep1 remains with one solid body")]
fn straight_sweep_stays_in_the_browser(world: &mut ApplicationWorld) {
    accepted_sweep(world);
    let browser = world.state().browser.unwrap();
    let feature = browser.bodies[0].feature;
    assert_eq!(
        browser
            .sweep_edits
            .get(&feature)
            .map(|seed| seed.taper.as_str()),
        Some("0 deg")
    );
}

fn replace_taper(world: &mut ApplicationWorld, expression: &str) {
    let label = world.run_in_window("Sweep", "Taper").clone();
    let (origin, size) = super::diagnostics::window_geometry("Sweep").unwrap();
    let point = [origin[0] + size[0] * 0.73, label.center()[1]];
    world.click(super::diagnostics::pointer_pixels(point));
    world.key(ViewportPhysicalKey::A, true);
    world.type_text(expression);
    world.pump(Vec::new());
}

fn preview_width(world: &mut ApplicationWorld) -> f32 {
    let (left, right) = world
        .app()
        .window
        .scene_triangles
        .iter()
        .filter(|triangle| triangle.color == crate::part_scene::NEW_SOLID_PREVIEW)
        .flat_map(|triangle| triangle.positions.map(|point| point[0]))
        .fold((f32::INFINITY, f32::NEG_INFINITY), |(left, right), x| {
            (left.min(x), right.max(x))
        });
    right - left
}

#[given("Sweep1 is open for editing with its profile and path selected")]
fn sweep_is_open_for_editing(world: &mut ApplicationWorld) {
    reopen_sweep(world);
    chosen_sources(world);
}

#[when("the path is cleared while editing")]
fn clear_sweep_path(world: &mut ApplicationWorld) {
    click_panel(world, "ClearPath");
}

#[then("the path selector requests a curve or edge")]
fn sweep_requests_a_path(world: &mut ApplicationWorld) {
    let draft = world.state().sweep_preview.unwrap();
    assert_eq!(draft.result, GuiSweepPreview::MissingPath);
    assert!(draft.request.path_cleared);
    let panel = super::world::squeezed(&world.window_text("Sweep"));
    assert!(panel.contains("SelectCurveorEdge"), "{panel}");
    assert!(panel.contains("1Profile"), "{panel}");
    assert!(!panel.contains("1Curve"), "{panel}");
}

#[then("the solid preview disappears and OK is disabled")]
fn missing_path_removes_the_sweep_preview(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled("OK"));
    assert!(world.app().window.scene_triangles.is_empty());
    assert_eq!(world.state().browser.unwrap().bodies.len(), 1);
    world.save_capture("sweep-path-cleared");
}

#[then("the existing Sweep1 solid returns")]
fn original_sweep_returns(world: &mut ApplicationWorld) {
    accepted_sweep(world);
    assert!(world.state().sweep_preview.is_none());
    assert!(!world.app().window.scene_triangles.is_empty());
}

#[then("reopening Sweep1 shows its profile and path retained")]
fn original_sweep_inputs_return(world: &mut ApplicationWorld) {
    reopen_sweep(world);
    taper_is_retained(world);
    assert!(!world.state().sweep_preview.unwrap().request.path_cleared);
}

#[when("orientation selector style is changed to Dropdown")]
fn dropdown_style(world: &mut ApplicationWorld) {
    click_panel(world, "Orientation");
    click_orientation_option(world, "Dropdown");
    world.pump(Vec::new());
    click_panel(world, "FollowPath");
}

#[then("Follow Path Fixed and Guide are listed")]
fn dropdown_lists_orientations(world: &mut ApplicationWorld) {
    for label in ["FollowPath", "Fixed", "Guide"] {
        assert!(
            world
                .frame
                .runs
                .iter()
                .any(|run| run.text == label && world.frame.is_popup_row(run))
        );
    }
}

#[given("the user selects Guide orientation without a guide reference")]
#[when("Guide is selected without a guide reference")]
fn select_guide(world: &mut ApplicationWorld) {
    click_orientation_option(world, "Guide");
}

#[then("selects Rail or Surfaces is required")]
fn guide_requires_geometry(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().sweep_preview.map(|state| state.result),
        Some(GuiSweepPreview::MissingGuide)
    );
    assert!(super::world::squeezed(&world.window_text("Sweep")).contains("SelectRailorSurfaces"));
}

#[when("Fixed is selected")]
fn select_fixed(world: &mut ApplicationWorld) {
    click_panel(world, "Fixed");
}

#[then("a solid preview returns and OK is enabled")]
fn fixed_preview_returns(world: &mut ApplicationWorld) {
    let draft = world.state().sweep_preview.unwrap();
    assert_eq!(draft.result, GuiSweepPreview::Ready);
    assert_eq!(
        draft.request.orientation,
        oblikovati_gui::GuiSweepOrientation::Fixed
    );
    assert!((preview_width(world) - 1.).abs() < 0.03);
    assert!(world.run_is_enabled("OK"));
}

#[then("the Taper and Twist fields are absent")]
fn fixed_hides_angle_fields(world: &mut ApplicationWorld) {
    let panel = world.window_text("Sweep");
    assert!(
        !panel.contains("Taper") && !panel.contains("Twist"),
        "{panel}"
    );
}

#[when("Advanced Properties is expanded")]
fn show_advanced(world: &mut ApplicationWorld) {
    click_panel(world, "AdvancedProperties");
}

#[then("Optimize for Single Selection is shown unchecked")]
fn optimization_defaults_unchecked(world: &mut ApplicationWorld) {
    world.run_in_window("Sweep", "OptimizeforSingleSelection");
    assert!(!world.checked("Optimize for Single Selection"));
}

#[then("Sweep1 and one solid body remain")]
fn accepted_orientation_is_fixed(world: &mut ApplicationWorld) {
    accepted_sweep(world);
    let browser = world.state().browser.unwrap();
    let feature = browser.bodies[0].feature;
    assert_eq!(
        browser.sweep_edits[&feature].orientation,
        oblikovati_gui::GuiSweepOrientation::Fixed
    );
}

fn click_orientation_option(world: &mut ApplicationWorld, label: &str) {
    let popup = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == label && world.frame.is_popup_row(run))
        .map(super::diagnostics::RenderedRun::center);
    if let Some(position) = popup {
        world.click(super::diagnostics::pointer_pixels(position));
        world.pump(Vec::new());
    } else {
        click_panel(world, label);
    }
}
