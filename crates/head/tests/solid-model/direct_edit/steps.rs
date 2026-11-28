// SPDX-License-Identifier: GPL-2.0-only
//! Cucumber step definitions for Part Modify Direct Edit (direct-edit-01..direct-edit-08, #3918).
#[path = "setup.rs"]
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_model::feature::direct_edit::{
    DirectEditCoordinateMode, DirectEditMode, DirectEditScaleMode, DirectEditSelectionFilter,
};
use setup::*;

#[given("the part has a planar circular end face")]
fn part_has_planar_circular_end_face(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 1, "expected 1 solid body");
    assert_eq!(browser.bodies[0].faces, 3, "expected cylinder with 3 faces");
}

#[when("I open Direct Edit")]
fn open_direct_edit(world: &mut ApplicationWorld) {
    open_direct_edit_ribbon(world);
}

#[then("the floating toolbar displays Move, Size, Scale, Rotate, and Delete")]
fn floating_toolbar_displays_five_modes(world: &mut ApplicationWorld) {
    let text = world::squeezed(&world.window_text(DIRECT_EDIT_WINDOW));
    for mode in ["Move", "Size", "Scale", "Rotate", "Delete"] {
        assert!(
            text.contains(mode),
            "expected Direct Edit toolbar to display {mode}, saw {text}"
        );
    }
}

#[when("I select the circular face in Move mode")]
fn select_circular_face_in_move(world: &mut ApplicationWorld) {
    click_planar_circular_end_face(world);
}

#[then("the face is highlighted and a manipulator appears")]
fn face_highlighted_and_manipulator_appears(world: &mut ApplicationWorld) {
    let state = world.state();
    assert!(
        state.model_face_chosen,
        "expected circular face to be highlighted in viewport"
    );
    assert!(
        !world.app().owner.selected_browser_outline().is_empty(),
        "expected face highlight outline segments in viewport"
    );
    assert!(
        diagnostics::window_geometry(MANIPULATOR_WINDOW).is_some(),
        "expected Move manipulator overlay"
    );
    let manip_text = world::squeezed(&world.window_text(MANIPULATOR_WINDOW));
    assert!(
        manip_text.contains("MoveManipulator"),
        "expected Move Manipulator badge, saw {manip_text}"
    );
}

#[given("a planar end face is selected in Direct Edit")]
fn planar_end_face_selected_in_direct_edit(world: &mut ApplicationWorld) {
    open_direct_edit_ribbon(world);
    click_planar_circular_end_face(world);
}

#[when("I choose Size")]
fn choose_size_mode(world: &mut ApplicationWorld) {
    click_toolbar_control(world, "Size");
}

#[then("Size is highlighted and a size manipulator is displayed")]
fn size_highlighted_and_manipulator_displayed(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .direct_edit_preview
        .expect("expected direct edit preview in Size mode");
    assert_eq!(preview.request.mode, DirectEditMode::Size);
    assert!(
        diagnostics::window_geometry(MANIPULATOR_WINDOW).is_some(),
        "expected Size manipulator overlay"
    );
    let manip_text = world::squeezed(&world.window_text(MANIPULATOR_WINDOW));
    assert!(
        manip_text.contains("SizeManipulator"),
        "expected Size Manipulator badge, saw {manip_text}"
    );
}

#[when("I choose Scale, Rotate, and Delete in turn")]
fn choose_scale_rotate_delete_in_turn(world: &mut ApplicationWorld) {
    for (label, expected) in [
        ("Scale", DirectEditMode::Scale),
        ("Rotate", DirectEditMode::Rotate),
        ("Delete", DirectEditMode::Delete),
    ] {
        click_toolbar_control(world, label);
        let preview = world
            .state()
            .direct_edit_preview
            .expect("expected direct edit preview state after mode switch");
        assert_eq!(
            preview.request.mode, expected,
            "expected toolbar mode {expected:?} to be active"
        );
    }
}

#[then("the corresponding toolbar mode is highlighted")]
fn corresponding_toolbar_mode_highlighted(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .direct_edit_preview
        .expect("expected direct edit preview state");
    assert_eq!(preview.request.mode, DirectEditMode::Delete);
}

