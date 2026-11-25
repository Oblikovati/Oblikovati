// SPDX-License-Identifier: GPL-2.0-only
//! Move an icon-only user command through the native expanded panel.
use super::*;
use std::collections::BTreeSet;

const EXPANSION: &str = "##ribbon-additional-UserCommands";
fn main_icon(world: &ApplicationWorld) -> [f32; 2] {
    let title = world.run("User Commands");
    [title.minimum[0] + 10., title.minimum[1] - 56.]
}
#[cucumber::given("2D Sketches has been added to the Part Tools custom panel")]
fn add_sketch_command(world: &mut ApplicationWorld) {
    world.click_text("Customize");
    super::super::ribbon_customization::stage_sketch_visibility(world);
    super::super::ribbon_customization::click_in_customize(world, "OK");
    world.expect_text("User Commands");
}
#[cucumber::given("2D Sketches has been moved into the User Commands expanded area")]
fn move_sketch_command(world: &mut ApplicationWorld) {
    right_click_main_icon(world);
    move_to_expanded(world);
}
fn expanded_icon() -> [f32; 2] {
    let (position, _) = diagnostics::window_geometry(EXPANSION).unwrap();
    [position[0] + 14., position[1] + 18.]
}
fn icon_colors(world: &mut ApplicationWorld, point: [f32; 2]) -> usize {
    let [x, y] = diagnostics::pointer_pixels(point);
    let capture = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let width = capture.extent[0] as usize;
    let mut colors = BTreeSet::new();
    for row in (y - 6)..=(y + 6) {
        for column in (x - 6)..=(x + 6) {
            let offset = (row as usize * width + column as usize) * 4;
            colors.insert([
                capture.rgba[offset],
                capture.rgba[offset + 1],
                capture.rgba[offset + 2],
            ]);
        }
    }
    colors.len()
}
#[then("the 2D Sketches icon appears in the User Commands main panel")]
#[then("the 2D Sketches icon returns to the main panel")]
fn main_icon_visible(world: &mut ApplicationWorld) {
    let point = main_icon(world);
    assert!(
        icon_colors(world, point) > 1,
        "expected the rendered main-panel icon"
    );
}
#[when("the user right-clicks that icon")]
fn right_click_main_icon(world: &mut ApplicationWorld) {
    world.right_click(diagnostics::pointer_pixels(main_icon(world)));
}
#[then("its context menu offers Move to Expanded Panel")]
fn move_to_expanded_offered(world: &mut ApplicationWorld) {
    world.expect_text("Move to Expanded Panel");
}
#[when("the user selects Move to Expanded Panel")]
fn move_to_expanded(world: &mut ApplicationWorld) {
    world.click_text("Move to Expanded Panel");
    world.pump(Vec::new());
}
#[then("the icon is absent from the main panel")]
#[then("the 2D Sketches icon is absent from its main area")]
fn main_icon_absent(world: &mut ApplicationWorld) {
    let point = main_icon(world);
    assert_eq!(icon_colors(world, point), 1, "expected no main-panel icon");
}
#[then("the User Commands title offers an expansion control")]
#[then("the User Commands panel is collapsed and offers an expansion control")]
fn expansion_control(world: &mut ApplicationWorld) {
    let title = world.run("User Commands");
    assert!(
        title.text.ends_with(['v', '▼'])
            || world.frame.runs.iter().any(|run| matches!(run.text.as_str(), "v" | "▼")
                && run.minimum[0] > title.maximum[0]
                && (run.center()[1] - title.center()[1]).abs() < 8.)
    );
}
#[when("the user opens the User Commands expansion")]
#[when("the user opens the User Commands expanded area")]
fn open_user_expansion(world: &mut ApplicationWorld) {
    world.click_text("User Commands");
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry(EXPANSION).is_some());
}
#[then("the 2D Sketches icon appears in the expanded area")]
#[then("the 2D Sketches icon is visible in that area")]
fn expanded_icon_visible(world: &mut ApplicationWorld) {
    assert!(
        icon_colors(world, expanded_icon()) > 1,
        "expected the rendered expanded icon"
    );
}
#[then("the Tools expanded area is no longer visible")]
fn tools_expansion_hidden(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(EXPANSION).is_none());
}
#[cfg(target_os = "linux")]
#[test]
fn x11_custom_expansion_survives_a_ribbon_tab_round_trip() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.click_text("Tools");
    add_sketch_command(&mut world);
    move_sketch_command(&mut world);
    open_user_expansion(&mut world);
    click_native_label(&mut world, "3D Model");
    tools_expansion_hidden(&mut world);
    click_native_label(&mut world, "Tools");
    expansion_control(&mut world);
    main_icon_absent(&mut world);
    click_native_label(&mut world, "User Commands");
    expanded_icon_visible(&mut world);
    world.close();
}
#[cfg(target_os = "linux")]
fn click_native_label(world: &mut ApplicationWorld, label: &str) {
    let position = diagnostics::pointer_pixels(world.run(label).center());
    super::super::x11_input::click(position);
    world.pump(Vec::new());
}
#[when("the user right-clicks the expanded icon")]
fn right_click_expanded_icon(world: &mut ApplicationWorld) {
    world.right_click(diagnostics::pointer_pixels(expanded_icon()));
}
#[when("the user selects Move to Main Panel")]
fn move_to_main(world: &mut ApplicationWorld) {
    world.click_text("Move to Main Panel");
    world.pump(Vec::new());
}
#[then("the expansion closes")]
fn expansion_closed(_world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(EXPANSION).is_none());
    assert!(diagnostics::window_geometry("##ribbon-additional-ToolClipboard").is_none());
}
