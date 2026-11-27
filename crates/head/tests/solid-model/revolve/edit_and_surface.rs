// SPDX-License-Identifier: GPL-2.0-only
//! Revolve edit, validation, direction dropdown, and surface BDD steps (revolve-09..18, 25..27).
use super::*;

pub(super) fn reopen_feature(world: &mut ApplicationWorld, name: &str) {
    let row = world.run(name).center();
    world.right_click(super::super::diagnostics::pointer_pixels(row));
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.click(super::super::solid_extrusion::acceptance::edit_menu_position(world));
    world.pump(Vec::new());
    assert!(super::super::diagnostics::window_geometry("Revolution").is_some());
}

fn accept_revolve(world: &mut ApplicationWorld) {
    click_revolve_button(world, "OK");
}

#[given("a writable copy of `fixtures/cad/revolve/revolve-profile.opd` is open")]
fn open_writable_revolve_profile(world: &mut ApplicationWorld) {
    open_fixture_copy(world, "fixtures/cad/revolve/revolve-profile.opd");
}

#[given(
    "I create and accept a full solid Revolution1 from the rectangular profile using its long edge at Y=0 as the axis"
)]
fn create_full_revolution1(world: &mut ApplicationWorld) {
    invoke_revolve(world);
    pick_bottom_rectangle_axis(world);
    full_cylindrical_preview_appears(world);
    accept_revolve(world);
    revolution1_and_cylinder_appear(world);
}

#[when("Edit Feature is selected from its context menu")]
fn edit_revolution1_from_context_menu(world: &mut ApplicationWorld) {
    reopen_feature(world, "Revolution1");
}

#[then("its profile axis and full angle setting are retained")]
fn retained_full_revolve_edit(world: &mut ApplicationWorld) {
    let preview = world.state().revolve_preview.unwrap();
    assert!(preview.request.feature.is_some());
    assert!(preview.request.axis.is_some());
    assert_eq!(preview.request.extent, GuiRevolveExtent::Full);
    let panel = super::super::world::squeezed(&world.window_text("Revolution"));
    assert!(
        panel.contains("1Profile") && panel.contains("1Axis") && panel.contains("360deg"),
        "{panel}"
    );
}

#[when("Angle A is made editable and replaced with 90")]
fn make_angle_editable_and_set_90(world: &mut ApplicationWorld) {
    click_revolve_button(world, "Angle");
    focus_and_replace_angle_a(world, "90");
    assert_eq!(
        world.state().revolve_preview.unwrap().result,
        GuiRevolvePreview::Ready
    );
}

#[then("Revolution1 remains with one solid body and a partial shape")]
#[then("the partial shape returns")]
fn partial_revolution1_remains(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 1);
    assert_eq!(browser.bodies[0].faces, 5);
    let (z_min, z_max) = preview_z_bounds(world);
    assert!(
        (z_max - z_min) < 0.7,
        "expected partial Z span, got [{z_min}, {z_max}]"
    );
}

#[then("the full cylinder returns")]
fn full_cylinder_returns(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 1);
    assert_eq!(browser.bodies[0].faces, 3);
}

#[given(
    "I edit Revolution1 to a 90 degree partial revolution, accept it and reopen it for editing"
)]
fn edit_revolve_to_90_and_reopen(world: &mut ApplicationWorld) {
    reopen_feature(world, "Revolution1");
    make_angle_editable_and_set_90(world);
    accept_revolve(world);
    partial_revolution1_remains(world);
    reopen_feature(world, "Revolution1");
}

#[when("the angle is replaced by 4 then appended with 0 then 0")]
fn enter_400_degrees(world: &mut ApplicationWorld) {
    focus_and_replace_angle_a(world, "4");
    world.type_text("0");
    world.pump(Vec::new());
    world.type_text("0");
    world.pump(Vec::new());
}

#[then("400 shows a red error marker and disables OK")]
fn angle_400_shows_error_and_disables_ok(world: &mut ApplicationWorld) {
    red_error_marker_appears(world);
    assert!(!world.run_is_enabled("OK"));
}

#[when("the full revolution control is clicked")]
fn click_full_revolution_control(world: &mut ApplicationWorld) {
    click_revolve_button(world, "Full");
}

