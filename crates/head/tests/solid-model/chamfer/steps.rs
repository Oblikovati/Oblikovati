// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Part Modify Chamfer scenarios (chamfer-01..05, 08..09).
mod partial_and_layout;
pub(super) mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiChamferV2Preview;
use setup::{
    CHAMFER_WINDOW, accept_chamfer_and_wait, assert_chamfer_ready, click_body_face_for_chamfer,
    click_chamfer_control, click_matching_edge, open_chamfer_fixture, open_chamfer_ribbon,
    pick_valid_unmodified_edge, reopen_chamfer, replace_chamfer_field, solid_volume,
};

#[given("the part has two solid bodies with Extrude1, Extrusion2 and Fillet1 in its history")]
fn verify_two_body_fillet_history(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(browser.bodies.len(), 2, "expected 2 solid bodies");
    for name in ["Extrude1", "Extrusion2", "Fillet1"] {
        assert!(
            browser.features.iter().any(|f| f.name == name),
            "expected {name} in browser: {:?}",
            browser.features
        );
    }
}

#[given(
    "the equal-distance Chamfer dialog has the rounded cylinder’s planar-end circular edge selected and a distance of 2 mm"
)]
fn select_rounded_cylinder_circular_edge_2mm(world: &mut ApplicationWorld) {
    open_chamfer_ribbon(world);
    replace_chamfer_field(world, "Distance", "2 mm");
    // Locate the rounded cylinder body (5 edges) and click a circular edge that fails at 2 mm.
    let doc = world.active_id();
    let rounded_idx = {
        let app = world.app();
        let ws = app.owner.documents().workspace();
        let session = ws.session(doc).unwrap();
        let guid = ws.identity(doc).unwrap().internal_name;
        let eval = app.owner.part_scenes_current(doc).unwrap();
        eval.ordered_bodies()
            .enumerate()
            .find(|(_, body)| {
                body.edge_reference_keys(session.history(), guid)
                    .is_ok_and(|edges| edges.len() == 5)
            })
            .map(|(i, _)| i)
            .unwrap_or(1)
    };
    let candidates = setup::collect_edge_candidates(world, rounded_idx);
    setup::pan_clear_of_panel(world);
    for (_, _, _, mid, _, closed) in &candidates {
        if !closed {
            continue;
        }
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, *mid) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if world
            .state()
            .chamfer_v2_preview
            .as_ref()
            .is_some_and(|p| matches!(p.result, GuiChamferV2Preview::Failed(_)))
        {
            return;
        }
    }
    // Fallback: click any closed edge on rounded_idx.
    click_matching_edge(world, rounded_idx, |_, _, _, _, closed| closed);
}

#[when("I click OK on the part")]
fn click_ok_on_part(world: &mut ApplicationWorld) {
    click_chamfer_control(world, "OK");
    world.pump(Vec::new());
}

#[then("the MCAD Software reports that creating the chamfer feature failed")]
fn reports_chamfer_creation_failed(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry("Create Chamfer").is_some(),
        "expected Create Chamfer failure modal, saw {}",
        world.text()
    );
    let text = super::super::world::squeezed(&world.window_text("Create Chamfer"));
    assert!(
        text.contains("Creatingthechamferfeaturefailed"),
        "expected failure message in Create Chamfer modal, saw {text}"
    );
}

#[when("I choose Edit")]
fn choose_edit_in_failure_modal(world: &mut ApplicationWorld) {
    let center = world.run_in_window("Create Chamfer", "Edit").center();
    world.click(super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[then("the Chamfer dialog returns with the selection and distance visible")]
fn chamfer_dialog_returns_with_selection_and_distance(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry("Create Chamfer").is_none());
    assert!(super::super::diagnostics::window_geometry(CHAMFER_WINDOW).is_some());
    let text = super::super::world::squeezed(&world.window_text(CHAMFER_WINDOW));
    assert!(
        text.contains("1Edge") && text.contains("2mm"),
        "expected 1 Edge and 2 mm preserved in Chamfer dialog, saw {text}"
    );
}

#[given("I open Chamfer")]
#[when("I open Chamfer")]
fn open_chamfer_step(world: &mut ApplicationWorld) {
    open_chamfer_ribbon(world);
}

#[when("I choose the second chamfer type")]
fn choose_second_chamfer_type(world: &mut ApplicationWorld) {
    click_chamfer_control(world, "Dist/Angle");
}

#[then("the dialog shows edge and face selectors, Distance, and Angle")]
fn shows_dist_angle_controls(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(CHAMFER_WINDOW));
    for needle in ["Edges", "Face", "Distance", "Angle"] {
        assert!(
            text.contains(needle),
            "expected {needle} in Chamfer, saw {text}"
        );
    }
}

