// SPDX-License-Identifier: GPL-2.0-only
//! Desktop BDD bindings for Part Create Revolve scenarios.
use super::*;
use cucumber::{given, then, when};
use oblikovati_gui::{FeatureOutputMode, GuiRevolveExtent, GuiRevolvePreview};
use oblikovati_model::feature::sketch_revolve::SketchRevolveDirection;
use oblikovati_viewport::{ViewportNavigation, ViewportPhysicalKey};

#[path = "booleans.rs"]
mod booleans;
#[path = "edit_and_surface.rs"]
mod edit_and_surface;

fn open_fixture_copy(world: &mut ApplicationWorld, relative: &str) {
    let source = scenario_root().join(relative);
    let copy = world.scratch_folder().join(source.file_name().unwrap());
    std::fs::copy(source, &copy).unwrap();
    let document = world
        .app()
        .owner
        .document_host()
        .open_path(&copy, true)
        .unwrap();
    world.pump(Vec::new());
    world.activate_through_tab(document);
}

fn invoke_revolve(world: &mut ApplicationWorld) {
    world.click_text("Revolve");
    world.pump(Vec::new());
}

fn click_revolve_button(world: &mut ApplicationWorld, label: &str) {
    let pixel =
        super::diagnostics::pointer_pixels(world.run_in_window("Revolution", label).center());
    world.click(pixel);
    world.pump(Vec::new());
}

fn pick_bottom_rectangle_axis(world: &mut ApplicationWorld) {
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.25 },
    )
    .unwrap();
    world.pump(Vec::new());
    let pixel = crate::HeadWindow::model_pixel(&world.app().window.native, [0., 0., 0.]).unwrap();
    world.click(pixel);
    world.pump(Vec::new());
}

fn focus_and_replace_angle_a(world: &mut ApplicationWorld, value: &str) {
    let label = world.run_in_window("Revolution", "AngleA").clone();
    let (origin, size) = super::diagnostics::window_geometry("Revolution").unwrap();
    let field = super::diagnostics::pointer_pixels([origin[0] + size[0] * 0.73, label.center()[1]]);
    world.click(field);
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::A, true);
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::Backspace, false);
    world.pump(Vec::new());
    world.type_text(value);
    world.pump(Vec::new());
}

fn preview_z_bounds(world: &mut ApplicationWorld) -> (f32, f32) {
    world
        .app()
        .window
        .scene_triangles
        .iter()
        .flat_map(|tri| tri.positions.map(|p| p[2]))
        .fold((f32::INFINITY, f32::NEG_INFINITY), |(lo, hi), z| {
            (lo.min(z), hi.max(z))
        })
}

fn preview_y_bounds(world: &mut ApplicationWorld) -> (f32, f32) {
    world
        .app()
        .window
        .scene_triangles
        .iter()
        .flat_map(|tri| tri.positions.map(|p| p[1]))
        .fold((f32::INFINITY, f32::NEG_INFINITY), |(lo, hi), y| {
            (lo.min(y), hi.max(y))
        })
}

#[given("a fresh copy of `fixtures/cad/revolve/revolve-profile.opd` is open and Revolve is active")]
fn open_revolve_profile_active(world: &mut ApplicationWorld) {
    open_fixture_copy(world, "fixtures/cad/revolve/revolve-profile.opd");
    invoke_revolve(world);
    assert!(super::diagnostics::window_geometry("Revolution").is_some());
}

#[given("Revolution has its rectangular profile selected and no axis")]
fn revolve_has_profile_and_no_axis(world: &mut ApplicationWorld) {
    let panel = super::world::squeezed(&world.window_text("Revolution"));
    assert!(panel.contains("1Profile"), "{panel}");
    assert!(panel.contains("SelectAxis"), "{panel}");
    let preview = world.state().revolve_preview.unwrap();
    assert_eq!(preview.result, GuiRevolvePreview::MissingAxis);
    assert!(preview.request.axis.is_none());
}

#[when("the straight rectangle edge is selected")]
fn select_straight_rectangle_edge(world: &mut ApplicationWorld) {
    pick_bottom_rectangle_axis(world);
}

#[then("the Axis selector shows 1 Axis")]
fn axis_selector_shows_one_axis(world: &mut ApplicationWorld) {
    let panel = super::world::squeezed(&world.window_text("Revolution"));
    assert!(panel.contains("1Axis"), "{panel}");
    let preview = world.state().revolve_preview.unwrap();
    assert!(preview.request.axis.is_some());
}