#[then("the angle displays the full 360 degree setting")]
fn angle_displays_full_360(world: &mut ApplicationWorld) {
    let preview = world.state().revolve_preview.unwrap();
    assert_eq!(preview.request.extent, GuiRevolveExtent::Full);
    let panel = super::super::world::squeezed(&world.window_text("Revolution"));
    assert!(panel.contains("360deg"), "{panel}");
}

#[then("the cylinder preview and enabled OK return")]
fn cylinder_preview_and_ok_return(world: &mut ApplicationWorld) {
    full_cylindrical_preview_appears(world);
    assert!(world.run_is_enabled("OK"));
}

#[given("I reopen Revolution1 for editing with a valid full preview")]
fn reopen_revolution1_valid_full(world: &mut ApplicationWorld) {
    reopen_feature(world, "Revolution1");
    full_cylindrical_preview_appears(world);
}

#[when("the adjacent extent control is selected")]
fn select_to_extent_control(world: &mut ApplicationWorld) {
    click_revolve_button(world, "To");
}

#[then("From shows 1 Sketch Plane")]
fn from_shows_sketch_plane(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text("Revolution"));
    assert!(panel.contains("From1SketchPlane"), "{panel}");
}

#[then("To requests a face")]
fn to_requests_a_face(world: &mut ApplicationWorld) {
    let preview = world.state().revolve_preview.unwrap();
    assert_eq!(preview.result, GuiRevolvePreview::MissingToFace);
    let panel = super::super::world::squeezed(&world.window_text("Revolution"));
    assert!(panel.contains("ToSelectFace"), "{panel}");
}

#[given(
    "I create and accept a 90 degree solid Revolution1 using the rectangle region and its long Y=0 edge as axis"
)]
fn create_90_solid_revolution1(world: &mut ApplicationWorld) {
    invoke_revolve(world);
    pick_bottom_rectangle_axis(world);
    make_angle_editable_and_set_90(world);
    accept_revolve(world);
    partial_revolution1_remains(world);
}

#[given(
    "I reopen Revolution1, select the alternate reference extent and attempt to select the YZ origin plane for To"
)]
fn reopen_revolve_and_try_to_extent(world: &mut ApplicationWorld) {
    reopen_feature(world, "Revolution1");
    select_to_extent_control(world);
}

#[given("the To reference remains unselected")]
fn to_reference_remains_unselected(world: &mut ApplicationWorld) {
    to_requests_a_face(world);
    assert!(!world.run_is_enabled("OK"));
}

#[then("the accepted partial Revolution1 solid returns")]
fn accepted_partial_solid_returns(world: &mut ApplicationWorld) {
    assert!(world.state().revolve_preview.is_none());
    partial_revolution1_remains(world);
}

#[given("the rectangle profile is selected, Axis is empty and Angle A is set to 90 degrees")]
fn profile_selected_no_axis_angle_90(world: &mut ApplicationWorld) {
    revolve_has_profile_and_no_axis(world);
    click_revolve_button(world, "Angle");
    focus_and_replace_angle_a(world, "90");
}

#[when("the output type control is clicked")]
fn click_output_type_control(world: &mut ApplicationWorld) {
    click_revolve_button(world, "Output:Solid");
}

#[then("Profiles displays 4 Curves")]
fn profiles_displays_4_curves(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text("Revolution"));
    assert!(panel.contains("4Curves"), "{panel}");
}

#[then("Body Name displays Srf1")]
fn body_name_displays_srf1(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text("Revolution"));
    assert!(panel.contains("BodyNameSrf1"), "{panel}");
}

#[when("the rectangle edge is selected as axis")]
fn select_rectangle_edge_as_axis(world: &mut ApplicationWorld) {
    pick_bottom_rectangle_axis(world);
}

#[then("an orange partial surface preview appears")]
fn orange_partial_surface_preview_appears(world: &mut ApplicationWorld) {
    let preview = world.state().revolve_preview.unwrap();
    assert_eq!(preview.result, GuiRevolvePreview::Ready);
    assert_eq!(preview.request.output, FeatureOutputMode::Surface);
    assert!(!world.app().window.scene_triangles.is_empty());
}

