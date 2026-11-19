// SPDX-License-Identifier: GPL-2.0-only
//! Verify the bottom document strip against the current workspace scenario (#4061).
use super::*;
use cucumber::{given, then, when};

fn geometry(name: &str) -> ([f32; 2], [f32; 2]) {
    window_geometry(name).unwrap_or_else(|| panic!("expected window {name:?}"))
}
fn viewport_area(world: &mut ApplicationWorld) -> [u32; 4] {
    world.viewport_bounds()
}
#[given("the ribbon and model browser are visible")]
fn ribbon_and_browser_visible(world: &mut ApplicationWorld) {
    geometry("##application-ribbon");
    geometry("Model");
    world.expect_text("3D Model");
}
#[when("the user inspects the workspace layout")]
fn inspect_layout(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
}
#[then("the document tab strip appears below the model browser and viewport")]
fn strip_below_workspace(world: &mut ApplicationWorld) {
    let (strip, _) = geometry("##open-document-tabs");
    let area = viewport_area(world);
    assert!(
        area[3] as f32 <= strip[1],
        "viewport {area:?}, strip {strip:?}"
    );
}
#[then("the status area appears below the document tab strip")]
fn status_below_strip(_: &mut ApplicationWorld) {
    let (strip, size) = geometry("##open-document-tabs");
    let (status, _) = geometry("##status-bar");
    assert!(
        strip[1] + size[1] <= status[1],
        "strip {strip:?} / {size:?}, status {status:?}"
    );
}
#[then("no document tab strip separates the ribbon from the viewport")]
fn ribbon_does_not_meet_strip(world: &mut ApplicationWorld) {
    let (ribbon, size) = geometry("##application-ribbon");
    let (strip, _) = geometry("##open-document-tabs");
    let area = viewport_area(world);
    assert!(
        ribbon[1] + size[1] <= area[1] as f32,
        "ribbon {ribbon:?} / {size:?}, viewport {area:?}"
    );
    assert!(
        (area[1] as f32) < strip[1],
        "viewport {area:?}, strip {strip:?}"
    );
}
#[then("the active document remains visible in the viewport")]
fn active_document_visible(world: &mut ApplicationWorld) {
    assert_eq!(world.state().document, world.previous_document);
    assert!(world.previous_document.is_some());
    let area = viewport_area(world);
    assert!(
        area[0] < area[2] && area[1] < area[3],
        "empty viewport {area:?}"
    );
    world.expect_text("End of Part");
}

#[cfg(target_os = "linux")]
#[test]
fn native_mouse_switches_tabs_in_the_bottom_strip() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let first = world.active_id();
    let second = world.add_part();
    strip_below_workspace(&mut world);
    status_below_strip(&mut world);
    let first_name = world.document_name(first);
    let point = super::super::diagnostics::pointer_pixels(world.expect_tab(&first_name).center());
    super::super::x11_input::click(point);
    world.pump(Vec::new());
    assert_eq!(world.state().document, Some(first));
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(super::super::solid_visibility::visible_solid_fill(
        &capture, &windows
    ));
    let second_name = world.document_name(second);
    let point = super::super::diagnostics::pointer_pixels(world.expect_tab(&second_name).center());
    super::super::x11_input::click(point);
    world.pump(Vec::new());
    assert_eq!(world.state().document, Some(second));
    let point = super::super::diagnostics::pointer_pixels(world.run("Documents").center());
    super::super::x11_input::click(point);
    world.pump(Vec::new());
    world.expect_text("Close All");
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn native_mouse_closes_the_selected_document_tab() {
    let mut world = ApplicationWorld::new();
    let first = world.add_part();
    let second = world.add_part();
    let name = world.document_name(second);
    let point = super::super::diagnostics::pointer_pixels(world.expect_tab(&name).center());
    super::super::x11_input::move_pointer(point);
    world.pump(Vec::new());
    let center = close_mark_center(&world, &name);
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    assert!(!world.document_is_open(second));
    assert!(world.document_is_open(first));
    assert_eq!(world.state().document, Some(first));
    world.close();
}

#[cfg(target_os = "linux")]
fn close_mark_center(world: &ApplicationWorld, name: &str) -> [f32; 2] {
    let run = world.expect_tab(name);
    let marks: Vec<_> = world
        .frame
        .marks
        .iter()
        .filter(|mark| {
            mark.list == run.list
                && mark.position[0] > run.maximum[0]
                && mark.position[0] < run.maximum[0] + 30.
                && (mark.position[1] - run.center()[1]).abs() < 10.
        })
        .map(|mark| mark.position)
        .collect();
    assert!(!marks.is_empty(), "tab {run:?}: expected its close mark");
    std::array::from_fn(|axis| {
        marks.iter().map(|mark| mark[axis]).sum::<f32>() / marks.len() as f32
    })
}

