// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_gui::GuiSweepOutput;
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchEntityId};

#[given("an existing solid has a circular profile on its top face")]
fn block_has_a_circular_top_profile(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 1);
    assert_eq!(browser.bodies[0].faces, 6);
    let profile = DocumentSketchId(2);
    assert!(
        browser
            .sketches
            .iter()
            .any(|sketch| sketch.id == profile && sketch.visible)
    );
    let workspace = world.app().owner.documents().workspace();
    let snapshot = workspace
        .session(workspace.active_document().unwrap())
        .unwrap()
        .snapshot();
    assert_eq!(
        snapshot
            .resolve_sketch_plane(profile)
            .unwrap()
            .to_model([0., 0.])
            .unwrap()[2],
        2.
    );
    assert_eq!(
        snapshot
            .sketches
            .get(profile)
            .unwrap()
            .geometry()
            .circle(SketchEntityId(36))
            .unwrap()
            .radius,
        0.5
    );
}

#[given("a straight path extending upward from the profile")]
fn straight_path_extends_upward(world: &mut ApplicationWorld) {
    let workspace = world.app().owner.documents().workspace();
    let snapshot = workspace
        .session(workspace.active_document().unwrap())
        .unwrap()
        .snapshot();
    let path = DocumentSketchId(3);
    let plane = snapshot.resolve_sketch_plane(path).unwrap();
    let geometry = snapshot.sketches.get(path).unwrap().geometry();
    let line = geometry.line(SketchEntityId(40)).unwrap();
    let ends = line.endpoints.map(|id| {
        plane
            .to_model(geometry.point(id).unwrap().position)
            .unwrap()
    });
    assert_eq!(ends, [[-2., -1.25, 2.], [-2., -1.25, 5.]]);
}

#[when("Sweep is invoked and New Solid is selected")]
fn create_separate_solid(world: &mut ApplicationWorld) {
    invoke_sweep(world);
    click_panel(world, "NewSolid");
}

#[then("Body Name shows Solid2 and OK is enabled")]
fn new_solid_name_is_shown(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text("Sweep"));
    assert!(panel.contains("BodyNameSolid2"), "{panel}");
    assert!(world.run_is_enabled("OK"));
    assert_eq!(
        world.state().sweep_preview.unwrap().request.output,
        GuiSweepOutput::NewSolid
    );
}

#[then("Sweep1 and Solid Bodies(2) appear")]
fn two_solid_bodies_are_shown(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 2);
    assert!(
        browser
            .bodies
            .iter()
            .any(|body| body.name == "Solid2" && body.faces == 3)
    );
    world.expect_text("Sweep1");
    world.expect_text("Solid Bodies");
}

#[when("Sweep1 is edited to Join and accepted")]
fn edit_sweep_to_join(world: &mut ApplicationWorld) {
    let original = world
        .state()
        .browser
        .unwrap()
        .bodies
        .into_iter()
        .find(|body| body.name == "Solid1")
        .unwrap();
    reopen_sweep(world);
    click_panel(world, "Join");
    let draft = world.state().sweep_preview.unwrap();
    assert_eq!(draft.result, GuiSweepPreview::Ready);
    assert_eq!(draft.request.output, GuiSweepOutput::Join);
    assert!(world.run_is_enabled("OK"));
    accept(world);
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 1);
    assert_eq!(browser.bodies[0].key, original.key);
    assert_eq!(
        browser.sweep_edits[&browser.bodies[0].feature].output,
        GuiSweepOutput::Join
    );
}

#[then("the raised boss remains and Solid Bodies(1) is shown")]
fn joined_part_has_one_body_with_the_boss(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 1);
    assert_eq!(browser.bodies[0].faces, 8);
    let top = world
        .app()
        .window
        .scene_triangles
        .iter()
        .flat_map(|triangle| triangle.positions.map(|point| point[2]))
        .fold(f32::NEG_INFINITY, f32::max);
    assert!(
        (top - 5.).abs() < 0.01,
        "expected the raised Sweep cap at 5 cm, got {top}"
    );
    world.expect_text("Sweep1");
    world.expect_text("Solid Bodies");
}

