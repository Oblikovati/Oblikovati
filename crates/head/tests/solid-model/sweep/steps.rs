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
        world.state().sweep_preview,
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
    assert_eq!(world.state().sweep_preview, Some(GuiSweepPreview::Ready));
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
