// SPDX-License-Identifier: GPL-2.0-only
//! Native Part face picking, presentation, and context menus.
use super::{ApplicationWorld, diagnostics, viewport_orientation};
use crate::{HeadWindow, HeadWork};
use cucumber::{given, then, when};
use oblikovati_api::{client::Client, wire::OpenDocumentArgs};
use oblikovati_gui::GuiAuthoringContext;

#[path = "pixels.rs"]
mod pixels;

#[given("the MCAD Software is in the part modeling workspace")]
fn part_modeling_workspace(world: &mut ApplicationWorld) {
    world.add_part();
    assert_eq!(world.state().authoring, GuiAuthoringContext::Part);
}

#[given("no command or modal dialog is active")]
fn viewport_ready(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert!(!world.state().transaction_open);
}

#[given("`fixtures/cad/docking-visible-solid-current.opd` is open with an unselected front face")]
fn open_unselected_face(world: &mut ApplicationWorld) {
    viewport_orientation::orientation_part_upright(world);
    assert!(!world.state().model_face_chosen);
}

#[given("`fixtures/cad/docking-visible-solid-current.opd` is open with a solid face selected")]
fn open_selected_face(world: &mut ApplicationWorld) {
    open_unselected_face(world);
    click_visible_face(world);
    assert!(world.state().model_face_chosen);
}

#[when("the user clicks an unoccupied region of the face")]
fn click_visible_face(world: &mut ApplicationWorld) {
    let point = super::ribbon_keyboard_navigation::visible_face_point(world);
    let line = world.app().window.view_line(point).unwrap();
    let expected = world.app().owner.face_edges_on_line(line).unwrap();
    left_click(world, point);
    assert_eq!(world.app().owner.selected_browser_outline(), expected);
}

fn left_click(world: &mut ApplicationWorld, at: [i32; 2]) {
    #[cfg(target_os = "linux")]
    super::x11_input::click(at);
    #[cfg(not(target_os = "linux"))]
    world.click(at);
    world.pump(Vec::new());
}

#[then("the face appears teal with a bright green boundary")]
fn face_has_selection_colors(world: &mut ApplicationWorld) {
    pixels::expect_highlighted_face(world);
}

#[then("the pick cue identifies \"1. Face\"")]
fn face_pick_cue_visible(world: &mut ApplicationWorld) {
    world.expect_text("1. Face");
}

#[then("a compact contextual toolbar appears beside the pointer")]
fn face_toolbar_beside_pointer(world: &mut ApplicationWorld) {
    let cue = world
        .state()
        .model_pick_cue
        .expect("expected a face pick position");
    let at = world.app().window.model_pixel(cue.position).unwrap();
    let (origin, size) = diagnostics::window_geometry("##selection-toolbar")
        .expect("expected the contextual selection toolbar");
    let center: [f32; 2] = std::array::from_fn(|axis| origin[axis] + size[axis] / 2.);
    assert!((center[0] - at[0] as f32).abs() < size[0] + 32.);
    assert!((center[1] - at[1] as f32).abs() < size[1] + 32.);
    world.run_in_window("##selection-toolbar", "ZoomSelected");
}

#[when("the user right-clicks the face")]
fn right_click_selected_face(world: &mut ApplicationWorld) {
    let cue = world
        .state()
        .model_pick_cue
        .expect("expected the selected face");
    let point = world.app().window.model_pixel(cue.position).unwrap();
    #[cfg(target_os = "linux")]
    super::x11_input::right_click(point);
    #[cfg(not(target_os = "linux"))]
    world.right_click(point);
    world.pump(Vec::new());
}

#[then("a standard context menu appears")]
fn standard_face_context_menu(world: &mut ApplicationWorld) {
    expect_context_menu_row(world, "Properties");
}

#[then("a radial shortcut menu appears near the pointer")]
fn radial_face_context_menu(world: &mut ApplicationWorld) {
    let top = expect_context_menu_row(world, "New Sketch");
    let side = expect_context_menu_row(world, "Extrude");
    assert!(
        top[0] < side[0] && top[1] < side[1],
        "radial centers: {top:?}, {side:?}"
    );
}

#[then(
    "the radial menu includes \"Extrude\", \"Select Other...\", \"Select Tangencies\", and \"New Sketch\""
)]
fn radial_face_commands(world: &mut ApplicationWorld) {
    for label in [
        "Extrude",
        "Select Other...",
        "Select Tangencies",
        "New Sketch",
    ] {
        expect_context_menu_row(world, label);
    }
}

#[then("the standard menu includes \"Properties\" and \"Export Face As...\"")]
fn standard_face_commands(world: &mut ApplicationWorld) {
    expect_context_menu_row(world, "Properties");
    expect_context_menu_row(world, "Export Face As...");
}

fn expect_context_menu_row(world: &ApplicationWorld, name: &str) -> [f32; 2] {
    let wanted = name.split_whitespace().collect::<String>();
    world
        .frame
        .runs
        .iter()
        .find(|row| world.frame.is_popup_row(row) && row.text == wanted)
        .unwrap_or_else(|| {
            panic!(
                "face context menu {name:?}: expected a visible entry in {:?}",
                world.text()
            )
        })
        .center()
}

#[when("the user left-clicks empty viewport space")]
fn click_empty_viewport_space(world: &mut ApplicationWorld) {
    let point = pixels::empty_viewport_point(world);
    left_click(world, point);
}

#[then("the entity returns to its neutral appearance")]
fn face_returns_to_neutral(world: &mut ApplicationWorld) {
    assert!(!world.state().model_face_chosen);
    pixels::expect_neutral_face(world);
}

#[then("the contextual selection cue closes")]
fn contextual_feedback_closes(world: &mut ApplicationWorld) {
    world.expect_no_text("1. Face");
    assert!(diagnostics::window_geometry("##selection-toolbar").is_none());
}

#[given(
    "the MCAD Software displays fixture \"viewport-selection-solid.ipt\" in the part modeling workspace"
)]
fn open_viewport_selection_fixture(world: &mut ApplicationWorld) {
    let path = super::scenario_root().join("fixtures/cad/viewport-selection-solid.ipt");
    let request = OpenDocumentArgs {
        full_document_name: path.to_string_lossy().into_owned(),
        visible: true,
        ..Default::default()
    };
    Client::new(world.app().owner.document_host())
        .documents()
        .open(&request)
        .unwrap_or_else(|error| {
            panic!("fixture {path:?}: expected an open Inventor part: {error}")
        });
    world.pump(Vec::new());
    assert!(
        world.state().document.is_some(),
        "fixture {path:?}: expected an active part"
    );
}