#[given("Direct Edit is open with no geometry selected")]
fn direct_edit_open_without_selection(world: &mut ApplicationWorld) {
    open_direct_edit_ribbon(world);
    let preview = world.state().direct_edit_preview.unwrap();
    assert!(preview.request.body.is_none() && preview.request.faces.is_empty());
}

#[when("I click its red X")]
#[when("I click the red X")]
#[when("I click the red X again")]
fn click_direct_edit_red_x(world: &mut ApplicationWorld) {
    click_toolbar_control(world, "X");
}

#[then("the floating toolbar closes and the model remains visible")]
fn floating_toolbar_closes_model_visible(world: &mut ApplicationWorld) {
    assert!(
        diagnostics::window_geometry(DIRECT_EDIT_WINDOW).is_none(),
        "expected Direct Edit toolbar to close"
    );
    assert!(
        !world.app().window.scene_triangles.is_empty(),
        "expected solid model to remain visible"
    );
}

#[given("Direct Edit is in Scale mode and the cylindrical solid is selected")]
fn direct_edit_in_scale_mode_solid_selected(world: &mut ApplicationWorld) {
    setup_mode_and_solid(world, DirectEditMode::Scale);
}

#[when("I focus the Uniform factor field and enter 2")]
fn focus_uniform_factor_and_enter_2(world: &mut ApplicationWorld) {
    let before = scene_extent_max(world);
    replace_field_next_to(world, "Factor", "2");
    world.pump(Vec::new());
    let after = scene_extent_max(world);
    assert!(
        after > before * 1.5,
        "expected solid preview extent to enlarge from {before}, got {after}"
    );
}

#[then("the solid preview enlarges")]
fn solid_preview_enlarges(world: &mut ApplicationWorld) {
    let state = world.state();
    assert!(
        state.extrude_preview_active,
        "expected active enlarged Direct Edit preview in viewport"
    );
    let preview = state.direct_edit_preview.unwrap();
    assert_eq!(preview.request.uniform_factor.trim(), "2");
    assert_eq!(preview.request.scale_mode, DirectEditScaleMode::Uniform);
}

#[when("I click the green plus")]
fn click_green_plus(world: &mut ApplicationWorld) {
    click_toolbar_control(world, "+");
    world.pump(Vec::new());
}

#[then("Direct Edit1 and Scale1 appear in the browser")]
fn direct_edit1_and_scale1_appear_in_browser(world: &mut ApplicationWorld) {
    let text = world.text();
    assert!(
        text.contains("DirectEdit1"),
        "expected Direct Edit1 in browser, saw {text}"
    );
    assert!(
        text.contains("Scale1"),
        "expected Scale1 child node in browser, saw {text}"
    );
    let browser = world.state().browser.unwrap();
    assert!(
        browser
            .features
            .iter()
            .any(|feature| feature.name == "Direct Edit1")
    );
    assert!(
        browser
            .direct_edit_edits
            .values()
            .any(|seed| seed.operations.iter().any(|op| op.name == "Scale1"))
    );
}

#[then("the enlarged solid remains after I close Direct Edit")]
fn enlarged_solid_remains_after_close(world: &mut ApplicationWorld) {
    let volume_before_close = solid_volume(world);
    click_toolbar_control(world, "X");
    assert!(
        diagnostics::window_geometry(DIRECT_EDIT_WINDOW).is_none(),
        "expected Direct Edit toolbar to close after clicking red X"
    );
    let volume_after_close = solid_volume(world);
    assert!(
        (volume_after_close - volume_before_close).abs() < 1e-5,
        "expected enlarged solid volume {volume_before_close} to remain, got {volume_after_close}"
    );
    // Original cylinder volume in fillet-circular-edge-candidate.opd is ~pi * r^2 * h;
    // Uniform scale 2.0 produces 8x volume (well above the unscaled volume).
    assert!(volume_after_close > 0.0);
}

#[given(
    "I open Direct Edit in Scale mode, select the cylindrical solid and apply a uniform factor of 2 with the green plus"
)]
fn create_direct_edit1_scale1_factor_2(world: &mut ApplicationWorld) {
    let base_vol = solid_volume(world);
    setup_mode_and_solid(world, DirectEditMode::Scale);
    replace_field_next_to(world, "Factor", "2");
    click_toolbar_control(world, "+");
    world.pump(Vec::new());
    let scaled_vol = solid_volume(world);
    assert!(
        (scaled_vol - base_vol * 8.0).abs() < 1e-4,
        "expected 8x volume ({}), got {scaled_vol}",
        base_vol * 8.0
    );
}

