// SPDX-License-Identifier: GPL-2.0-only
//! Exercise the observed Appearance and Material Browser docking workflows.
use super::{ApplicationWorld, diagnostics, native_drag, resize_boundary, viewport};
use cucumber::{given, then, when};
const APPEARANCE: &str = "Appearance Browser";
const MATERIAL: &str = "Material Browser";

#[given("Appearance Browser is floating over the workspace")]
fn appearance_floating(world: &mut ApplicationWorld) {
    let path = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../test-utilities/fixtures/docking-visible-solid.opd");
    world.reopen_through_dialog(&path);
    world.previous_document = Some(world.active_id());
    open_floating(world, APPEARANCE);
}

fn open_floating(world: &mut ApplicationWorld, panel: &str) {
    super::super::workspace_layout::dockspace_native::open_toolbar_tool(world, panel);
    assert_eq!(diagnostics::window_dock_id(panel), Some(0));
}

#[when("the user drags the Appearance Browser title bar to the far right workspace edge")]
fn dock_appearance_right(world: &mut ApplicationWorld) {
    let area = viewport(world);
    let (position, size) = diagnostics::window_geometry(APPEARANCE).unwrap();
    let start = [
        (position[0] + size[0] * 0.4) as i32,
        (position[1] + 10.) as i32,
    ];
    native_drag(
        world,
        start,
        [area[2] as i32 - 15, ((area[1] + area[3]) / 2) as i32],
    );
}

#[given("Appearance Browser is docked on the right")]
fn appearance_docked(world: &mut ApplicationWorld) {
    appearance_floating(world);
    dock_appearance_right(world);
    appearance_on_right(world);
}

#[then("Appearance Browser docks along the right edge")]
fn appearance_on_right(world: &mut ApplicationWorld) {
    let (position, size) = diagnostics::window_geometry(APPEARANCE).unwrap();
    assert_ne!(diagnostics::window_dock_id(APPEARANCE), Some(0));
    assert!((position[0] + size[0] - world.extent()[0] as f32).abs() <= 1.);
}

#[then("the Model Browser remains visible on the left")]
fn model_browser_left(_: &mut ApplicationWorld) {
    let (position, _) = diagnostics::window_geometry("Model").expect("model browser");
    assert!(position[0].abs() <= 1., "model browser at {position:?}");
}

#[then("the viewport occupies the space between both browser panels")]
#[then("the viewport remains visible to the left of the dock region")]
fn viewport_between_browsers(world: &mut ApplicationWorld) {
    let area = viewport(world);
    let (left, left_size) = diagnostics::window_geometry("Model").unwrap();
    let (right, _) = diagnostics::window_geometry(APPEARANCE).unwrap();
    assert!(area[0] as f32 >= left[0] + left_size[0] - 1.);
    assert!(area[2] as f32 <= right[0] + 1. && area[0] < area[2]);
    super::super::workspace_layout::browser_feature::expect_solid_visible(world);
}

#[then("the active part document remains open")]
fn active_part_remains(world: &mut ApplicationWorld) {
    assert_eq!(world.state().document, world.previous_document);
    super::super::workspace_layout::browser_feature::expect_solid_visible(world);
}

#[when("the user drags the shared boundary toward the viewport to widen Appearance Browser")]
fn widen_appearance(world: &mut ApplicationWorld) {
    let area = viewport(world);
    let (_, size) = diagnostics::window_geometry(APPEARANCE).unwrap();
    world.docking_resize = Some((area, size[0]));
    resize_boundary(world, area, -80);
}

#[then("Appearance Browser becomes wider")]
fn appearance_wider(world: &mut ApplicationWorld) {
    let (_, before) = world.docking_resize.unwrap();
    let (_, size) = diagnostics::window_geometry(APPEARANCE).unwrap();
    assert!(size[0] > before + 40., "width {before} -> {}", size[0]);
}

#[then("the view orientation control remains visible in the viewport")]
fn orientation_visible(world: &mut ApplicationWorld) {
    let (position, size) =
        diagnostics::window_geometry("##orientation-control").expect("visible orientation control");
    let area = viewport(world);
    assert!(position[0] >= area[0] as f32 && position[1] >= area[1] as f32);
    assert!(position[0] + size[0] <= area[2] as f32 + 1.);
    assert!(position[1] + size[1] <= area[3] as f32 + 1.);
}

