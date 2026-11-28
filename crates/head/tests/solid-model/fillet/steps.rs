// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Part Modify Fillet scenarios (fillet-01..fillet-08, fillet-11).
mod setbacks_and_modes;
pub(super) mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_model::feature::fillet_options::{FilletContinuity, FilletMode};
use setup::{
    FILLET_WINDOW, accept_fillet_and_wait, click_any_solid_edge, click_fillet_control,
    open_fillet_fixture, open_fillet_ribbon, reopen_fillet1, replace_fillet_field,
};

#[given("a writable copy of `fixtures/cad/docking-visible-solid-current.opd` is open")]
fn open_visible_solid_for_fillet(world: &mut ApplicationWorld) {
    open_fillet_fixture(world, "fixtures/cad/docking-visible-solid-current.opd");
}

#[given("a writable copy of `fixtures/cad/fillet-constant-regenerated.opd` is open")]
#[given(
    "a writable copy of `fixtures/cad/fillet-constant-regenerated.opd` is open with Fillet1 at a 2 mm radius"
)]
fn open_fillet_constant_regenerated(world: &mut ApplicationWorld) {
    open_fillet_fixture(world, "fixtures/cad/fillet-constant-regenerated.opd");
}

#[given("a writable copy of `fixtures/cad/fillet-circular-edge-candidate.opd` is open")]
fn open_fillet_circular_candidate(world: &mut ApplicationWorld) {
    open_fillet_fixture(world, "fixtures/cad/fillet-circular-edge-candidate.opd");
}

#[given("a writable copy of `fixtures/cad/fillet-two-bodies-rounded-cylinder.opd` is open")]
fn open_fillet_two_bodies(world: &mut ApplicationWorld) {
    open_fillet_fixture(world, "fixtures/cad/fillet-two-bodies-rounded-cylinder.opd");
}

#[given("the user opens Fillet on its rectangular solid")]
#[given("I open Fillet on its rectangular solid")]
#[given("the user opens Fillet with a 2 mm radius")]
fn open_fillet_on_block(world: &mut ApplicationWorld) {
    open_fillet_ribbon(world);
}

#[when("I select the edge")]
fn select_one_edge(world: &mut ApplicationWorld) {
    click_any_solid_edge(world, false);
}

#[then("the panel shows one edge and a 2 mm radius preview")]
fn panel_shows_one_edge_and_2mm(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(FILLET_WINDOW));
    assert!(
        text.contains("1Edge") && text.contains("2mm"),
        "expected 1 Edge and 2 mm in Fillet panel, saw {text}"
    );
    assert!(world.state().extrude_preview_active);
    assert!(world.run_is_enabled("OK"));
}

#[then("Fillet1 appears in the model browser")]
fn fillet1_in_model_browser(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "Fillet1"),
        "expected Fillet1 in browser: {:?}",
        browser.features
    );
}

#[given("the user opens Fillet and selects an edge of its rectangular solid")]
fn open_fillet_and_select_edge(world: &mut ApplicationWorld) {
    open_fillet_ribbon(world);
    click_any_solid_edge(world, false);
}

#[when("I expand Advanced Properties")]
fn expand_advanced_properties(world: &mut ApplicationWorld) {
    click_fillet_control(world, ">AdvancedProperties");
}

#[then("Roll Along Sharp Edges is unchecked")]
fn roll_along_sharp_unchecked(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(FILLET_WINDOW));
    assert!(
        text.contains("RollAlongSharpEdges"),
        "expected Roll Along Sharp Edges checkbox, saw {text}"
    );
    assert!(!world.checked("Roll Along Sharp Edges"));
}

#[then("Rolling Ball Where Possible and Preserve All Features are checked")]
fn rolling_ball_and_preserve_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Rolling Ball Where Possible"));
    assert!(world.checked("Preserve All Features"));
}

#[then("Automatic Edge Chain is disabled")]
fn automatic_edge_chain_disabled(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled("Automatic Edge Chain"));
}