#[given("the model browser is docked at the left")]
fn model_browser_docked_left(world: &mut ApplicationWorld) {
    let (browser, _) = geometry("Model");
    assert!(
        browser[0] < world.extent()[0] as f32 / 4.,
        "browser {browser:?}"
    );
}
#[given("the 3D Model ribbon is expanded at the top")]
fn model_ribbon_expanded(world: &mut ApplicationWorld) {
    geometry("##application-ribbon");
    world.expect_text("3D Model");
    world.expect_text("Start 2D Sketch");
}
#[then("the model browser and viewport occupy the workspace below the ribbon")]
fn workspace_below_ribbon(world: &mut ApplicationWorld) {
    let (ribbon, size) = geometry("##application-ribbon");
    let (browser, _) = geometry("Model");
    let area = viewport_area(world);
    assert!(
        ribbon[1] + size[1] <= browser[1],
        "ribbon {ribbon:?} / {size:?}, browser {browser:?}"
    );
    assert!(
        ribbon[1] + size[1] <= area[1] as f32,
        "ribbon {ribbon:?} / {size:?}, viewport {area:?}"
    );
}
#[then("the model browser shows the part hierarchy at the left")]
fn model_hierarchy_visible(world: &mut ApplicationWorld) {
    geometry("Model");
    world.expect_text("Origin");
    world.expect_text("End of Part");
}
#[then("the viewport displays the part to the right of the browser")]
fn viewport_right_of_browser(world: &mut ApplicationWorld) {
    let (browser, size) = geometry("Model");
    let area = viewport_area(world);
    assert!(
        browser[0] + size[0] <= area[0] as f32,
        "browser {browser:?} / {size:?}, viewport {area:?}"
    );
    assert!(world.state().document.is_some());
}
#[then("the document tab strip shows Home and the active part below the workspace")]
fn tabs_show_home_and_active(world: &mut ApplicationWorld) {
    strip_below_workspace(world);
    let id = world.state().document.expect("expected an active part");
    let name = world.document_name(id);
    world.expect_tab("Home");
    world.expect_tab(&name);
}
#[then("the status area is visible along the bottom")]
fn status_at_bottom(world: &mut ApplicationWorld) {
    status_below_strip(world);
    let (position, size) = geometry("##status-bar");
    assert!((position[1] + size[1] - world.extent()[1] as f32).abs() < 1.);
}

#[given("the model browser is docked at the left with its default width")]
fn default_browser_left(world: &mut ApplicationWorld) {
    super::browser_feature::open_visible_extruded_part(world);
    model_browser_docked_left(world);
}

#[when("the user inspects the workspace proportions")]
#[when("the user inspects the workspace boundaries")]
#[when("the user inspects the bottom workspace rows")]
fn inspect_workspace_boundaries(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
}

#[then("the model browser occupies about one quarter of the available workspace width")]
fn browser_quarter_width(world: &mut ApplicationWorld) {
    let (_, size) = geometry("Model");
    let width = world.extent()[0] as f32;
    assert!(
        (size[0] / width - 0.25).abs() < 0.02,
        "browser {size:?}, width {width}"
    );
}

#[then("the viewport occupies the remaining width beside the browser")]
#[then("the viewport begins at the browser's shared divider without an empty gutter")]
fn viewport_shares_browser_edge(world: &mut ApplicationWorld) {
    let (browser, size) = geometry("Model");
    let area = viewport_area(world);
    assert!(
        (browser[0] + size[0] - area[0] as f32).abs() <= 1.,
        "browser {browser:?} / {size:?}, viewport {area:?}"
    );
}

#[then("the model browser begins at the left content edge directly below the ribbon")]
fn browser_starts_at_ribbon_edge(_: &mut ApplicationWorld) {
    let (browser, _) = geometry("Model");
    let (ribbon, size) = geometry("##application-ribbon");
    assert!((browser[0] - ribbon[0]).abs() <= 1.);
    assert!((browser[1] - ribbon[1] - size[1]).abs() <= 1.);
}

#[then("the viewport fills the remaining content width")]
fn viewport_fills_width(world: &mut ApplicationWorld) {
    let area = viewport_area(world);
    assert_eq!(area[2], world.extent()[0], "viewport {area:?}");
}

#[then("the ribbon, browser, viewport and status area use aligned outer edges")]
fn workspace_outer_edges(world: &mut ApplicationWorld) {
    browser_starts_at_ribbon_edge(world);
    viewport_fills_width(world);
    let (status, size) = geometry("##status-bar");
    assert_eq!(status[0], 0.);
    assert_eq!(size[0], world.extent()[0] as f32);
}

#[then("the browser extends to the status area above the bottom row")]
fn browser_reaches_status(_: &mut ApplicationWorld) {
    let (browser, size) = geometry("Model");
    let (status, _) = geometry("##status-bar");
    assert!((browser[1] + size[1] - status[1]).abs() <= 1.);
}

#[then("the document tab strip begins at the viewport's left edge")]
#[then("no document tabs appear below the model browser")]
fn tabs_start_at_viewport(world: &mut ApplicationWorld) {
    let (strip, _) = geometry("##open-document-tabs");
    let area = viewport_area(world);
    assert!((strip[0] - area[0] as f32).abs() <= 1.);
}

#[then("the document tab strip ends at the viewport's right edge")]
fn tabs_end_at_viewport(world: &mut ApplicationWorld) {
    let (list, size) = geometry("##document-list-control");
    let area = viewport_area(world);
    assert!((list[0] + size[0] - area[2] as f32).abs() <= 1.);
}

#[then("the status area spans the full application width")]
fn status_spans_window(world: &mut ApplicationWorld) {
    let (position, size) = geometry("##status-bar");
    assert_eq!((position[0], size[0]), (0., world.extent()[0] as f32));
}
