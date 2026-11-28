// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for User Coordinate System scenarios (user-coordinate-system-01..05, #3918).
use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiUcsStage;
use oblikovati_viewport::ViewportNavigation;

fn first_selectable_vertex(world: &mut ApplicationWorld) -> [f64; 3] {
    let document = world
        .state()
        .document
        .expect("expected an open Part document");
    let vertices = world
        .app()
        .owner
        .ucs_solid_vertices(document)
        .expect("expected selectable solid vertices");
    vertices
        .into_iter()
        .map(|(_, point)| point)
        .max_by(|a, b| (a[0] + a[1] + a[2]).total_cmp(&(b[0] + b[1] + b[2])))
        .expect("expected at least one selectable solid vertex")
}

fn fit_view_home(world: &mut ApplicationWorld) {
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.25 },
    )
    .expect("home navigation");
    world.pump(Vec::new());
}

fn start_ucs_at_solid_vertex(world: &mut ApplicationWorld) {
    fit_view_home(world);
    let vertex = first_selectable_vertex(world);
    world.click_text("UCS");
    world.pump(Vec::new());
    let pixel = crate::HeadWindow::model_pixel(&world.app().window.native, vertex)
        .expect("vertex should project onto viewport");
    world.click(pixel);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

fn project_world(world: &mut ApplicationWorld, point: [f64; 3]) -> [i32; 2] {
    crate::HeadWindow::model_pixel(&world.app().window.native, point)
        .unwrap_or_else(|cause| panic!("expected model point {point:?} to project: {cause}"))
}

fn click_displayed_x_direction(world: &mut ApplicationWorld) {
    let triad = world
        .state()
        .ucs
        .active
        .expect("expected active UCS placement")
        .preview_triad
        .expect("expected displayed UCS triad");
    let pixel = project_world(world, triad.x_tip_world);
    world.click(pixel);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

fn click_displayed_y_direction(world: &mut ApplicationWorld) {
    let triad = world
        .state()
        .ucs
        .active
        .expect("expected active UCS placement")
        .preview_triad
        .expect("expected displayed UCS triad");
    let pixel = project_world(world, triad.y_tip_world);
    // Click slightly offset near the displayed Y direction to mirror recorded user clicks.
    world.click([pixel[0] + 2, pixel[1] - 2]);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

fn right_click_ucs1(world: &mut ApplicationWorld) {
    let at = world.run_in_window("Model", "UCS1").center();
    world.right_click(diagnostics::pointer_pixels(at));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

fn expand_ucs1(world: &mut ApplicationWorld) {
    let run = world.run_in_window("Model", "UCS1");
    let label_left = diagnostics::caption_icon_left(run).unwrap_or(run.minimum[0]);
    let arrow = diagnostics::pointer_pixels([label_left - 10., run.center()[1]]);
    world.click(arrow);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[given("the rectangular solid has a selectable model vertex")]
fn rectangular_solid_has_selectable_vertex(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(
        browser.bodies.len(),
        1,
        "expected rectangular solid body, saw {:?}",
        browser.bodies
    );
    let _ = first_selectable_vertex(world);
}

#[when("the user starts UCS and selects the vertex")]
fn start_ucs_and_select_vertex(world: &mut ApplicationWorld) {
    start_ucs_at_solid_vertex(world);
}

#[then("the MCAD Software displays a triad at the selected location")]
fn displays_triad_at_selected_location(world: &mut ApplicationWorld) {
    let vertex = first_selectable_vertex(world);
    let active = world
        .state()
        .ucs
        .active
        .expect("expected active UCS placement");
    let triad = active
        .preview_triad
        .expect("expected displayed UCS triad at selected vertex");
    for axis in 0..3 {
        assert!(
            (triad.origin_world[axis] - vertex[axis]).abs() < 1e-6,
            "triad origin {:?} did not match selected vertex {:?}",
            triad.origin_world,
            vertex
        );
    }
    let origin_px = project_world(world, triad.origin_world);
    let x_px = project_world(world, triad.x_tip_world);
    let y_px = project_world(world, triad.y_tip_world);
    let z_px = project_world(world, triad.z_tip_world);
    assert_ne!(origin_px, x_px, "expected visible X axis extent");
    assert_ne!(origin_px, y_px, "expected visible Y axis extent");
    assert_ne!(origin_px, z_px, "expected visible Z axis extent");
    world.save_capture("ucs-triad-at-vertex");
}

#[then("prompts for the X-axis direction")]
fn prompts_for_x_axis_direction(world: &mut ApplicationWorld) {
    let active = world
        .state()
        .ucs
        .active
        .expect("expected active UCS placement");
    assert_eq!(active.stage, GuiUcsStage::PickXDirection);
    world.expect_status("X-axis direction");
}

#[given("UCS has been started at a solid vertex and is prompting for the X-axis direction")]
fn ucs_started_and_prompting_x(world: &mut ApplicationWorld) {
    start_ucs_at_solid_vertex(world);
    prompts_for_x_axis_direction(world);
}

#[when("the user chooses the displayed X direction")]
fn choose_displayed_x_direction(world: &mut ApplicationWorld) {
    click_displayed_x_direction(world);
}

#[then("the MCAD Software prompts for the Y-axis direction")]
fn prompts_for_y_axis_direction(world: &mut ApplicationWorld) {
    let active = world
        .state()
        .ucs
        .active
        .expect("expected active UCS placement");
    assert_eq!(active.stage, GuiUcsStage::PickYDirection);
    world.expect_status("Y-axis direction");
}

#[given("UCS has been started at a solid vertex and displays its triad")]
fn ucs_started_and_displays_triad(world: &mut ApplicationWorld) {
    start_ucs_at_solid_vertex(world);
    displays_triad_at_selected_location(world);
}

#[when(
    "the user clicks the displayed triad handle and the MCAD Software prompts for the Y-axis direction"
)]
fn click_triad_handle_prompt_y(world: &mut ApplicationWorld) {
    click_displayed_x_direction(world);
    prompts_for_y_axis_direction(world);
}

#[when("the user clicks near the displayed Y direction")]
fn click_near_displayed_y_direction(world: &mut ApplicationWorld) {
    click_displayed_y_direction(world);
}

#[then("the MCAD Software returns to Ready and UCS1 appears in the browser")]
fn returns_ready_and_ucs1_in_browser(world: &mut ApplicationWorld) {
    world.expect_ready();
    let state = world.state();
    assert!(
        state.ucs.active.is_none(),
        "UCS placement should be finished"
    );
    assert_eq!(
        state.ucs.triads.len(),
        1,
        "expected persistent UCS1 triad in viewport"
    );
    let browser = state.browser.expect("expected browser");
    assert!(
        browser
            .features
            .iter()
            .any(|feature| feature.name == "UCS1" && feature.kind == "User Coordinate System"),
        "expected UCS1 feature in browser, saw {:?}",
        browser.features
    );
    let model_text = world::squeezed(&world.window_text("Model"));
    assert!(
        model_text.contains("UCS1"),
        "expected UCS1 in Model browser window, saw {model_text}"
    );
}

#[when("the user expands UCS1")]
fn user_expands_ucs1(world: &mut ApplicationWorld) {
    expand_ucs1(world);
}

#[then("the browser lists its YZ, XZ, and XY planes, X, Y, and Z axes, and center point")]
fn browser_lists_ucs_children(world: &mut ApplicationWorld) {
    let model_text = world::squeezed(&world.window_text("Model"));
    let ucs_pos = model_text
        .find("UCS1")
        .unwrap_or_else(|| panic!("expected UCS1 in Model window, saw {model_text}"));
    let after_ucs = &model_text[ucs_pos..];
    for expected in [
        "YZPlane",
        "XZPlane",
        "XYPlane",
        "XAxis",
        "YAxis",
        "ZAxis",
        "CenterPoint",
    ] {
        assert!(
            after_ucs.contains(expected),
            "expected child work feature {expected:?} under UCS1, saw {after_ucs}"
        );
    }
}

#[given("the user starts UCS at a selectable solid vertex")]
fn user_starts_ucs_at_vertex(world: &mut ApplicationWorld) {
    start_ucs_at_solid_vertex(world);
}

#[given(
    "the user selects the displayed X direction and then the displayed Y direction to create UCS1"
)]
fn select_x_then_y_to_create_ucs1(world: &mut ApplicationWorld) {
    click_displayed_x_direction(world);
    prompts_for_y_axis_direction(world);
    click_displayed_y_direction(world);
    returns_ready_and_ucs1_in_browser(world);
}

#[given("UCS1 is visible and expanded in the browser")]
fn ucs1_visible_and_expanded(world: &mut ApplicationWorld) {
    expand_ucs1(world);
    browser_lists_ucs_children(world);
    let state = world.state();
    assert_eq!(state.ucs.triads.len(), 1, "expected UCS1 triad visible");
    let browser = state.browser.expect("expected browser");
    assert!(
        browser.ucs_features.values().any(|&visible| visible),
        "expected UCS1 visibility true"
    );
}

#[when("the user opens its context menu and clicks Visibility")]
fn open_ucs_menu_and_click_visibility(world: &mut ApplicationWorld) {
    right_click_ucs1(world);
    assert!(
        world.checked("Visibility"),
        "expected Visibility checked before toggling off"
    );
    world.click_text("Visibility");
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        world.state().ucs.triads.is_empty(),
        "expected UCS1 triad hidden after unchecking Visibility"
    );
}