#[when("I edit Fillet1 and change the radius to 1 mm")]
fn edit_fillet1_to_1mm(world: &mut ApplicationWorld) {
    reopen_fillet1(world);
    replace_fillet_field(world, "Radius", "1 mm");
}

#[when("I accept and reopen the feature")]
fn accept_and_reopen_fillet1(world: &mut ApplicationWorld) {
    accept_fillet_and_wait(world);
    reopen_fillet1(world);
}

#[then("the radius field shows 1 mm")]
fn radius_field_shows_1mm(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(FILLET_WINDOW));
    assert!(
        text.contains("1mm") || text.contains("0.1cm"),
        "expected 1 mm radius in Fillet panel, saw {text}"
    );
}

#[given("Fillet1 is open for editing")]
fn fillet1_open_for_editing(world: &mut ApplicationWorld) {
    reopen_fillet1(world);
}

#[when("I choose the variable-radius palette icon")]
fn choose_variable_radius_icon(world: &mut ApplicationWorld) {
    click_fillet_control(world, "Variable");
}

#[then("a Variable Radius Fillets edge selection row appears")]
fn variable_radius_row_appears(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(FILLET_WINDOW));
    assert!(
        text.contains("VariableRadiusFillets") && text.contains("Edges"),
        "expected Variable Radius Fillets row, saw {text}"
    );
}

#[then("Smooth Radius Transition is checked")]
#[given("Smooth Radius Transition is checked")]
fn smooth_radius_transition_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Smooth Radius Transition"));
}

#[when("I choose the corner setback palette icon")]
fn choose_corner_setback_icon(world: &mut ApplicationWorld) {
    click_fillet_control(world, "Setbacks");
}

#[then("the panel displays Corner Setbacks")]
fn panel_displays_corner_setbacks(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(FILLET_WINDOW));
    assert!(
        text.contains("CornerSetbacks"),
        "expected Corner Setbacks heading, saw {text}"
    );
}

#[then("the status bar reports no vertex available")]
fn status_bar_reports_no_vertex_available(world: &mut ApplicationWorld) {
    world.expect_status("No vertex available");
}

#[given("I open Fillet, choose Variable Radius Fillets and select a circular end edge")]
#[given("I open Fillet, choose variable-radius mode and select a circular end edge")]
fn open_variable_fillet_and_pick_circular_edge(world: &mut ApplicationWorld) {
    open_fillet_ribbon(world);
    click_fillet_control(world, "Variable");
    click_any_solid_edge(world, false);
}

#[given("the initial position 0.0 has a radius of 2 mm")]
fn initial_pos_0_is_2mm(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .fillet_v2_preview
        .expect("expected variable fillet preview");
    assert_eq!(preview.request.mode, FilletMode::Variable);
    assert_eq!(preview.request.variable_points[0].position, "0.0");
    assert_eq!(preview.request.variable_points[0].radius, "2 mm");
}

#[when("I add a position and set the 0.5 radius to 1 mm")]
#[given("I set radius 2 mm at position 0.0 and radius 1 mm at position 0.5")]
fn add_position_0_5_radius_1mm(world: &mut ApplicationWorld) {
    click_fillet_control(world, "AddPosition");
    replace_fillet_field(world, "Pos0.5", "1 mm");
}

#[then("the edge summary shows a range from 1 to 2 mm")]
fn edge_summary_shows_1_to_2mm(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(FILLET_WINDOW));
    assert!(
        text.contains("1to2mm"),
        "expected 1 to 2 mm summary in Fillet panel, saw {text}"
    );
}

#[when("I accept and reopen the fillet")]
#[given("I accept the fillet and reopen it for editing")]
fn accept_and_reopen_the_fillet(world: &mut ApplicationWorld) {
    accept_fillet_and_wait(world);
    reopen_fillet1(world);
}

