// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Part Surface Replace Face scenarios (replace-face-01..04, #3918).
mod setup;
mod viewport;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiReplaceFacePreview;
use setup::{
    FAILURE_MODAL_WINDOW, REPLACE_FACE_WINDOW, click_modal_control, click_replace_face_control,
    create_offset_surface_1mm_on_block_top, create_offset_surface_1mm_on_exposed_planar_face,
    existing_count, new_count, open_replace_face_ribbon, revolve_sketch9_full_turn_torus,
    select_block_top_and_offset_surface, select_torus_and_offset_surface,
};

#[given(expr = "a disposable copy of the native fixture {string} is open in 3D Model")]
fn open_fixture_in_3d_model(world: &mut ApplicationWorld, relative: String) {
    let source = super::super::scenario_root().join(&relative);
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
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
}

#[when("Replace Face is opened")]
#[given("Replace Face is open")]
#[given("Replace Face is open without selections")]
fn replace_face_is_opened(world: &mut ApplicationWorld) {
    open_replace_face_ribbon(world);
    assert_eq!(existing_count(world), 0);
    assert_eq!(new_count(world), 0);
}

#[then("Existing Faces and New Faces selectors and Automatic Face Chain are displayed")]
fn replace_face_selectors_and_chain_displayed(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(REPLACE_FACE_WINDOW));
    for expected in ["ExistingFaces", "NewFaces", "AutomaticFaceChain"] {
        assert!(
            text.contains(expected),
            "expected {expected} in Replace Face panel, saw {text}"
        );
    }
}

#[given("the user creates OffsetSrf1 by offsetting the top planar face outward by 1 mm")]
fn user_creates_offsetsrf1_1mm_outward(world: &mut ApplicationWorld) {
    create_offset_surface_1mm_on_block_top(world);
}

#[when("the original top face and the offset surface face are selected")]
fn select_original_top_and_offset_surface(world: &mut ApplicationWorld) {
    select_block_top_and_offset_surface(world);
}

#[then("OK becomes available.")]
fn ok_becomes_available(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .replace_face_preview
        .expect("expected Replace Face preview");
    assert_eq!(
        preview.result,
        GuiReplaceFacePreview::Ready,
        "expected Ready preview after selecting original top face and offset surface, got {:?}",
        preview.result
    );
    assert!(
        world.run_in_window(REPLACE_FACE_WINDOW, "OK").alpha >= 200,
        "expected OK enabled"
    );
}

#[given(
    "the user revolves Sketch9 through 360 degrees about its centerline to create a separate toroidal solid"
)]
fn revolve_sketch9_torus(world: &mut ApplicationWorld) {
    revolve_sketch9_full_turn_torus(world);
}

#[given("the user creates a separate offset surface 1 mm from an exposed planar block face")]
fn create_separate_offset_surface(world: &mut ApplicationWorld) {
    create_offset_surface_1mm_on_exposed_planar_face(world);
}

#[given("the toroidal solid and offset surface are visible")]
fn toroidal_solid_and_offset_surface_visible(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.bodies.len() >= 2,
        "expected multiple solid bodies including torus, saw {:?}",
        browser.bodies
    );
    assert!(
        !browser.surface_bodies.is_empty(),
        "expected visible offset surface body"
    );
}

#[given(
    "Replace Face is open with a torus face selected as an existing face and the separate surface selected as a new face"
)]
fn open_replace_face_with_torus_and_separate_surface(world: &mut ApplicationWorld) {
    open_replace_face_ribbon(world);
    select_torus_and_offset_surface(world);
}

#[then("the MCAD Software reports that creation of the Replace Face feature failed")]
fn reports_replace_face_creation_failed(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(FAILURE_MODAL_WINDOW).is_some(),
        "expected Create Replace Face feature failed modal, saw {}",
        world.text()
    );
    let text = super::super::world::squeezed(&world.window_text(FAILURE_MODAL_WINDOW));
    assert!(
        text.contains("CreateReplaceFacefeaturefailed")
            && text.contains("Edit")
            && text.contains("Cancel"),
        "expected Replace Face failure message with Edit and Cancel, saw {text}"
    );
}

#[when("Edit is clicked")]
fn edit_is_clicked(world: &mut ApplicationWorld) {
    click_modal_control(world, "Edit");
}

#[then("the Replace Face dialog returns with selections visible")]
fn replace_face_dialog_returns_with_selections(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry(FAILURE_MODAL_WINDOW).is_none());
    assert!(super::super::diagnostics::window_geometry(REPLACE_FACE_WINDOW).is_some());
    assert_eq!(
        existing_count(world),
        1,
        "expected 1 Existing Face retained"
    );
    assert_eq!(new_count(world), 1, "expected 1 New Face retained");
    let text = super::super::world::squeezed(&world.window_text(REPLACE_FACE_WINDOW));
    assert!(
        text.contains("1Face"),
        "expected 1 Face selection badges visible after Edit, saw {text}"
    );
}

#[then("the Replace Face dialog closes.")]
fn replace_face_dialog_closes(_world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(REPLACE_FACE_WINDOW).is_none(),
        "expected Replace Face dialog to close after Cancel"
    );
}

#[when("Automatic Face Chain is checked")]
fn automatic_face_chain_is_checked(world: &mut ApplicationWorld) {
    click_replace_face_control(world, "AutomaticFaceChain");
}

#[then("the checkbox is checked.")]
fn automatic_face_chain_checkbox_is_checked(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .replace_face_preview
        .expect("expected Replace Face preview state");
    assert!(
        preview.request.automatic_face_chain,
        "expected Automatic Face Chain checkbox to be checked"
    );
}
