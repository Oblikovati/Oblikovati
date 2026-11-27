// SPDX-License-Identifier: GPL-2.0-only
//! Check the Home view on a new desktop session with no document.

use super::super::diagnostics::window_dock_id;
use super::*;

#[given("the MCAD Software is launched without opening a document")]
fn start_without_recent_document(world: &mut ApplicationWorld) {
    let state = world.state();
    assert!(state.documents.is_empty());
}

#[then("the Get Started tab is selected")]
fn startup_ribbon_is_get_started(world: &mut ApplicationWorld) {
    world.run_in_window("##application-ribbon", "GetStarted");
    world.run_in_window("##application-ribbon", "NewPart");
}

#[then("the Home workspace offers a New Part action")]
fn home_offers_new_part(world: &mut ApplicationWorld) {
    assert!(home_shows(world, "New Part"), "expected New Part in Home");
}

#[then("the Model browser reports that no document is open")]
fn model_reports_no_document(world: &mut ApplicationWorld) {
    world.run_in_window("Model", "(nodocument)");
}

#[then("Material Browser and Appearance Browser are available in the right dock")]
pub(super) fn startup_asset_browsers_are_docked(world: &mut ApplicationWorld) {
    let material = window_dock_id("Material Browser").expect("expected Material Browser docked");
    let appearance =
        window_dock_id("Appearance Browser").expect("expected Appearance Browser docked");
    assert_ne!(material, 0, "expected an attached asset dock");
    assert_eq!(material, appearance, "expected the same asset dock");
    let (position, _) = window_geometry("Material Browser").expect("expected the asset dock");
    assert!(
        position[0] >= world.extent()[0] as f32 / 2.,
        "expected the right dock"
    );
}

#[then("the selected browser panel asks the user to select a part to browse its assets")]
fn selected_asset_browser_requests_a_part(world: &mut ApplicationWorld) {
    world.expect_text("Select a part to browse its assets.");
    world.save_capture("home-without-a-document");
}

#[test]
fn new_part_from_startup_home_opens_the_part_workspace() {
    let mut world = ApplicationWorld::new();
    start_without_recent_document(&mut world);
    assert!(home_shows(&world, "No recent documents"));
    home_offers_new_part(&mut world);
    for panel in ["Model", "Material Browser", "Appearance Browser"] {
        assert!(window_geometry(panel).is_none(), "{panel} on Home");
    }
    let [left, _, right, _] = world.viewport_bounds();
    assert_eq!([left, right], [0, world.extent()[0]]);
    world.save_capture("home-without-empty-docks");
    open_part_from_home(&mut world);
    world.close();
}

fn open_part_from_home(world: &mut ApplicationWorld) {
    let action = home_run(world, "New Part").expect("expected New Part in Home");
    let at = super::super::diagnostics::pointer_pixels(action.center());
    if world.app().window.native.view.uses_wayland() {
        world.click(at);
    } else {
        super::super::x11_input::click(at);
        world.pump(Vec::new());
    }
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
    world.run_in_window("##application-ribbon", "3DModel");
    assert!(!home_shows(world, "Recent"));
    world.pump(Vec::new());
    assert!(window_dock_id("Model").is_some_and(|id| id != 0));
    startup_asset_browsers_are_docked(world);
}