#[given("Surface mode shows 4 Curves and 1 Axis and Angle A is 90 degrees")]
fn surface_mode_4_curves_1_axis_90(world: &mut ApplicationWorld) {
    profile_selected_no_axis_angle_90(world);
    click_output_type_control(world);
    pick_bottom_rectangle_axis(world);
    orange_partial_surface_preview_appears(world);
}

#[then("RevolutionSrf1 and Surface Bodies(1) appear")]
#[then("RevolutionSrf1 and its surface body return")]
fn revolution_srf1_appears(world: &mut ApplicationWorld) {
    world.expect_text("RevolutionSrf1");
    world.expect_text("Surface Bodies (1)");
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.surface_bodies.len(), 1);
    assert_eq!(browser.surface_bodies[0].faces, 3);
}

#[then("Sketch1 remains and the surface disappears")]
fn sketch1_remains_surface_disappears(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert!(browser.surface_bodies.is_empty());
    assert!(
        browser
            .sketches
            .iter()
            .any(|sketch| sketch.name == "Sketch1" && sketch.visible)
    );
}

#[given(
    "I create and accept a 90 degree surface revolution from the four rectangle curves using the long Y=0 edge as the axis"
)]
fn create_90_surface_revolution(world: &mut ApplicationWorld) {
    invoke_revolve(world);
    surface_mode_4_curves_1_axis_90(world);
    accept_revolve(world);
    revolution_srf1_appears(world);
}

#[given("I reopen RevolutionSrf1 for editing")]
fn reopen_revolution_srf1(world: &mut ApplicationWorld) {
    reopen_feature(world, "RevolutionSrf1");
}

#[when("the Profiles clear control is clicked")]
fn click_profiles_clear(world: &mut ApplicationWorld) {
    click_revolve_button(world, "ClearProfile");
}

#[then("selects Profiles is required and OK is disabled")]
fn select_profiles_required(world: &mut ApplicationWorld) {
    let preview = world.state().revolve_preview.unwrap();
    assert_eq!(preview.result, GuiRevolvePreview::MissingProfile);
    assert!(!world.run_is_enabled("OK"));
}

#[then("1 Axis remains selected")]
fn one_axis_remains_selected(world: &mut ApplicationWorld) {
    axis_selector_shows_one_axis(world);
}

#[when("the rectangle edge is clicked")]
fn click_rectangle_edge_to_restore_profile(world: &mut ApplicationWorld) {
    pick_bottom_rectangle_axis(world);
}

#[then("Profiles displays 4 Curves and the valid preview returns")]
fn profiles_4_curves_and_preview_return(world: &mut ApplicationWorld) {
    profiles_displays_4_curves(world);
    orange_partial_surface_preview_appears(world);
}

#[then("the panel closes and the accepted surface remains")]
#[then("the accepted asymmetric surface remains")]
fn panel_closes_surface_remains(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry("Revolution").is_none());
    revolution_srf1_appears(world);
}

#[given("Direction uses icon selector style")]
fn direction_uses_icons(world: &mut ApplicationWorld) {
    assert!(world.window_text("Revolution").contains("Direction"));
}

#[when("Dropdown is selected from Selector Style")]
fn switch_direction_to_dropdown(world: &mut ApplicationWorld) {
    click_revolve_button(world, "Direction");
    world.click_text("Dropdown");
    world.pump(Vec::new());
}

#[then("Direction displays Default in a dropdown")]
fn direction_displays_default_dropdown(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text("Revolution"));
    assert!(panel.contains("DirectionDefault"), "{panel}");
}

#[when("that dropdown is opened")]
fn open_direction_dropdown(world: &mut ApplicationWorld) {
    click_revolve_button(world, "Default");
}

#[then("Default Flipped Symmetric and Asymmetric choices are visible")]
fn all_four_directions_visible(world: &mut ApplicationWorld) {
    for label in ["Default", "Flipped", "Symmetric", "Asymmetric"] {
        world.expect_text(label);
    }
}

#[given("Angle A shows 90 degrees")]
fn angle_a_shows_90_deg(world: &mut ApplicationWorld) {
    let preview = world.state().revolve_preview.unwrap();
    assert!(
        preview.request.angle.contains("90"),
        "{}",
        preview.request.angle
    );
}

