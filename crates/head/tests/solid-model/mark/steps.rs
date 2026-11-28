// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Part Modify Mark scenarios (mark-01..mark-04, #3918).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_model::feature::mark::{MarkMethod, MarkType};
use setup::{
    MARK_WINDOW, click_add_geometry_set_plus, click_mark_control, click_mark_ribbon_raw,
    click_planar_prism_face, open_first_mark_type_combo, open_mark_panel, popup_option_texts,
    select_mark_popup_item,
};

#[given("the part has two solid bodies")]
fn part_has_two_solid_bodies(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(
        browser.bodies.len(),
        2,
        "expected 2 solid bodies in part, saw {:?}",
        browser.bodies
    );
}

#[when("the user opens Mark")]
fn user_opens_mark(world: &mut ApplicationWorld) {
    click_mark_ribbon_raw(world);
}

#[then(
    "the MCAD Software displays a tip that Mark is not supported when multiple solid bodies exist."
)]
fn displays_multi_body_mark_tip(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(MARK_WINDOW).is_some(),
        "expected Mark multi-body tip modal, saw {}",
        world.text()
    );
    let text = super::super::world::squeezed(&world.window_text(MARK_WINDOW));
    assert!(
        text.contains("Markisnotsupportedwhenmultiplesolidbodiesexist"),
        "expected multi-body Mark tip text, saw {text}"
    );
    assert!(
        world.state().mark_preview.is_none(),
        "Mark panel should not open when multiple solid bodies exist"
    );
}

#[given("the single-solid part has a visible open-line sketch")]
fn single_solid_has_visible_open_line_sketch(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(
        browser.bodies.len(),
        1,
        "expected 1 solid body, saw {:?}",
        browser.bodies
    );
    assert!(
        browser.sketches.iter().any(|sketch| sketch.visible),
        "expected a visible open-line sketch, saw {:?}",
        browser.sketches
    );
}

#[given("Mark is open")]
fn mark_is_open(world: &mut ApplicationWorld) {
    open_mark_panel(world);
}

#[when("the user opens the geometry-set type list")]
fn user_opens_geometry_set_type_list(world: &mut ApplicationWorld) {
    open_first_mark_type_combo(world);
}

#[then("Mark Surface and Mark Through are available.")]
fn mark_surface_and_through_available(world: &mut ApplicationWorld) {
    let options = popup_option_texts(world);
    assert!(
        options.iter().any(|item| item == "MarkSurface"),
        "expected Mark Surface in dropdown, saw {options:?}"
    );
    assert!(
        options.iter().any(|item| item == "MarkThrough"),
        "expected Mark Through in dropdown, saw {options:?}"
    );
}

#[when("the user chooses Mark Through and clicks the plus beside Mark Geometry Sets")]
fn choose_mark_through_and_add_set(world: &mut ApplicationWorld) {
    select_mark_popup_item(world, "Mark Through");
    let preview = world
        .state()
        .mark_preview
        .expect("expected Mark preview state");
    assert_eq!(preview.request.geometry_sets.len(), 1);
    assert_eq!(
        preview.request.geometry_sets[0].mark_type,
        MarkType::MarkThrough
    );
    click_add_geometry_set_plus(world);
}

#[then("another selection row appears.")]
fn another_selection_row_appears(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .mark_preview
        .expect("expected Mark preview state");
    assert_eq!(
        preview.request.geometry_sets.len(),
        2,
        "expected second Mark Geometry Set row, saw {:?}",
        preview.request.geometry_sets
    );
    let text = super::super::world::squeezed(&world.window_text(MARK_WINDOW));
    assert!(
        text.contains("Set1") && text.contains("Set2"),
        "expected Set 1 and Set 2 rows in Mark panel, saw {text}"
    );
}

#[given("the candidate single-solid prism has 45 faces")]
fn candidate_prism_has_45_faces(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(browser.bodies.len(), 1);
    assert_eq!(
        browser.bodies[0].faces, 45,
        "expected 45-face prism candidate, saw {:?}",
        browser.bodies
    );
}

#[given("the user opens Mark and selects the second Method option")]
fn open_mark_and_select_second_method(world: &mut ApplicationWorld) {
    open_mark_panel(world);
    click_mark_control(world, "Wrap to Face");
    let preview = world
        .state()
        .mark_preview
        .expect("expected Mark preview state");
    assert_eq!(preview.request.method, MarkMethod::WrapToFace);
}

#[when("the user selects the planar face")]
fn user_selects_planar_face(world: &mut ApplicationWorld) {
    click_planar_prism_face(world);
}

#[then("the Face selector shows 1 Face.")]
fn face_selector_shows_1_face(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(MARK_WINDOW));
    assert!(
        text.contains("1Face"),
        "expected Face selector showing 1 Face, saw {text}"
    );
    let preview = world
        .state()
        .mark_preview
        .expect("expected Mark preview state");
    assert_eq!(preview.request.faces.len(), 1);
}

#[when("the user enables Automatic Face Chain")]
fn user_enables_automatic_face_chain(world: &mut ApplicationWorld) {
    click_mark_control(world, "Automatic Face Chain");
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[then("the selector shows 45 Faces on this part.")]
fn selector_shows_45_faces(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(MARK_WINDOW));
    assert!(
        text.contains("45Faces"),
        "expected Face selector showing 45 Faces, saw {text}"
    );
    let preview = world
        .state()
        .mark_preview
        .expect("expected Mark preview state");
    assert!(preview.request.automatic_face_chain);
    assert_eq!(preview.request.faces.len(), 45);
}

#[given("Mark is open and no sketch geometry has been selected")]
fn mark_open_no_sketch_geometry(world: &mut ApplicationWorld) {
    open_mark_panel(world);
    let preview = world
        .state()
        .mark_preview
        .expect("expected Mark preview state");
    assert!(
        preview
            .request
            .geometry_sets
            .iter()
            .all(|set| set.curves.is_empty()),
        "expected no sketch geometry selected initially"
    );
}

#[then("OK is disabled.")]
fn mark_ok_is_disabled(world: &mut ApplicationWorld) {
    let window = if super::super::diagnostics::window_geometry("Replace Face").is_some() {
        "Replace Face"
    } else {
        MARK_WINDOW
    };
    assert!(
        world.run_in_window(window, "OK").alpha < 200,
        "expected OK disabled in {window}"
    );
}

#[then("the Mark panel closes without creating a mark.")]
fn mark_panel_closes_without_creating_mark(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(MARK_WINDOW).is_none(),
        "expected Mark panel to close on Cancel"
    );
    let browser = world.state().browser.expect("expected browser");
    assert!(
        !browser
            .features
            .iter()
            .any(|feature| feature.kind == "Mark"),
        "expected no Mark feature created after Cancel, saw {:?}",
        browser.features
    );
}
