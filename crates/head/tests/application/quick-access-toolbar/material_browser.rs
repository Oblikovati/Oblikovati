// SPDX-License-Identifier: GPL-2.0-only
//! Open and use the Material Browser from the native quick access toolbar.
use super::*;

#[test]
fn appearance_browser_opens_its_own_window_and_assigns_an_appearance() {
    let mut world = ApplicationWorld::with_asset_tools_closed();
    part_active(&mut world);
    world.click(row_center(&world, "MaterialBrowser"));
    world.click(row_center(&world, "AppearanceBrowser"));
    world.pump(Vec::new());
    assert!(window_geometry("Material Browser").is_some());
    assert!(window_geometry("Appearance Browser").is_some());
    let chosen = world.state().browser.unwrap().appearances[0].id.clone();
    let button = world
        .run_in_window("Appearance Browser", "Assigntopart")
        .center();
    world.click(super::super::diagnostics::pointer_pixels(button));
    assert_eq!(
        world.state().browser.unwrap().part_appearance.as_deref(),
        Some(chosen.as_str())
    );
    let (position, size) = window_geometry("Appearance Browser").unwrap();
    let close = [position[0] + size[0] - 13., position[1] + 10.];
    world.click(super::super::diagnostics::pointer_pixels(close));
    assert!(window_geometry("Appearance Browser").is_none());
    assert!(window_geometry("Material Browser").is_some());
    world.click(row_center(&world, "Undo"));
    assert_eq!(world.state().browser.unwrap().part_appearance, None);
    world.close();
}

#[test]
fn quick_access_materials_assigns_a_material_and_undoes_it() {
    let mut world = ApplicationWorld::new();
    part_active(&mut world);
    let browser = world.state().browser.unwrap();
    let choice = browser
        .materials
        .get(1)
        .expect("expected two catalog materials");
    let (id, name) = (choice.id.clone(), choice.name.clone());
    world.click(row_center(&world, "Materials"));
    world.click_text(&name);
    assert_eq!(
        world.state().browser.unwrap().part_material.as_deref(),
        Some(id.as_str())
    );
    world.click(row_center(&world, "Undo"));
    assert_eq!(world.state().browser.unwrap().part_material, None);
    for visible in [false, true] {
        open_customization(&mut world);
        assert_ne!(world.checked("Materials"), visible);
        let choice = world
            .frame
            .runs
            .iter()
            .rev()
            .find(|run| run.text == "Materials")
            .unwrap()
            .center();
        world.click(super::super::diagnostics::pointer_pixels(choice));
        assert_eq!(on_row(&world, "Materials"), visible);
    }
    world.close();
}

#[given("the Tools ribbon is selected")]
#[given("the full Tools ribbon is selected")]
fn tools_selected(world: &mut ApplicationWorld) {
    world.click_text("Tools");
    world.expect_text("Material and Appearance");
}
#[given("the Material Browser quick access command is visible")]
fn browser_command_visible(world: &mut ApplicationWorld) {
    assert!(on_row(world, "MaterialBrowser"));
    let icon = super::super::quick_access_icons::standard_icon("Material Browser").unwrap();
    assert_eq!(icon.alpha, 255, "expected an enabled Material Browser icon");
}
#[given("the Material Browser window is closed")]
fn prepare_closed_browser(world: &mut ApplicationWorld) {
    use super::super::diagnostics::{pointer_pixels, window_dock_tab_bounds};
    use super::super::workspace_layout::dockspace_native::{native_drag, viewport};
    if let Some((min, max)) = window_dock_tab_bounds("Material Browser") {
        let start = pointer_pixels(std::array::from_fn(|axis| (min[axis] + max[axis]) * 0.5));
        let area = viewport(world);
        let target = std::array::from_fn(|axis| ((area[axis] + area[axis + 2]) / 2) as i32);
        native_drag(world, start, target);
    }
    close_browser(world);
    world.pump(Vec::new());
    browser_closed(world);
}
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
    assert_eq!(
        super::super::diagnostics::window_dock_id("Material Browser"),
        Some(0)
    );
    world.run_in_window("Material Browser", "Material");
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
    let mut world = ApplicationWorld::with_asset_tools_closed();
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