#[then("a full cylindrical preview appears")]
fn full_cylindrical_preview_appears(world: &mut ApplicationWorld) {
    let preview = world.state().revolve_preview.unwrap();
    assert_eq!(preview.result, GuiRevolvePreview::Ready);
    assert_eq!(preview.request.extent, GuiRevolveExtent::Full);
    assert!(!world.app().window.scene_triangles.is_empty());
    let (z_min, z_max) = preview_z_bounds(world);
    assert!(
        (z_min + 0.5).abs() < 0.05 && (z_max - 0.5).abs() < 0.05,
        "expected full cylinder Z in [-0.5, 0.5], got [{z_min}, {z_max}]"
    );
}

#[given("a valid profile and the long rectangle edge at Y=0 are selected with editable Angle A")]
fn profile_and_axis_with_editable_angle_a(world: &mut ApplicationWorld) {
    pick_bottom_rectangle_axis(world);
    click_revolve_button(world, "Angle");
    let preview = world.state().revolve_preview.unwrap();
    assert_eq!(preview.result, GuiRevolvePreview::Ready);
    assert_eq!(preview.request.extent, GuiRevolveExtent::Angle);
    assert!(!world.app().window.scene_triangles.is_empty());
}

#[when("Angle A is replaced with 0")]
fn replace_angle_a_with_zero(world: &mut ApplicationWorld) {
    focus_and_replace_angle_a(world, "0");
}

#[then("a red error marker appears")]
fn red_error_marker_appears(world: &mut ApplicationWorld) {
    let preview = world.state().revolve_preview.unwrap();
    assert!(
        matches!(preview.result, GuiRevolvePreview::Failed(_)),
        "expected failed angle validation, got {:?}",
        preview.result
    );
    let panel = world.window_text("Revolution");
    assert!(panel.contains('!'), "expected error badge '!' in {panel}");
}

#[then("the preceding preview remains visible")]
fn preceding_preview_remains_visible(world: &mut ApplicationWorld) {
    assert!(
        !world.app().window.scene_triangles.is_empty(),
        "expected preceding preview triangles to remain visible"
    );
}

#[given("the profile and axis are selected, Angle A is zero and OK is disabled")]
fn profile_axis_angle_zero_disabled(world: &mut ApplicationWorld) {
    profile_and_axis_with_editable_angle_a(world);
    focus_and_replace_angle_a(world, "0");
    assert!(!world.run_is_enabled("OK"));
}

#[when("the selected value is replaced with 9 then appended with 0")]
fn replace_with_9_then_append_0(world: &mut ApplicationWorld) {
    focus_and_replace_angle_a(world, "9");
    world.type_text("0");
    world.pump(Vec::new());
}

#[then("the field shows 90")]
fn angle_field_shows_90(world: &mut ApplicationWorld) {
    let preview = world.state().revolve_preview.unwrap();
    assert_eq!(preview.request.angle, "90");
    assert_eq!(preview.result, GuiRevolvePreview::Ready);
}

#[then("the preview changes to a partial revolution")]
fn preview_is_partial_revolution(world: &mut ApplicationWorld) {
    assert!(!world.app().window.scene_triangles.is_empty());
    let (z_min, z_max) = preview_z_bounds(world);
    assert!(
        (z_max - z_min) < 0.7,
        "expected 90-degree partial Z span < 0.7 cm, got [{z_min}, {z_max}]"
    );
}

#[given("the profile and axis are selected and a 90 degree preview uses the first direction icon")]
fn profile_axis_90_first_direction(world: &mut ApplicationWorld) {
    profile_and_axis_with_editable_angle_a(world);
    focus_and_replace_angle_a(world, "90");
    let preview = world.state().revolve_preview.unwrap();
    assert_eq!(preview.result, GuiRevolvePreview::Ready);
    assert_eq!(preview.request.direction, SketchRevolveDirection::Positive);
}

#[when("the second direction icon is selected")]
fn select_second_direction_icon(world: &mut ApplicationWorld) {
    click_revolve_button(world, "Flipped");
}

#[then("the preview moves to the opposite side of the sketch")]
fn preview_moves_opposite_side(world: &mut ApplicationWorld) {
    let preview = world.state().revolve_preview.unwrap();
    assert_eq!(preview.request.direction, SketchRevolveDirection::Negative);
    assert_eq!(preview.result, GuiRevolvePreview::Ready);
    let (z_min, z_max) = preview_z_bounds(world);
    assert!(
        z_min < -0.4 && z_max < 0.05,
        "expected flipped preview in negative Z, got [{z_min}, {z_max}]"
    );
}

#[when("the third direction icon is selected")]
fn select_third_direction_icon(world: &mut ApplicationWorld) {
    click_revolve_button(world, "Symmetric");
}

