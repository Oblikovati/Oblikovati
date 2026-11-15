// SPDX-License-Identifier: GPL-2.0-only
//! Real persisted assembly for the express-open journey.
use super::*;

const COMPONENT: &str = "visible-component.opd";
const OCCURRENCE: &str = "Visible:1";

fn write_assembly(folder: &Path) {
    let part = super::super::workspace_layout::browser_feature::extruded_part_bytes(
        "Extrude1", "Profile", 2.,
    );
    std::fs::write(folder.join(COMPONENT), part).unwrap();
    let assembly = format!(
        "schemaVersion: 2\ndocumentType: 2\nmodel: {{occurrences: [{{name: '{OCCURRENCE}', component: {COMPONENT}}}]}}\n"
    );
    std::fs::write(folder.join(ASSEMBLY_FILE), assembly).unwrap();
}
#[given("a saved assembly with visible components is available and not already open")]
fn saved_assembly(world: &mut ApplicationWorld) {
    let drawing = world.active_id();
    assert_eq!(
        world.state().document_type,
        Some(oblikovati_api::types::DOCUMENT_DRAWING)
    );
    let folder = models_folder(world);
    write_assembly(&folder);
    let path = folder.join(ASSEMBLY_FILE);
    let state = world.state();
    assert!(
        state
            .document_storage
            .values()
            .all(|storage| { storage.path.as_deref() != path.to_str() }),
        "fixture assembly {path:?}: expected it not already open"
    );
    assert_eq!(world.active_id(), drawing);
    world.other_document = Some(drawing);
}
#[when("the user opens the file selection dialog")]
fn file_selection_dialog(world: &mut ApplicationWorld) {
    open_dialog(world);
}
#[when("the user selects the assembly and enable Load in Express Mode")]
fn select_express(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    look_in(world, &folder);
    select_row(world, ASSEMBLY_FILE);
    world.click_text("Load in Express Mode");
    assert!(world.checked("Load in Express Mode"));
}
#[then("the assembly opens in a new active document tab")]
fn assembly_opens(world: &mut ApplicationWorld) {
    world.expect_no_text("Look in");
    let state = world.state();
    assert_eq!(
        state.document_type,
        Some(oblikovati_api::types::DOCUMENT_ASSEMBLY)
    );
    assert_ne!(state.document, world.other_document);
}
#[then("its component geometry is visible in the viewport")]
fn component_visible(world: &mut ApplicationWorld) {
    let status = world.app().window.native.scene_status().unwrap();
    assert_eq!(status.visible_bodies, 1, "scene {status:?}");
    assert!(
        status.scene_errors.is_empty(),
        "scene errors {:?}",
        status.scene_errors
    );
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(super::super::solid_visibility::visible_solid_fill(
        &capture, &windows
    ));
}
#[then("the assembly browser lists its components")]
fn assembly_components(world: &mut ApplicationWorld) {
    world.expect_text(OCCURRENCE);
    world.expect_no_text("[unresolved]");
}
#[then("the assembly ribbon offers Load Full")]
fn offers_load_full(world: &mut ApplicationWorld) {
    world.expect_text("Load Full");
    assert!(world.run_is_enabled("Load Full"));
}
#[when("the user chooses Load Full")]
fn load_full(world: &mut ApplicationWorld) {
    world.click_text("Load Full");
    settle(world);
}
#[then("the assembly remains active with its component geometry displayed")]
fn still_visible(world: &mut ApplicationWorld) {
    assembly_opens(world);
    component_visible(world);
}
#[then("Load Full is no longer displayed in the ribbon")]
fn no_load_full(world: &mut ApplicationWorld) {
    world.expect_no_text("Load Full");
}
#[then("the Finish and Simplify commands are available")]
fn full_commands(world: &mut ApplicationWorld) {
    for label in ["Finish", "Simplify"] {
        world.expect_text(label);
        assert!(world.run_is_enabled(label), "expected {label:?} enabled");
    }
}
#[test]
fn saved_assembly_fixture_opens_its_real_visible_component() {
    let mut world = ApplicationWorld::new();
    world.add_drawing();
    saved_assembly(&mut world);
    file_selection_dialog(&mut world);
    let folder = models_folder(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, ASSEMBLY_FILE);
    choose_open(&mut world);
    assembly_opens(&mut world);
    component_visible(&mut world);
    assembly_components(&mut world);
    world.close();
}