#[given(
    "the user creates Sweep1 by joining the circular profile along the outward straight path at zero taper"
)]
fn create_joined_boss(world: &mut ApplicationWorld) {
    block_has_a_circular_top_profile(world);
    straight_path_extends_upward(world);
    invoke_sweep(world);
    click_panel(world, "Join");
    let preview = world.state().sweep_preview.unwrap();
    assert_eq!(preview.request.output, GuiSweepOutput::Join);
    assert_eq!(preview.result, GuiSweepPreview::Ready);
    accept(world);
    joined_part_has_one_body_with_the_boss(world);
}

#[when("Sweep1 is edited to Cut and OK is clicked")]
fn try_to_cut_outside_the_part(world: &mut ApplicationWorld) {
    reopen_sweep(world);
    click_panel(world, "Cut");
    let preview = world.state().sweep_preview.unwrap();
    assert_eq!(preview.request.output, GuiSweepOutput::Cut);
    assert_eq!(
        preview.result,
        GuiSweepPreview::UnchangedCut("The feature did not change the number of faces (6).".into())
    );
    assert!(world.run_is_enabled("OK"));
    accept(world);
}

#[then("an Edit Sweep Feature warning is displayed")]
fn unchanged_cut_warning_is_displayed(world: &mut ApplicationWorld) {
    world.expect_text("Edit Sweep Feature");
    world.expect_text("Details");
}

#[then("its expanded detail reports that the feature did not change the number of faces")]
fn expand_unchanged_cut_detail(world: &mut ApplicationWorld) {
    click_warning(world, "Details");
    let warning = super::super::world::squeezed(&world.window_text("Edit Sweep Feature"));
    assert!(
        warning.contains("didnotchangethenumberoffaces(6)"),
        "{warning}"
    );
}

#[when("Cancel is clicked in the warning")]
fn cancel_unchanged_cut(world: &mut ApplicationWorld) {
    let generation = world.state().sweep_preview.unwrap().request.generation;
    click_warning(world, "Cancel");
    assert!(world.state().sweep_preview.is_none());
    let workspace = world.app().owner.documents().workspace();
    let session = workspace
        .session(workspace.active_document().unwrap())
        .unwrap();
    assert_eq!(session.generation(), generation);
}

fn click_warning(world: &mut ApplicationWorld, label: &str) {
    let row = world.run_in_window("Edit Sweep Feature", label);
    world.click(super::diagnostics::pointer_pixels(row.center()));
    world.pump(Vec::new());
}

#[then("the joined boss and one solid body return")]
fn the_previous_join_is_retained(world: &mut ApplicationWorld) {
    joined_part_has_one_body_with_the_boss(world);
    let browser = world.state().browser.unwrap();
    assert_eq!(
        browser.sweep_edits[&browser.bodies[0].feature].output,
        GuiSweepOutput::Join
    );
}

#[given("Sweep1 is open for editing with its path and zero taper")]
fn open_join_with_zero_taper(world: &mut ApplicationWorld) {
    reopen_sweep(world);
    chosen_sources(world);
    let draft = world.state().sweep_preview.unwrap();
    assert_eq!(draft.request.output, GuiSweepOutput::Join);
    assert_eq!(draft.request.taper, "0 deg");
}

#[when(expr = "the taper is changed to {int} degrees and the preview updates")]
fn change_join_taper(world: &mut ApplicationWorld, degrees: i32) {
    let expression = format!("{degrees} deg");
    replace_taper(world, &expression);
    let draft = world.state().sweep_preview.unwrap();
    assert_eq!(draft.request.taper, expression);
    assert_eq!(draft.result, GuiSweepPreview::Ready);
}