#[when("the user restores the boundary to its earlier position")]
fn restore_appearance_boundary(world: &mut ApplicationWorld) {
    let current = viewport(world);
    let (before, _) = world.docking_resize.unwrap();
    resize_boundary(world, current, before[2] as i32 - current[2] as i32);
}

#[then("the docked panel and viewport return to their earlier proportions")]
fn appearance_proportions_restored(world: &mut ApplicationWorld) {
    let (before, width) = world.docking_resize.unwrap();
    let current = viewport(world);
    let (_, size) = diagnostics::window_geometry(APPEARANCE).unwrap();
    assert!(before[2].abs_diff(current[2]) <= 1);
    assert!((width - size[0]).abs() <= 1.);
}

#[given("Material Browser is floating over the workspace")]
fn material_floating(world: &mut ApplicationWorld) {
    open_floating(world, MATERIAL);
}

#[when("the user drags the Material Browser title bar onto the Appearance Browser tab strip")]
fn join_browser_tabs(world: &mut ApplicationWorld) {
    let (source, size) = diagnostics::window_geometry(MATERIAL).unwrap();
    let (target, target_size) = diagnostics::window_geometry(APPEARANCE).unwrap();
    let start = [(source[0] + size[0] * 0.4) as i32, (source[1] + 10.) as i32];
    let end = [
        (target[0] + target_size[0] * 0.5) as i32,
        (target[1] + 10.) as i32,
    ];
    native_drag(world, start, end);
}

#[then("both browsers occupy one right-side dock region as tabs")]
#[then("Appearance Browser returns as a tab in the right dock region")]
fn browsers_share_dock(_: &mut ApplicationWorld) {
    let node = diagnostics::window_dock_id(APPEARANCE).unwrap();
    assert_ne!(node, 0);
    assert_eq!(diagnostics::window_dock_id(MATERIAL), Some(node));
}

#[then("one browser is active while the other remains available")]
fn one_browser_active(_: &mut ApplicationWorld) {
    assert_ne!(
        diagnostics::window_dock_tab_visible(APPEARANCE),
        diagnostics::window_dock_tab_visible(MATERIAL)
    );
}

#[when("the user selects the Appearance Browser tab")]
fn select_appearance(world: &mut ApplicationWorld) {
    select_tab(world, APPEARANCE);
}
#[when("the user selects the Material Browser tab")]
fn select_material(world: &mut ApplicationWorld) {
    select_tab(world, MATERIAL);
}

fn select_tab(world: &mut ApplicationWorld, panel: &str) {
    let (min, max) = diagnostics::window_dock_tab_bounds(panel).unwrap();
    world.click(diagnostics::pointer_pixels([
        (min[0] + max[0]) * 0.5,
        (min[1] + max[1]) * 0.5,
    ]));
    world.pump(Vec::new());
    assert_eq!(diagnostics::window_dock_tab_visible(panel), Some(true));
}

#[then("Appearance Browser contents become visible in the dock region")]
fn appearance_contents(world: &mut ApplicationWorld) {
    world.run_in_window(APPEARANCE, "Assigntopart");
}
#[then("Material Browser contents become visible in the dock region")]
fn material_contents(world: &mut ApplicationWorld) {
    world.run_in_window(MATERIAL, "Assigntopart");
}

#[when("the user selects Appearance Browser and closes it from its panel controls")]
fn close_appearance(world: &mut ApplicationWorld) {
    select_tab(world, APPEARANCE);
    let close = world.run_in_window(APPEARANCE, "Close").center();
    super::super::x11_input::click(diagnostics::pointer_pixels(close));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[then("Appearance Browser is removed from the tab group")]
fn appearance_closed(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(APPEARANCE).is_none());
}
#[then("Material Browser remains visible in the right dock region")]
fn material_remains(world: &mut ApplicationWorld) {
    assert_eq!(diagnostics::window_dock_tab_visible(MATERIAL), Some(true));
    world.run_in_window(MATERIAL, "Assigntopart");
}

#[when("the user opens View and then User Interface")]
fn open_view_interface(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.click_text("User Interface");
}
#[when("the user selects Appearance Browser")]
fn restore_appearance(world: &mut ApplicationWorld) {
    world.click_text(APPEARANCE);
    world.pump(Vec::new());
    world.docking_selected_tab = Some(APPEARANCE.to_owned());
}