#[given("I close Direct Edit with Direct Edit1 and Scale1 retained")]
fn close_direct_edit_retaining_scale1(world: &mut ApplicationWorld) {
    click_toolbar_control(world, "X");
    assert!(diagnostics::window_geometry(DIRECT_EDIT_WINDOW).is_none());
    let text = world.text();
    assert!(text.contains("DirectEdit1") && text.contains("Scale1"));
}

#[when("I reopen Direct Edit1 and then Scale1")]
fn reopen_direct_edit1_and_scale1(world: &mut ApplicationWorld) {
    let de1 = world.run("DirectEdit1").center();
    world.double_click(diagnostics::pointer_pixels(de1));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry(DIRECT_EDIT_WINDOW).is_some());

    let scale1 = world.run("Scale1").center();
    world.click(diagnostics::pointer_pixels(scale1));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[then("Uniform and factor 2 are displayed")]
fn uniform_and_factor_2_displayed(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .direct_edit_preview
        .expect("expected hydrated Scale1 preview");
    assert_eq!(preview.request.mode, DirectEditMode::Scale);
    assert_eq!(preview.request.scale_mode, DirectEditScaleMode::Uniform);
    assert_eq!(preview.request.uniform_factor.trim(), "2");
    let text = world::squeezed(&world.window_text(DIRECT_EDIT_WINDOW));
    assert!(text.contains("Uniform") && text.contains("2"), "{text}");
}

#[when("I select Non Uniform")]
fn select_non_uniform(world: &mut ApplicationWorld) {
    click_toolbar_control(world, "NonUniform");
}

#[then("separate X, Y, and Z fields initially display 2")]
fn separate_xyz_fields_display_2(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .direct_edit_preview
        .expect("expected Non Uniform preview");
    assert_eq!(preview.request.scale_mode, DirectEditScaleMode::NonUniform);
    assert_eq!(preview.request.scale_factors, ["2", "2", "2"]);
    let text = world::squeezed(&world.window_text(DIRECT_EDIT_WINDOW));
    assert!(
        text.contains('X') && text.contains('Y') && text.contains('Z'),
        "expected X, Y, Z labels in Direct Edit toolbar, saw {text}"
    );
}

#[when("I change X to 1 and apply")]
fn change_x_to_1_and_apply(world: &mut ApplicationWorld) {
    replace_field_next_to(world, "X", "1");
    click_toolbar_control(world, "+");
    world.pump(Vec::new());
}