#[then("the preview narrows toward its upper end")]
fn joined_boss_narrows(world: &mut ApplicationWorld) {
    let width = preview_cap_width(world);
    let expected = 1. - 6. * 5_f32.to_radians().tan();
    assert!(
        (width - expected).abs() < 0.03,
        "cap width {width}, expected {expected}"
    );
}

#[then("the preview widens toward its upper end")]
fn joined_boss_widens(world: &mut ApplicationWorld) {
    let width = preview_cap_width(world);
    let expected = 1. + 6. * 5_f32.to_radians().tan();
    assert!(
        (width - expected).abs() < 0.03,
        "cap width {width}, expected {expected}"
    );
}

fn preview_cap_width(world: &mut ApplicationWorld) -> f32 {
    let (left, right) = world
        .app()
        .window
        .scene_triangles
        .iter()
        .filter(|triangle| triangle.color == crate::part_scene::NEW_SOLID_PREVIEW)
        .flat_map(|triangle| triangle.positions)
        .filter(|point| (point[2] - 5.).abs() < 0.001)
        .fold(
            (f32::INFINITY, f32::NEG_INFINITY),
            |(left, right), point| (left.min(point[0]), right.max(point[0])),
        );
    right - left
}

#[then("the edit closes")]
fn join_taper_is_unchanged_after_cancel(world: &mut ApplicationWorld) {
    assert!(super::diagnostics::window_geometry("Sweep").is_none());
    assert!(world.state().sweep_preview.is_none());
    joined_part_has_one_body_with_the_boss(world);
    let browser = world.state().browser.unwrap();
    let seed = &browser.sweep_edits[&browser.bodies[0].feature];
    assert_eq!(seed.output, GuiSweepOutput::Join);
    assert_eq!(seed.taper, "0 deg");
}

#[given("a circular sketch curve and a straight path are available")]
fn sweep_geometry_is_available(world: &mut ApplicationWorld) {
    block_has_a_circular_top_profile(world);
    straight_path_extends_upward(world);
}

#[given("the user opens Sweep and selects the circular profile and straight path")]
fn select_surface_sources_in_the_viewport(world: &mut ApplicationWorld) {
    invoke_sweep(world);
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.25 },
    )
    .unwrap();
    click_panel(world, "ClearProfile");
    assert_eq!(
        world.state().sweep_preview.unwrap().result,
        GuiSweepPreview::MissingProfile
    );
    click_sweep_point(world, [-2., -1.25, 2.]);
    let draft = world.state().sweep_preview.unwrap();
    let oblikovati_gui::GuiSweepProfile::Picked(profile) = draft.request.profile else {
        panic!("expected the closed circle to be selected in the viewport");
    };
    assert_eq!(profile.outer[0].curve, SketchEntityId(36));

    click_panel(world, "ClearPath");
    assert_eq!(
        world.state().sweep_preview.unwrap().result,
        GuiSweepPreview::MissingPath
    );
    click_sweep_point(world, [-2., -1.25, 3.5]);
    chosen_sources(world);
    assert!(!world.state().sweep_preview.unwrap().request.path_cleared);
}

fn click_sweep_point(world: &mut ApplicationWorld, point: [f64; 3]) {
    let pixel = crate::HeadWindow::model_pixel(&world.app().window.native, point).unwrap();
    world.click(pixel);
    world.pump(Vec::new());
}

#[when("the Sweep is switched to surface output")]
fn select_surface_output(world: &mut ApplicationWorld) {
    click_panel(world, "Surface");
}

#[then("Profiles shows 1 Curve and Body Name shows Srf1")]
fn surface_inputs_and_name(world: &mut ApplicationWorld) {
    let draft = world.state().sweep_preview.unwrap();
    assert_eq!(draft.request.output, GuiSweepOutput::Surface);
    assert_eq!(draft.result, GuiSweepPreview::Ready);
    let panel = super::super::world::squeezed(&world.window_text("Sweep"));
    assert!(panel.contains("Profiles1Curve"), "{panel}");
    assert!(panel.contains("BodyNameSrf1"), "{panel}");
    assert!(world.run_is_enabled("OK"));
}

