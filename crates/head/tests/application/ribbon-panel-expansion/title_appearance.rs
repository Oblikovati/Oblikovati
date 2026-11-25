// SPDX-License-Identifier: GPL-2.0-only
//! Appearance selection from an expanded panel title.
use super::{ApplicationWorld, WINDOW, diagnostics};
use cucumber::{given, then, when};

#[given("the ribbon appearance is \"Normal\"")]
fn normal_appearance(world: &mut ApplicationWorld) {
    let title = diagnostics::pointer_pixels(world.run("Modify").center());
    world.right_click(title);
    world.click_text("Ribbon Appearance");
    assert!(world.checked("Normal"));
    world.escape();
    world.escape();
}
#[when("the user right-clicks the lower Modify title and opens \"Ribbon Appearance\"")]
fn open_appearance(world: &mut ApplicationWorld) {
    let title = diagnostics::pointer_pixels(world.run_in_window(WINDOW, "Modify").center());
    world.right_click(title);
    world.click_text("Ribbon Appearance");
}
#[when("the user opens the \"Ribbon Appearance\" submenu")]
fn open_appearance_submenu(world: &mut ApplicationWorld) {
    world.click_text("Ribbon Appearance");
}
#[then("\"Normal\" is checked and the Modify additional commands remain visible")]
fn normal_checked_with_expansion(world: &mut ApplicationWorld) {
    assert!(world.checked("Normal"));
    assert!(diagnostics::window_geometry(WINDOW).is_some());
    world.expect_text("Move Bodies");
}
#[when("the user selects the checked \"Normal\" option")]
fn select_normal(world: &mut ApplicationWorld) {
    assert!(world.checked("Normal"));
    world.click_text("Normal");
}
#[then("the ribbon displays command icons without their command labels")]
fn icon_only_ribbon(world: &mut ApplicationWorld) {
    let (position, size) = diagnostics::window_geometry("##application-ribbon").unwrap();
    let icons =
        super::super::ribbon_appearance::icon_quads::ribbon_icon_quads(position[1] + size[1]);
    assert!(icons.len() >= 4, "expected visible command icons");
    let ribbon = world.window_text("##application-ribbon");
    assert!(!ribbon.contains("Extrude") && !ribbon.contains("Hole"));
}
#[then("\"Text Off\" is checked")]
fn text_off_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Text Off"));
}
#[then("the full ribbon displays its command labels again")]
fn restored_labels(world: &mut ApplicationWorld) {
    full_ribbon_with_labels(world);
}
#[then("the full ribbon remains visible with command labels")]
fn full_ribbon_with_labels(world: &mut ApplicationWorld) {
    world.run_in_window("##application-ribbon", "Extrude");
    world.run_in_window("##application-ribbon", "Hole");
}

#[cfg(target_os = "linux")]
#[test]
fn x11_expansion_title_changes_and_restores_appearance() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_modify_native(&mut world);
    open_native_title_menu(&mut world);
    click_native(&mut world, "Ribbon Appearance");
    click_native(&mut world, "Text Off");
    assert!(diagnostics::window_geometry(WINDOW).is_none());
    icon_only_ribbon(&mut world);
    open_modify_native(&mut world);
    open_native_title_menu(&mut world);
    click_native(&mut world, "Ribbon Appearance");
    text_off_checked(&mut world);
    click_native(&mut world, "Normal");
    full_ribbon_with_labels(&mut world);
    assert!(diagnostics::window_geometry(WINDOW).is_none());
    world.close();
}
#[cfg(target_os = "linux")]
fn open_modify_native(world: &mut ApplicationWorld) {
    let title = world.run("Modify");
    let point = [title.maximum[0] - 2., title.center()[1]];
    super::super::x11_input::click(diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
}
#[cfg(target_os = "linux")]
fn click_native(world: &mut ApplicationWorld, label: &str) {
    let position = diagnostics::pointer_pixels(world.run(label).center());
    super::super::x11_input::click(position);
    world.pump(Vec::new());
}
#[cfg(target_os = "linux")]
fn open_native_title_menu(world: &mut ApplicationWorld) {
    let position = diagnostics::pointer_pixels(world.run_in_window(WINDOW, "Modify").center());
    super::super::x11_input::right_click(position);
    world.pump(Vec::new());
}

#[cfg(target_os = "linux")]
#[test]
fn x11_pinned_expansion_survives_appearance_changes() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_modify_native(&mut world);
    let pin = world.run("Pin");
    let position = [pin.minimum[0] + 6., pin.center()[1]];
    super::super::x11_input::click(diagnostics::pointer_pixels(position));
    world.pump(Vec::new());
    open_native_title_menu(&mut world);
    click_native(&mut world, "Ribbon Appearance");
    click_native(&mut world, "Text Off");
    assert!(diagnostics::window_geometry(WINDOW).is_some());
    icon_only_ribbon(&mut world);
    open_native_title_menu(&mut world);
    click_native(&mut world, "Ribbon Appearance");
    click_native(&mut world, "Normal");
    assert!(diagnostics::window_geometry(WINDOW).is_some());
    world.expect_text("Move Bodies");
    world.close();
}