#[when("I choose the third chamfer type")]
fn choose_third_chamfer_type(world: &mut ApplicationWorld) {
    click_chamfer_control(world, "TwoDist");
}

#[then("the dialog shows Distance1 and Distance2")]
fn shows_two_distance_controls(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(CHAMFER_WINDOW));
    assert!(
        text.contains("Distance1") && text.contains("Distance2"),
        "expected Distance1 and Distance2, saw {text}"
    );
}

#[when("I open the Partial tab")]
fn open_partial_tab(world: &mut ApplicationWorld) {
    click_chamfer_control(world, "Partial");
}

#[then("a table and a Set Driven Dimension control are shown")]
fn shows_partial_table_and_driven_dimension(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(CHAMFER_WINDOW));
    assert!(
        text.contains("SetDrivenDimension") && text.contains("ToStart"),
        "expected Partial table and Set Driven Dimension, saw {text}"
    );
}

#[when("I open equal-distance Chamfer, select an upper edge and accept the 2 mm preview")]
fn create_2mm_equal_chamfer_on_upper_edge(world: &mut ApplicationWorld) {
    let before_vol = solid_volume(world);
    open_chamfer_ribbon(world);
    replace_chamfer_field(world, "Distance", "2 mm");
    pick_valid_unmodified_edge(world, false);
    assert_chamfer_ready(world);
    accept_chamfer_and_wait(world);
    let after_vol = solid_volume(world);
    assert!(after_vol < before_vol, "expected bevel to remove volume");
}

#[then("a bevel appears and Chamfer1 is added to the browser")]
fn verify_chamfer1_in_browser(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "Chamfer1"),
        "expected Chamfer1 in browser: {:?}",
        browser.features
    );
}

#[when("I reopen Chamfer1 and change Distance to 1 mm")]
fn reopen_chamfer1_and_change_to_1mm(world: &mut ApplicationWorld) {
    reopen_chamfer(world, "Chamfer1");
    replace_chamfer_field(world, "Distance", "1 mm");
    assert_chamfer_ready(world);
}

#[then("the preview narrows and accepting it updates the bevel")]
fn accept_narrowed_chamfer1(world: &mut ApplicationWorld) {
    let before_vol = solid_volume(world);
    accept_chamfer_and_wait(world);
    let after_vol = solid_volume(world);
    assert!(
        after_vol > before_vol,
        "expected narrower 1 mm bevel to increase solid volume ({after_vol} > {before_vol})"
    );
}

#[given("a writable copy of `fixtures/cad/chamfer-first-candidate.opd` is open with Chamfer1")]
fn open_chamfer_first_candidate(world: &mut ApplicationWorld) {
    open_chamfer_fixture(world, "fixtures/cad/chamfer-first-candidate.opd");
    let browser = world.state().browser.expect("expected browser");
    assert!(browser.features.iter().any(|f| f.name == "Chamfer1"));
}

#[given("the block has an unmodified side face and edge")]
fn verify_unmodified_face_and_edge(world: &mut ApplicationWorld) {
    let edges = setup::collect_edge_candidates(world, 0);
    assert!(edges.len() >= 12, "expected block with unmodified edges");
}

#[when("I choose distance-and-angle mode, select the face and its upper edge")]
fn choose_distance_angle_face_and_edge(world: &mut ApplicationWorld) {
    open_chamfer_ribbon(world);
    click_chamfer_control(world, "Dist/Angle");
    replace_chamfer_field(world, "Distance", "2 mm");
    replace_chamfer_field(world, "Angle", "45 deg");
    click_chamfer_control(world, "SelectFace");
    click_body_face_for_chamfer(world, 0);
    pick_valid_unmodified_edge(world, false);
}

#[then("the MCAD Software previews the chamfer with 2 mm and 45 degrees")]
fn verify_2mm_45deg_preview(world: &mut ApplicationWorld) {
    assert_chamfer_ready(world);
    let preview = world.state().chamfer_v2_preview.unwrap();
    assert_eq!(preview.request.distance.trim(), "2 mm");
    assert!(preview.request.angle.contains("45"));
    assert!(world.state().extrude_preview_active);
}

#[when("I accept it")]
#[when("I accept the preview")]
#[when("I accept the dialog")]
fn accept_current_chamfer(world: &mut ApplicationWorld) {
    accept_chamfer_and_wait(world);
}

#[then("a second bevel and Chamfer2 appear")]
fn verify_chamfer2_appears(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "Chamfer2"),
        "expected Chamfer2 in browser: {:?}",
        browser.features
    );
}

#[given(
    "I create Chamfer2 on an unmodified edge using distance-and-angle mode at 2 mm and 45 degrees"
)]
fn create_chamfer2_dist_angle(world: &mut ApplicationWorld) {
    choose_distance_angle_face_and_edge(world);
    assert_chamfer_ready(world);
    accept_chamfer_and_wait(world);
    verify_chamfer2_appears(world);
}