#[when("the Sweep is accepted")]
fn accept_surface_sweep(world: &mut ApplicationWorld) {
    accept(world);
}

#[then("SweepSrf1 and Surface Bodies(1) appear")]
fn named_surface_is_in_the_browser(world: &mut ApplicationWorld) {
    world.expect_text("SweepSrf1");
    world.expect_text("Surface Bodies (1)");
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.surface_bodies.len(), 1);
    assert_eq!(browser.surface_bodies[0].faces, 1);
}

#[then("the Surface Bodies folder contains Srf1")]
fn the_surface_body_has_its_own_name(world: &mut ApplicationWorld) {
    world.expect_text("Srf1");
    assert_eq!(
        world.state().browser.unwrap().surface_bodies[0].name,
        "Srf1"
    );
}

#[then("Solid Bodies(1) remains")]
fn the_original_solid_is_retained(world: &mut ApplicationWorld) {
    world.expect_text("Solid Bodies");
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 1);
    assert_eq!(browser.bodies[0].faces, 6);
}

#[given("a surface Sweep is started with Guide orientation and no path or guide selected")]
fn start_incomplete_guided_sweep(world: &mut ApplicationWorld) {
    invoke_sweep(world);
    select_surface_output(world);
    select_guide(world);
    click_panel(world, "ClearPath");
    click_panel(world, "ClearProfile");
    let draft = world.state().sweep_preview.unwrap();
    assert_eq!(
        draft.request.orientation,
        oblikovati_gui::GuiSweepOrientation::Guide
    );
    assert!(draft.request.path_cleared);
    assert_eq!(draft.result, GuiSweepPreview::MissingProfile);
}

#[when("the circular profile curve is selected")]
fn pick_the_guided_sweep_profile(world: &mut ApplicationWorld) {
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.25 },
    )
    .unwrap();
    click_sweep_point(world, [-2., -1.25, 2.]);
}

#[then("Profiles shows 1 Curve")]
fn guided_surface_profile_is_retained(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text("Sweep"));
    assert!(text.contains("Profiles1Curve"), "{text}");
    assert!(matches!(
        world.state().sweep_preview.unwrap().request.profile,
        oblikovati_gui::GuiSweepProfile::Picked(_)
    ));
}

#[then("Path and Guide still request geometry")]
fn both_guided_inputs_remain_empty(world: &mut ApplicationWorld) {
    path_needs_a_viewport_hit(world);
    world.run_in_window("Sweep", "Guide");
    let text = super::super::world::squeezed(&world.window_text("Sweep"));
    assert!(text.contains("SelectRailorSurfaces"), "{text}");
}

#[when("the path sketch is clicked in the browser")]
fn click_path_sketch_row(world: &mut ApplicationWorld) {
    world.click_text("Sketch3");
    world.pump(Vec::new());
}

#[then("Path still requests a curve or edge")]
fn path_needs_a_viewport_hit(world: &mut ApplicationWorld) {
    let draft = world.state().sweep_preview.unwrap();
    assert!(draft.request.path_cleared);
    assert_eq!(draft.result, GuiSweepPreview::MissingPath);
    let text = super::super::world::squeezed(&world.window_text("Sweep"));
    assert!(text.contains("PathSelectCurveorEdge"), "{text}");
    assert!(!world.run_is_enabled("OK"));
}

#[then("the incomplete command closes")]
fn guided_draft_is_cancelled(world: &mut ApplicationWorld) {
    assert!(super::diagnostics::window_geometry("Sweep").is_none());
    assert!(world.state().sweep_preview.is_none());
    assert_eq!(world.state().browser.unwrap().bodies.len(), 1);
    assert!(world.state().browser.unwrap().surface_bodies.is_empty());
}
