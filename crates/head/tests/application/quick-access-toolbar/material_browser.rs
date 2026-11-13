// SPDX-License-Identifier: GPL-2.0-only
//! Open and use the Material Browser from the native quick access toolbar.
use super::*;

#[given("the Tools ribbon is selected")]
fn tools_selected(world: &mut ApplicationWorld) {
    world.click_text("Tools");
    world.expect_text("Material and Appearance");
}
#[given("the Material Browser quick access command is visible")]
fn browser_command_visible(world: &mut ApplicationWorld) {
    assert!(on_row(world, "MaterialBrowser"));
    assert!(world.run_is_enabled("Material Browser"));
}
#[given("the Material Browser window is closed")]
#[then("the browser window closes")]
fn browser_closed(_world: &mut ApplicationWorld) {
    assert!(window_geometry("Material Browser").is_none());
}
#[when("the user selects the Material Browser quick access command")]
fn open_browser(world: &mut ApplicationWorld) {
    world.click(row_center(world, "MaterialBrowser"));
}
#[then("the Material Browser window opens over the part workspace")]
fn browser_open(world: &mut ApplicationWorld) {
    assert!(window_geometry("Material Browser").is_some());
    world.run_in_window("Material Browser", "Materials");
    world.run_in_window("Material Browser", "Assigntopart");
}
#[when("the user closes the Material Browser window")]
fn close_browser(world: &mut ApplicationWorld) {
    let position = world.run_in_window("Material Browser", "Close").center();
    world.click(super::super::diagnostics::pointer_pixels(position));
}
#[then("the Tools ribbon and active part remain visible")]
fn tools_and_part(world: &mut ApplicationWorld) {
    world.expect_text("Material and Appearance");
    part_stays(world);
}
#[test]
fn material_browser_assigns_and_undoes_a_catalog_material() {
    let mut world = ApplicationWorld::new();
    part_active(&mut world);
    tools_selected(&mut world);
    open_browser(&mut world);
    browser_open(&mut world);
    let browser = world.state().browser.unwrap();
    let choice = browser
        .materials
        .get(1)
        .expect("expected two catalog materials");
    let (material, name) = (choice.id.clone(), choice.name.clone());
    let field = world.run_in_window("Material Browser", "Material").clone();
    world.click([(field.minimum[0] - 40.) as i32, field.center()[1] as i32]);
    world.click_text(&name);
    world.click_text("Assign to part");
    assert_eq!(
        world.state().browser.unwrap().part_material.as_deref(),
        Some(material.as_str())
    );
    world.save_capture("material-browser-assigned");
    close_browser(&mut world);
    browser_closed(&mut world);
    world.click_text("Undo");
    assert_eq!(world.state().browser.unwrap().part_material, None);
    world.close();
}

#[test]
fn material_browser_assigns_appearance_without_leaking_to_another_part() {
    let mut world = ApplicationWorld::new();
    part_active(&mut world);
    tools_selected(&mut world);
    open_browser(&mut world);
    world.click_text("Appearances");
    let appearance = world.state().browser.unwrap().appearances[0].id.clone();
    world.click_text("Assign to part");
    assert_eq!(
        world.state().browser.unwrap().part_appearance.as_deref(),
        Some(appearance.as_str())
    );
    world.add_part();
    assert_eq!(world.state().browser.unwrap().part_appearance, None);
    assert!(window_geometry("Material Browser").is_some());
    close_browser(&mut world);
    world.close();
}