#[when("the user clicks Visibility again")]
fn click_visibility_again(world: &mut ApplicationWorld) {
    world.click_text("Visibility");
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[then("the UCS triad is visible in the part display")]
fn ucs_triad_visible_in_part_display(world: &mut ApplicationWorld) {
    let state = world.state();
    assert_eq!(
        state.ucs.triads.len(),
        1,
        "expected visible UCS triad after re-enabling Visibility"
    );
    let triad = state.ucs.triads[0].clone();
    let origin_px = project_world(world, triad.origin_world);
    let x_px = project_world(world, triad.x_tip_world);
    let y_px = project_world(world, triad.y_tip_world);
    let z_px = project_world(world, triad.z_tip_world);
    assert_ne!(origin_px, x_px, "expected visible X axis");
    assert_ne!(origin_px, y_px, "expected visible Y axis");
    assert_ne!(origin_px, z_px, "expected visible Z axis");
    let browser = state.browser.expect("expected browser");
    assert!(
        browser.ucs_features.values().all(|&visible| visible),
        "expected UCS1 visibility persisted as true"
    );
    let path = world.scratch_folder().join("ucs-visibility-roundtrip.opd");
    solid_extrusion::acceptance::save_and_reopen(world, &path);
    world.pump(Vec::new());
    let reopened = world.state();
    assert_eq!(
        reopened.ucs.triads.len(),
        1,
        "expected UCS1 triad preserved across save/reopen"
    );
}

#[given("UCS1 exists in the browser")]
fn ucs1_exists_in_browser(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser
            .features
            .iter()
            .any(|feature| feature.name == "UCS1" && feature.kind == "User Coordinate System"),
        "expected UCS1 in browser"
    );
}

#[when("the user opens its context menu and chooses Redefine Feature")]
fn open_ucs_menu_and_redefine(world: &mut ApplicationWorld) {
    right_click_ucs1(world);
    world.click_text("Redefine Feature");
    world.pump(Vec::new());
    world.pump(Vec::new());
    let active = world
        .state()
        .ucs
        .active
        .expect("expected active UCS redefinition mode");
    assert!(
        active.redefining.is_some(),
        "expected redefining existing UCS1"
    );
}