#[then("the position rows show 0.0 with 2 mm and 0.5 with 1 mm")]
fn position_rows_retained(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .fillet_v2_preview
        .expect("expected reopened variable fillet preview");
    assert_eq!(preview.request.variable_points.len(), 2);
    assert_eq!(preview.request.variable_points[0].position, "0.0");
    assert_eq!(preview.request.variable_points[0].radius, "2 mm");
    assert_eq!(preview.request.variable_points[1].position, "0.5");
    assert_eq!(preview.request.variable_points[1].radius, "1 mm");
}

#[when("I clear Smooth Radius Transition and accept the edit")]
fn clear_smooth_transition_and_accept(world: &mut ApplicationWorld) {
    click_fillet_control(world, "SmoothRadiusTransition");
    accept_fillet_and_wait(world);
}

#[when("I reopen the fillet")]
#[when("I reopen Fillet1")]
fn reopen_fillet_when(world: &mut ApplicationWorld) {
    reopen_fillet1(world);
}

#[then("Smooth Radius Transition is unchecked")]
fn smooth_transition_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("Smooth Radius Transition"));
}

#[then("the two radius positions remain present")]
fn two_radius_positions_remain(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .fillet_v2_preview
        .expect("expected reopened variable fillet preview");
    assert_eq!(preview.request.variable_points.len(), 2);
}

#[when("I open the edge continuity menu")]
fn open_continuity_menu(world: &mut ApplicationWorld) {
    let row = world.run_in_window(FILLET_WINDOW, "Continuity").clone();
    let (origin, size) = super::super::diagnostics::window_geometry(FILLET_WINDOW).unwrap();
    let combo =
        super::super::diagnostics::pointer_pixels([origin[0] + size[0] * 0.72, row.center()[1]]);
    world.click(combo);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[then("Tangent and Smooth G2 are offered")]
fn tangent_and_smooth_g2_offered(world: &mut ApplicationWorld) {
    for item in ["Tangent", "SmoothG2"] {
        assert!(
            world
                .frame
                .runs
                .iter()
                .any(|run| world.frame.is_popup_row(run) && run.text == item),
            "expected continuity item {item}, saw {}",
            world.text()
        );
    }
}

#[when("I select Smooth G2, accept, and reopen the feature")]
fn select_smooth_g2_accept_reopen(world: &mut ApplicationWorld) {
    let item = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| world.frame.is_popup_row(run) && run.text == "SmoothG2")
        .unwrap()
        .center();
    world.click(super::super::diagnostics::pointer_pixels(item));
    world.pump(Vec::new());
    world.pump(Vec::new());
    accept_fillet_and_wait(world);
    reopen_fillet1(world);
}

#[then("Smooth G2 is highlighted in the continuity menu")]
fn smooth_g2_highlighted(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .fillet_v2_preview
        .expect("expected reopened fillet preview");
    assert_eq!(preview.request.continuity, FilletContinuity::SmoothG2);
    let text = super::super::world::squeezed(&world.window_text(FILLET_WINDOW));
    assert!(text.contains("SmoothG2"), "expected Smooth G2, saw {text}");
}

#[when("I enter zero as the radius")]
fn enter_zero_radius(world: &mut ApplicationWorld) {
    replace_fillet_field(world, "Radius", "0");
}

#[then("error indicators appear and OK is disabled")]
fn error_indicators_and_ok_disabled(world: &mut ApplicationWorld) {
    let text = world.window_text(FILLET_WINDOW);
    assert!(text.contains('!'), "expected error indicator, saw {text}");
    assert!(!world.run_is_enabled("OK"));
}

#[when("I cancel and reopen the feature")]
fn cancel_and_reopen_fillet(world: &mut ApplicationWorld) {
    click_fillet_control(world, "Cancel");
    reopen_fillet1(world);
}

#[then("the previously accepted radius is restored")]
fn previously_accepted_radius_restored(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(FILLET_WINDOW));
    assert!(
        text.contains("2mm") || text.contains("0.2cm"),
        "expected restored 2 mm radius, saw {text}"
    );
}