#[given("I open Chamfer in two-distance mode and select another unmodified block edge")]
fn open_two_distance_and_select_edge(world: &mut ApplicationWorld) {
    open_chamfer_ribbon(world);
    click_chamfer_control(world, "TwoDist");
    replace_chamfer_field(world, "Distance1", "2 mm");
    replace_chamfer_field(world, "Distance2", "2 mm");
    pick_valid_unmodified_edge(world, false);
}

#[when("I set Distance1 to 1 mm while Distance2 is 2 mm and click the reverse control")]
fn set_two_distances_and_reverse(world: &mut ApplicationWorld) {
    replace_chamfer_field(world, "Distance1", "1 mm");
    replace_chamfer_field(world, "Distance2", "2 mm");
    assert_chamfer_ready(world);
    click_chamfer_control(world, "Reverse");
}

#[then("the displayed values swap to 2 mm and 1 mm and the preview changes orientation")]
fn verify_reversed_two_distances(world: &mut ApplicationWorld) {
    assert_chamfer_ready(world);
    let preview = world.state().chamfer_v2_preview.unwrap();
    assert_eq!(preview.request.distance.trim(), "2 mm");
    assert_eq!(preview.request.distance2.trim(), "1 mm");
    assert!(preview.request.reversed);
}

#[then("the MCAD Software creates Chamfer3 on the block")]
fn verify_chamfer3_created(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "Chamfer3"),
        "expected Chamfer3 in browser: {:?}",
        browser.features
    );
}

#[given("I open Chamfer and select an unmodified edge")]
fn open_chamfer_select_unmodified_edge(world: &mut ApplicationWorld) {
    open_chamfer_ribbon(world);
    pick_valid_unmodified_edge(world, false);
}

#[given(
    "a 1 mm edge chamfer preview is visible with the second Edge Chain and Setback options selected"
)]
fn select_second_edge_chain_and_setback_1mm(world: &mut ApplicationWorld) {
    replace_chamfer_field(world, "Distance", "1 mm");
    click_chamfer_control(world, "Single");
    click_chamfer_control(world, "Flat");
    assert_chamfer_ready(world);
    let preview = world.state().chamfer_v2_preview.unwrap();
    assert!(preview.request.edge_chain_second && preview.request.setback_second);
}

#[when("I click Apply")]
fn click_chamfer_apply(world: &mut ApplicationWorld) {
    click_chamfer_control(world, "Apply");
    world.pump(Vec::new());
}

#[then("Chamfer2 appears in the browser and the dialog remains open for another selection")]
fn verify_apply_created_chamfer2_and_kept_open(world: &mut ApplicationWorld) {
    verify_chamfer2_appears(world);
    assert!(
        super::super::diagnostics::window_geometry(CHAMFER_WINDOW).is_some(),
        "expected Chamfer dialog to remain open after Apply"
    );
    let text = super::super::world::squeezed(&world.window_text(CHAMFER_WINDOW));
    assert!(
        text.contains("SelectEdges"),
        "expected Chamfer selection cleared for next edge, saw {text}"
    );
}

#[then("the dialog closes and Chamfer2 remains in the browser")]
fn verify_closed_and_chamfer2_remains(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry(CHAMFER_WINDOW).is_none());
    verify_chamfer2_appears(world);
}

#[given("I open equal-distance Chamfer and select a circular end edge")]
fn open_chamfer_select_circular_end_edge(world: &mut ApplicationWorld) {
    open_chamfer_ribbon(world);
    click_matching_edge(world, 0, |_, _, _, _, closed| closed);
    assert_chamfer_ready(world);
}

#[when("the Distance value is replaced with zero")]
fn replace_chamfer_distance_with_zero(world: &mut ApplicationWorld) {
    replace_chamfer_field(world, "Distance", "0");
}

#[then("the value is displayed in red")]
fn verify_zero_value_displayed_in_red(world: &mut ApplicationWorld) {
    let preview = world.state().chamfer_v2_preview;
    assert!(
        preview.is_none_or(|p| p.request.distance.trim() == "0"),
        "expected zero Distance in Chamfer draft"
    );
    assert!(!world.run_is_enabled("OK"));
}

#[when("the Distance value is replaced with 1")]
fn replace_chamfer_distance_with_one(world: &mut ApplicationWorld) {
    replace_chamfer_field(world, "Distance", "1");
    assert_chamfer_ready(world);
}

#[then("the Chamfer dialog closes without creating a feature")]
fn verify_chamfer_closes_without_feature(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry(CHAMFER_WINDOW).is_none());
    let browser = world.state().browser.expect("expected browser");
    assert!(
        !browser
            .features
            .iter()
            .any(|f| f.name.starts_with("Chamfer")),
        "expected no Chamfer created, saw {:?}",
        browser.features
    );
}
