// SPDX-License-Identifier: GPL-2.0-only
//! Desktop BDD bindings for Part Create Decal scenarios (decal-01..decal-04).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiDecalPreview;
use setup::{
    DECAL_WINDOW, IMAGE_TIP_WINDOW, click_decal_control, click_decal_ribbon,
    extrude_trapezoid_20mm, insert_decal_image, open_disposable_decal_fixture, select_sloped_face,
};

#[given(expr = "a copy of `{word}` is open")]
fn copy_of_fixture_is_open(world: &mut ApplicationWorld, relative: String) {
    open_disposable_decal_fixture(world, &relative);
}

#[given(expr = "a disposable copy of the native fixture `{word}` is open")]
fn disposable_backtick_native_fixture_is_open(world: &mut ApplicationWorld, relative: String) {
    open_disposable_decal_fixture(world, &relative);
}

#[given("no visible 2D sketch with an inserted image exists in the part")]
fn no_visible_image_sketch(world: &mut ApplicationWorld) {
    assert!(
        world.state().decal_images.is_empty(),
        "expected no visible 2D image sketch, saw {:?}",
        world.state().decal_images
    );
}

#[when("Decal is opened")]
fn decal_is_opened(world: &mut ApplicationWorld) {
    click_decal_ribbon(world);
}

#[then("the MCAD Software shows a tip requesting a visible 2D sketch with an inserted image.")]
fn shows_image_sketch_tip(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(IMAGE_TIP_WINDOW).is_some(),
        "expected Decal image-sketch-required tip modal, saw {}",
        world.text()
    );
    let text = super::super::world::squeezed(&world.window_text(IMAGE_TIP_WINDOW));
    assert!(
        text.contains("visible2Dsketch") && text.contains("insertedimage"),
        "expected visible 2D image sketch tip text, saw {text}"
    );
    assert!(
        super::super::diagnostics::window_geometry(DECAL_WINDOW).is_none(),
        "expected Decal panel to remain closed without an image sketch"
    );
}

#[given("the user extrudes the closed trapezoid 20 mm in the positive Z direction")]
fn user_extrudes_trapezoid(world: &mut ApplicationWorld) {
    extrude_trapezoid_20mm(world);
}

#[given(
    expr = "the user inserts `{word}` into the visible sketch on the sloped-face plane at \\(5, 5\\) mm with 10 mm width and height"
)]
fn user_inserts_decal_image(world: &mut ApplicationWorld, relative: String) {
    insert_decal_image(world, &relative);
    assert!(
        !world.state().decal_images.is_empty(),
        "expected inserted image in visible sketch"
    );
}

#[given("Decal is open")]
fn decal_is_open_given(world: &mut ApplicationWorld) {
    click_decal_ribbon(world);
    assert!(
        super::super::diagnostics::window_geometry(DECAL_WINDOW).is_some(),
        "expected Decal dialog to open, saw {}",
        world.text()
    );
}

#[then("the Image and Face selectors are empty and OK is disabled.")]
fn image_face_empty_ok_disabled(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(DECAL_WINDOW));
    assert!(
        text.contains("SelectImage"),
        "expected empty Image selector, saw {text}"
    );
    assert!(
        text.contains("SelectFace"),
        "expected empty Face selector, saw {text}"
    );
    assert!(
        world.run_in_window(DECAL_WINDOW, "OK").alpha < 200,
        "expected OK disabled when Image and Face are empty"
    );
}

#[then("Automatic Face Chain is checked and Wrap to Face is unchecked.")]
fn chain_checked_wrap_unchecked(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .decal_preview
        .expect("expected decal preview state");
    assert!(
        preview.request.chain_faces,
        "expected Automatic Face Chain checked"
    );
    assert!(
        !preview.request.wrap_to_face,
        "expected Wrap to Face unchecked"
    );
}

#[when("the user selects the inserted image")]
fn user_selects_inserted_image(world: &mut ApplicationWorld) {
    click_decal_control(world, "SelectImage");
}

#[then("the Image selector shows 1 Image.")]
fn image_selector_shows_1_image(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(DECAL_WINDOW));
    assert!(
        text.contains("1Image"),
        "expected 1 Image in Decal panel, saw {text}"
    );
}