#[then("a different preview placement appears")]
fn symmetric_preview_placement_appears(world: &mut ApplicationWorld) {
    let preview = world.state().revolve_preview.unwrap();
    assert_eq!(preview.request.direction, SketchRevolveDirection::Symmetric);
    assert_eq!(preview.result, GuiRevolvePreview::Ready);
    let (z_min, z_max) = preview_z_bounds(world);
    let (y_min, _) = preview_y_bounds(world);
    assert!(
        z_min < -0.2 && z_max > 0.2 && y_min > -0.05,
        "expected symmetric preview spanning Z symmetrically, got Z=[{z_min}, {z_max}] Y_min={y_min}"
    );
}

#[given("the profile and axis are selected and Angle A is 90 degrees")]
fn profile_axis_angle_90(world: &mut ApplicationWorld) {
    profile_axis_90_first_direction(world);
}

#[when("the fourth direction icon is selected")]
fn select_fourth_direction_icon(world: &mut ApplicationWorld) {
    click_revolve_button(world, "Asymmetric");
}

#[then("Angle B displays 45,00 deg")]
fn angle_b_displays_45(world: &mut ApplicationWorld) {
    let preview = world.state().revolve_preview.unwrap();
    assert_eq!(
        preview.request.direction,
        SketchRevolveDirection::Asymmetric
    );
    assert_eq!(preview.request.second_angle, "45,00 deg");
    assert_eq!(preview.result, GuiRevolvePreview::Ready);
}

#[then("both angle controls are visible")]
fn both_angle_controls_visible(world: &mut ApplicationWorld) {
    let panel = super::world::squeezed(&world.window_text("Revolution"));
    assert!(panel.contains("AngleA"), "{panel}");
    assert!(panel.contains("AngleB"), "{panel}");
}

#[given(
    "the profile and axis are selected and the preferences menu is open over the uncommitted preview"
)]
fn profile_axis_and_preferences_menu_open(world: &mut ApplicationWorld) {
    pick_bottom_rectangle_axis(world);
    assert_eq!(
        world.state().revolve_preview.unwrap().result,
        GuiRevolvePreview::Ready
    );
    click_revolve_button(world, "AdvancedSettings");
    world.expect_text("Hide Presets");
}

#[then("the Revolution panel and preview close")]
fn revolution_panel_and_preview_close(world: &mut ApplicationWorld) {
    assert!(super::diagnostics::window_geometry("Revolution").is_none());
    assert!(world.state().revolve_preview.is_none());
    assert!(world.app().window.scene_triangles.is_empty());
}

#[then("Sketch1 remains without a solid body")]
#[then("Sketch1 remains and the solid disappears")]
fn sketch1_remains_without_solid(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert!(browser.bodies.is_empty());
    assert!(
        browser
            .sketches
            .iter()
            .any(|sketch| sketch.name == "Sketch1" && sketch.visible)
    );
}

#[when("R is pressed")]
fn press_r_shortcut(world: &mut ApplicationWorld) {
    world.pump(vec![oblikovati_viewport::ViewportEvent::Focus(true)]);
    world.key(ViewportPhysicalKey::R, false);
    world.pump(Vec::new());
}

#[then("Revolution reopens with 1 Profile and no selected axis")]
fn revolution_reopens_clean(world: &mut ApplicationWorld) {
    assert!(super::diagnostics::window_geometry("Revolution").is_some());
    revolve_has_profile_and_no_axis(world);
}

#[then("the full 360 degree setting and first direction are shown")]
fn full_360_and_first_direction_shown(world: &mut ApplicationWorld) {
    let preview = world.state().revolve_preview.unwrap();
    assert_eq!(preview.request.extent, GuiRevolveExtent::Full);
    assert_eq!(preview.request.direction, SketchRevolveDirection::Positive);
    let panel = super::world::squeezed(&world.window_text("Revolution"));
    assert!(panel.contains("360deg"), "{panel}");
}

#[given("the rectangular profile and its long edge at Y=0 are selected with the full-turn setting")]
fn profile_and_axis_full_turn(world: &mut ApplicationWorld) {
    pick_bottom_rectangle_axis(world);
    full_cylindrical_preview_appears(world);
}

#[then("Revolution1 and Solid Bodies(1) appear with a cylinder")]
#[then("Revolution1 and the solid return")]
fn revolution1_and_cylinder_appear(world: &mut ApplicationWorld) {
    world.expect_text("Revolution1");
    world.expect_text("Solid Bodies");
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 1);
    assert_eq!(browser.bodies[0].faces, 3);
    assert!(
        browser
            .features
            .iter()
            .any(|f| f.name == "Revolution1" && f.kind == "Revolution")
    );
    assert!(!world.app().window.scene_triangles.is_empty());
}