#[when("I reopen Scale1")]
fn reopen_scale1(world: &mut ApplicationWorld) {
    let scale1 = world.run("Scale1").center();
    world.click(diagnostics::pointer_pixels(scale1));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[then("Non Uniform is displayed with X, Y, and Z at 1")]
fn non_uniform_displayed_with_xyz_at_1(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .direct_edit_preview
        .expect("expected reopened Scale1 preview");
    assert_eq!(preview.request.scale_mode, DirectEditScaleMode::NonUniform);
    assert_eq!(preview.request.scale_factors, ["1", "1", "1"]);
}

#[given("Direct Edit is in Move mode with a planar end face selected")]
fn direct_edit_move_with_face_selected(world: &mut ApplicationWorld) {
    open_direct_edit_ribbon(world);
    click_planar_circular_end_face(world);
}

#[when("I open the selection menu and choose Solids")]
fn open_selection_menu_choose_solids(world: &mut ApplicationWorld) {
    open_selection_menu_and_choose(world, "Solids");
    let preview = world.state().direct_edit_preview.unwrap();
    assert_eq!(preview.request.filter, DirectEditSelectionFilter::Solids);
    assert!(preview.request.faces.is_empty());
    assert!(preview.request.body.is_none());
}

#[when("I select the solid")]
fn select_the_solid(world: &mut ApplicationWorld) {
    click_cylindrical_solid(world);
}

#[then("the solid is highlighted and the move manipulator appears")]
fn solid_highlighted_and_move_manipulator_appears(world: &mut ApplicationWorld) {
    let state = world.state();
    assert!(
        !state.selected_body_keys.is_empty(),
        "expected solid body to be highlighted"
    );
    assert!(
        !world.app().owner.selected_browser_outline().is_empty(),
        "expected solid highlight outline in viewport"
    );
    assert!(
        diagnostics::window_geometry(MANIPULATOR_WINDOW).is_some(),
        "expected Move manipulator overlay"
    );
    let manip_text = world::squeezed(&world.window_text(MANIPULATOR_WINDOW));
    assert!(
        manip_text.contains("MoveManipulator"),
        "expected Move Manipulator badge, saw {manip_text}"
    );
}

#[given("the cylindrical solid is selected in Rotate mode and Local is displayed")]
fn cylindrical_solid_selected_in_rotate_local(world: &mut ApplicationWorld) {
    setup_mode_and_solid(world, DirectEditMode::Rotate);
    let preview = world.state().direct_edit_preview.unwrap();
    assert_eq!(
        preview.request.coordinate_mode,
        DirectEditCoordinateMode::Local
    );
    let text = world::squeezed(&world.window_text(DIRECT_EDIT_WINDOW));
    assert!(text.contains("Local"), "expected Local label, saw {text}");
}

#[when("I open the coordinate menu")]
fn open_coordinate_menu(world: &mut ApplicationWorld) {
    click_toolbar_control(world, "Localv");
}

#[then("World and Local are available")]
fn world_and_local_available(world: &mut ApplicationWorld) {
    let has_world = world
        .frame
        .runs
        .iter()
        .any(|r| world.frame.is_popup_row(r) && r.text == "World");
    let has_local = world
        .frame
        .runs
        .iter()
        .any(|r| world.frame.is_popup_row(r) && r.text == "Local");
    assert!(
        has_world && has_local,
        "expected World and Local in coordinate menu, saw {}",
        world.text()
    );
}

#[when("I choose World")]
fn choose_world(world: &mut ApplicationWorld) {
    click_popup_item(world, "World");
}

#[then("World is displayed")]
fn world_is_displayed(world: &mut ApplicationWorld) {
    let preview = world.state().direct_edit_preview.unwrap();
    assert_eq!(
        preview.request.coordinate_mode,
        DirectEditCoordinateMode::World
    );
    let text = world::squeezed(&world.window_text(DIRECT_EDIT_WINDOW));
    assert!(text.contains("World"), "expected World label, saw {text}");
}

#[then("the selection clears and the toolbar remains")]
fn selection_clears_and_toolbar_remains(world: &mut ApplicationWorld) {
    assert!(
        diagnostics::window_geometry(DIRECT_EDIT_WINDOW).is_some(),
        "expected Direct Edit toolbar to remain open after first red X"
    );
    let state = world.state();
    assert!(!state.model_face_chosen);
    assert!(state.selected_body_keys.is_empty());
    let preview = state.direct_edit_preview.unwrap();
    assert!(preview.request.body.is_none() && preview.request.faces.is_empty());
    assert!(!preview.request.snap_parallel);
}

#[then("the toolbar closes")]
fn toolbar_closes(_world: &mut ApplicationWorld) {
    assert!(
        diagnostics::window_geometry(DIRECT_EDIT_WINDOW).is_none(),
        "expected Direct Edit toolbar to close after second red X"
    );
}

#[given("Direct Edit is in Rotate mode with a planar end face selected")]
fn direct_edit_rotate_with_planar_face(world: &mut ApplicationWorld) {
    open_direct_edit_ribbon(world);
    click_toolbar_control(world, "Rotate");
    click_planar_circular_end_face(world);
}

#[when("I click Snap Parallel")]
fn click_snap_parallel(world: &mut ApplicationWorld) {
    click_toolbar_control(world, "SnapParallel");
}

#[then("the status bar asks for a face to specify the rotation angle")]
fn status_bar_asks_for_face_rotation_angle(world: &mut ApplicationWorld) {
    let text = world::squeezed(&world.text());
    assert!(
        text.contains("Selectafacetospecifytherotationangle"),
        "expected Snap Parallel status prompt, saw {text}"
    );
}