#[when("the user selects the sloped face")]
fn user_selects_sloped_face(world: &mut ApplicationWorld) {
    select_sloped_face(world);
}

#[then("Face shows 1 Face and OK becomes enabled.")]
fn face_shows_1_face_ok_enabled(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(DECAL_WINDOW));
    assert!(
        text.contains("1Face"),
        "expected 1 Face in Decal panel, saw {text}"
    );
    let preview = world
        .state()
        .decal_preview
        .expect("expected decal preview state");
    assert_eq!(preview.result, GuiDecalPreview::Ready);
    assert!(
        world.run_in_window(DECAL_WINDOW, "OK").alpha >= 200,
        "expected OK enabled once Image and Face are selected"
    );
}

#[then("Wrap to Face is disabled for the selected face.")]
fn wrap_to_face_disabled(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .decal_preview
        .expect("expected decal preview state");
    assert!(
        !preview.wrap_to_face_available,
        "expected Wrap to Face unavailable for planar sloped face"
    );
    assert!(
        world.run_in_window(DECAL_WINDOW, "WraptoFace").alpha < 200,
        "expected Wrap to Face checkbox disabled for planar face"
    );
}

#[given("Decal is open with the fixture image and sloped face selected")]
fn decal_open_with_image_and_face(world: &mut ApplicationWorld) {
    decal_is_open_given(world);
    click_decal_control(world, "SelectImage");
    select_sloped_face(world);
    face_shows_1_face_ok_enabled(world);
}

#[given("Automatic Face Chain is enabled")]
fn automatic_face_chain_enabled(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .decal_preview
        .expect("expected decal preview state");
    assert!(preview.request.chain_faces);
}

#[then("the properties close and Decal1 appears in the browser.")]
fn properties_close_and_decal1_appears(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(DECAL_WINDOW).is_none(),
        "expected Decal dialog to close after OK"
    );
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser
            .features
            .iter()
            .any(|feature| feature.name == "Decal1" && feature.kind == "Decal"),
        "expected Decal1 in browser, saw {:?}",
        browser.features
    );
}

#[given("the user creates Decal1 from the inserted image and the sloped face")]
fn user_creates_decal1(world: &mut ApplicationWorld) {
    decal_open_with_image_and_face(world);
    click_decal_control(world, "OK");
    properties_close_and_decal1_appears(world);
}

#[when("the user double-clicks Decal1")]
fn user_double_clicks_decal1(world: &mut ApplicationWorld) {
    let center = world.run_in_window("Model", "Decal1").center();
    world.double_click(super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::diagnostics::window_geometry(DECAL_WINDOW).is_some(),
        "expected Decal dialog to reopen on double-click of Decal1"
    );
}

#[then("the properties show 1 Image and 1 Face with Automatic Face Chain checked.")]
fn properties_show_saved_decal_state(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(DECAL_WINDOW));
    assert!(text.contains("1Image"), "expected 1 Image, saw {text}");
    assert!(text.contains("1Face"), "expected 1 Face, saw {text}");
    let preview = world
        .state()
        .decal_preview
        .expect("expected decal preview state");
    assert!(preview.request.chain_faces);
}

#[when("the user clears Automatic Face Chain and clicks Cancel")]
fn user_clears_chain_and_cancels(world: &mut ApplicationWorld) {
    click_decal_control(world, "AutomaticFaceChain");
    let preview = world
        .state()
        .decal_preview
        .expect("expected decal preview state");
    assert!(
        !preview.request.chain_faces,
        "expected Automatic Face Chain unchecked before Cancel"
    );
    let text = super::super::world::squeezed(&world.window_text(DECAL_WINDOW));
    assert!(text.contains("1Image") && text.contains("1Face"), "{text}");
    click_decal_control(world, "Cancel");
}

#[then("the properties close.")]
fn decal_properties_close(_world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(DECAL_WINDOW).is_none()
            && super::super::diagnostics::window_geometry("Unwrap").is_none(),
        "expected feature properties panel to close"
    );
}