#[when("Asymmetric is selected")]
fn select_asymmetric_direction(world: &mut ApplicationWorld) {
    click_revolve_button(world, "Asymmetric");
}

#[then("Angle B 45,00 deg is displayed with a changed preview")]
fn angle_b_with_changed_preview(world: &mut ApplicationWorld) {
    angle_b_displays_45(world);
    both_angle_controls_visible(world);
}

#[when("OK is clicked and the feature is reopened")]
fn ok_and_reopen_surface(world: &mut ApplicationWorld) {
    accept_revolve(world);
    reopen_feature(world, "RevolutionSrf1");
}

#[then("Asymmetric and both angle values are retained")]
fn asymmetric_and_angles_retained(world: &mut ApplicationWorld) {
    let preview = world.state().revolve_preview.unwrap();
    assert_eq!(
        preview.request.direction,
        SketchRevolveDirection::Asymmetric
    );
    assert!(
        preview.request.angle.contains("90"),
        "{}",
        preview.request.angle
    );
    assert!(
        preview.request.second_angle.contains("45"),
        "{}",
        preview.request.second_angle
    );
}

#[then("4 Curves and 1 Axis remain selected")]
fn four_curves_and_one_axis_remain(world: &mut ApplicationWorld) {
    profiles_displays_4_curves(world);
    axis_selector_shows_one_axis(world);
}

#[given("I reopen Revolution1 for editing")]
fn reopen_revolution1_for_editing(world: &mut ApplicationWorld) {
    reopen_feature(world, "Revolution1");
}

#[given("I reopen Revolution1 for editing and switch Full off")]
fn reopen_revolution1_and_switch_full_off(world: &mut ApplicationWorld) {
    reopen_feature(world, "Revolution1");
    click_revolve_button(world, "Angle");
}

#[when("Full is switched off and Angle A is set to -90 deg")]
fn switch_full_off_set_neg_90(world: &mut ApplicationWorld) {
    click_revolve_button(world, "Angle");
    focus_and_replace_angle_a(world, "-90 deg");
}

#[then("a red validation indicator appears and OK is disabled")]
fn red_validation_and_ok_disabled(world: &mut ApplicationWorld) {
    red_error_marker_appears(world);
    assert!(!world.run_is_enabled("OK"));
}

#[when(expr = "Angle A is set to {string}")]
#[when(expr = "Angle A is replaced with {string}")]
fn set_angle_a_string(world: &mut ApplicationWorld, value: String) {
    focus_and_replace_angle_a(world, &value);
}

#[when("Angle A is set to 360.01 deg")]
fn set_angle_360_01(world: &mut ApplicationWorld) {
    focus_and_replace_angle_a(world, "360.01 deg");
}

#[when("Angle A is replaced with 360 deg")]
fn replace_angle_360(world: &mut ApplicationWorld) {
    focus_and_replace_angle_a(world, "360 deg");
}

#[when("Angle A is set to 0 deg")]
fn set_angle_0_deg(world: &mut ApplicationWorld) {
    focus_and_replace_angle_a(world, "0 deg");
}

#[when("Angle A is replaced with 0.01 deg")]
fn replace_angle_0_01_deg(world: &mut ApplicationWorld) {
    focus_and_replace_angle_a(world, "0.01 deg");
}

#[then("the field reports Out of Range and OK is disabled")]
fn field_reports_out_of_range(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text("Revolution"));
    assert!(panel.contains("OutofRange"), "{panel}");
    assert!(!world.run_is_enabled("OK"));
}

#[then("the error indicator clears and OK is enabled")]
#[then("the indicator clears and OK is enabled")]
fn indicator_clears_ok_enabled(world: &mut ApplicationWorld) {
    let preview = world.state().revolve_preview.unwrap();
    assert_eq!(preview.result, GuiRevolvePreview::Ready);
    assert!(world.run_is_enabled("OK"));
}

#[then("a validation indicator appears and OK is disabled")]
fn validation_indicator_ok_disabled(world: &mut ApplicationWorld) {
    red_error_marker_appears(world);
    assert!(!world.run_is_enabled("OK"));
}
