// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Part Modify Split scenarios (split-01..split-07, #3918).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiSplitPreview;
use oblikovati_model::feature::split::{SplitKeepSide, SplitMode, SplitTool};
use setup::{
    SPLIT_WINDOW, accept_split_and_wait, click_clear_on_split_row,
    click_rectangular_solid_in_viewport, click_split_control, click_top_planar_face,
    click_visible_sketch_line_tool, create_split1_all_faces, create_split1_midplane_solid,
    double_click_split1, open_disposable_split_fixture, open_split_ribbon, pick_browser_solid_body,
    pick_browser_work_plane1, pick_browser_xy_origin_plane, set_all_faces,
    switch_split_to_solid_mode, total_evaluated_edge_count,
};

#[given("its open sketch line is visible across the solid")]
fn open_sketch_line_visible(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(browser.bodies.len(), 1);
    assert_eq!(browser.bodies[0].faces, 14);
    assert!(
        browser.sketches.iter().any(|sketch| sketch.visible),
        "expected a visible open sketch line in browser: {:?}",
        browser.sketches
    );
}

#[when("the user opens Split and enables All Faces")]
fn open_split_and_enable_all_faces(world: &mut ApplicationWorld) {
    open_split_ribbon(world);
    set_all_faces(world, true);
}

#[when("the user selects the line as the tool")]
#[when("the user selects the visible sketch line")]
fn select_visible_sketch_line_tool(world: &mut ApplicationWorld) {
    click_visible_sketch_line_tool(world);
}

#[then("the tool selector displays 1 Curve and the faces selector displays 14 Faces")]
fn verify_1_curve_and_14_faces(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(SPLIT_WINDOW));
    assert!(
        text.contains("1Curve") && text.contains("14Faces"),
        "expected 1 Curve and 14 Faces in Split panel, saw {text}"
    );
    let preview = world.state().split_preview.expect("expected split preview");
    assert_eq!(preview.result, GuiSplitPreview::Ready);
}

#[given(
    "the user opens Split, enables All Faces and selects the visible open sketch line as the tool"
)]
fn open_split_all_faces_and_select_line(world: &mut ApplicationWorld) {
    open_split_ribbon(world);
    set_all_faces(world, true);
    click_visible_sketch_line_tool(world);
    let preview = world.state().split_preview.expect("expected split preview");
    assert_eq!(preview.result, GuiSplitPreview::Ready);
}

pub(in super::super) fn accept_split_from_shared(world: &mut ApplicationWorld) {
    accept_split_and_wait(world);
}

#[when("the user accepts Split")]
fn accept_split(world: &mut ApplicationWorld) {
    accept_split_and_wait(world);
}

#[then("Split1 appears in the model browser and new edges are visible on the model")]
fn split1_appears_with_new_edges(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser
            .features
            .iter()
            .any(|feature| feature.name == "Split1" && feature.kind == "Split"),
        "expected Split1 in browser features: {:?}",
        browser.features
    );
    // Source two-openings block has 14 faces and 36 edges; splitting all faces adds new edges.
    assert!(
        browser.bodies[0].faces > 14,
        "expected face count to increase after All Faces split, saw {}",
        browser.bodies[0].faces
    );
    let edges = total_evaluated_edge_count(world);
    assert!(
        edges > 36,
        "expected new split edges (> 36) on evaluated model, saw {edges}"
    );
}

#[when("the user double-clicks Split1")]
#[when("the user reopens Split1")]
#[given("the user reopens Split1 for editing")]
#[given("Split1 is open for editing")]
fn reopen_split1(world: &mut ApplicationWorld) {
    double_click_split1(world);
}

#[then("its properties reopen with the curve and 14 faces selected")]
fn verify_reopened_curve_and_14_faces(world: &mut ApplicationWorld) {
    let preview = world.state().split_preview.expect("expected split preview");
    assert!(matches!(
        preview.request.tool,
        Some(SplitTool::SketchCurve { .. })
    ));
    assert!(preview.request.all_faces);
    let text = super::super::world::squeezed(&world.window_text(SPLIT_WINDOW));
    assert!(
        text.contains("1Curve") && text.contains("14Faces"),
        "expected reopened Split1 with 1 Curve and 14 Faces, saw {text}"
    );
}

#[given("the user creates Split1 using the open sketch line with All Faces enabled")]
fn create_split1_all_faces_step(world: &mut ApplicationWorld) {
    create_split1_all_faces(world);
}

#[when("the user disables All Faces")]
fn disable_all_faces_step(world: &mut ApplicationWorld) {
    set_all_faces(world, false);
}

#[then("the face selection is cleared and OK is unavailable")]
fn faces_cleared_and_ok_unavailable(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(SPLIT_WINDOW));
    assert!(
        text.contains("SelectFaces"),
        "expected Select Faces after disabling All Faces, saw {text}"
    );
    assert!(!world.run_is_enabled("OK"), "expected OK disabled");
}

#[when("the user selects one planar face")]
fn select_one_planar_face(world: &mut ApplicationWorld) {
    click_top_planar_face(world);
}

#[then("the selector displays 1 Face and OK becomes enabled")]
fn verify_1_face_and_ok_enabled(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(SPLIT_WINDOW));
    assert!(
        text.contains("1Face"),
        "expected 1 Face in Split panel, saw {text}"
    );
    assert!(world.run_is_enabled("OK"), "expected OK enabled");
}

#[then("the property panel closes and Split1 remains in the browser")]
fn panel_closes_and_split1_remains(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry(SPLIT_WINDOW).is_none());
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser
            .features
            .iter()
            .any(|feature| feature.name == "Split1"),
        "expected Split1 to remain in browser: {:?}",
        browser.features
    );
}

#[given("the Split panel is in Faces mode")]
fn split_panel_in_faces_mode(world: &mut ApplicationWorld) {
    open_split_ribbon(world);
    let preview = world.state().split_preview.expect("expected split preview");
    assert_eq!(preview.request.mode, SplitMode::Faces);
}

#[when("the user clicks the icon beside the Faces selector")]
fn click_mode_switch_beside_faces(world: &mut ApplicationWorld) {
    click_split_control(world, "Solid");
}

#[then("the selector changes to Solid")]
fn selector_changes_to_solid(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(SPLIT_WINDOW));
    assert!(
        text.contains("SelectSolid"),
        "expected Select Solid selector in Solid mode, saw {text}"
    );
    let preview = world.state().split_preview.expect("expected split preview");
    assert_eq!(preview.request.mode, SplitMode::Solid);
}

#[then("three Keep Side options appear")]
fn three_keep_side_options_appear(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(SPLIT_WINDOW));
    assert!(
        text.contains("KeepSide")
            && text.contains("KeepBoth")
            && text.contains("KeepSideA")
            && text.contains("KeepSideB"),
        "expected three Keep Side options in Split panel, saw {text}"
    );
}

#[given(expr = "a fresh MCAD Software session opens a disposable copy of `{word}`")]
fn fresh_session_opens_disposable_copy(world: &mut ApplicationWorld, relative: String) {
    open_disposable_split_fixture(world, &relative);
}

#[given("no other document with the same native identity is open")]
fn verify_single_document_identity(world: &mut ApplicationWorld) {
    let active = world.active_id();
    let docs = world.state().documents;
    let workspace = world.app().owner.documents().workspace();
    let active_guid = workspace
        .identity(active)
        .expect("expected active identity")
        .internal_name;
    let same_identity = docs
        .iter()
        .filter(|row| {
            workspace
                .identity(row.id)
                .is_ok_and(|id| id.internal_name == active_guid)
        })
        .count();
    assert_eq!(same_identity, 1, "expected unique active document identity");
}

#[given("the part has two solid bodies and the rectangular body crossing XY is named Solid3")]
fn verify_two_bodies_with_solid3(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(browser.bodies.len(), 2, "bodies: {:?}", browser.bodies);
    assert!(
        browser.bodies.iter().any(|body| body.name == "Solid3"),
        "expected Solid3 among bodies: {:?}",
        browser.bodies
    );
}

#[given("Split is in Solid mode")]
fn split_is_in_solid_mode(world: &mut ApplicationWorld) {
    open_split_ribbon(world);
    switch_split_to_solid_mode(world);
}

#[when("the user selects XY Plane as the tool and Solid3 as the solid")]
fn select_xy_plane_and_solid3(world: &mut ApplicationWorld) {
    pick_browser_xy_origin_plane(world);
    pick_browser_solid_body(world, "Solid3");
}

#[when("the user selects Work Plane1 as the tool and the rectangular solid")]
fn select_work_plane1_and_rectangular_solid(world: &mut ApplicationWorld) {
    pick_browser_work_plane1(world);
    click_rectangular_solid_in_viewport(world);
}

#[then("the selectors show 1 Work Plane and 1 Solid")]
fn selectors_show_1_work_plane_and_1_solid(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(SPLIT_WINDOW));
    assert!(
        text.contains("1WorkPlane") && text.contains("1Solid"),
        "expected 1 Work Plane and 1 Solid in Split panel, saw {text}"
    );
    let preview = world.state().split_preview.expect("expected split preview");
    assert_eq!(preview.result, GuiSplitPreview::Ready);
}

#[when("the user accepts with the first Keep Side option selected")]
fn accept_with_first_keep_side(world: &mut ApplicationWorld) {
    click_split_control(world, "KeepBoth");
    let preview = world.state().split_preview.expect("expected split preview");
    assert_eq!(preview.request.keep_side, SplitKeepSide::Both);
    accept_split_and_wait(world);
}

#[then("the Solid Bodies folder shows three bodies, including Solid4 and Solid5")]
fn solid_bodies_shows_three_with_solid4_and_solid5(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    let names: Vec<&str> = browser
        .bodies
        .iter()
        .map(|body| body.name.as_str())
        .collect();
    assert_eq!(names.len(), 3, "expected 3 solid bodies, saw {names:?}");
    assert!(
        names.contains(&"Solid4") && names.contains(&"Solid5"),
        "expected Solid4 and Solid5 among {names:?}"
    );
}

#[then("the Solid Bodies folder shows two bodies")]
fn solid_bodies_shows_two_bodies(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(
        browser.bodies.len(),
        2,
        "expected 2 solid bodies after midplane split, saw {:?}",
        browser.bodies
    );
}

#[given(
    "the user creates Split1 in Solid mode using Work Plane1, the rectangular solid and the first Keep Side option"
)]
fn create_split1_solid_mode_midplane(world: &mut ApplicationWorld) {
    create_split1_midplane_solid(world);
}

#[when("the user selects the second Keep Side option and clicks OK")]
fn select_second_keep_side_and_ok(world: &mut ApplicationWorld) {
    click_split_control(world, "KeepSideA");
    accept_split_and_wait(world);
}

#[then("the second option is selected")]
fn verify_second_keep_side_selected(world: &mut ApplicationWorld) {
    let preview = world.state().split_preview.expect("expected split preview");
    assert_eq!(preview.request.keep_side, SplitKeepSide::SideA);
}

#[when("the user selects the third Keep Side option and clicks OK")]
fn select_third_keep_side_and_ok(world: &mut ApplicationWorld) {
    click_split_control(world, "KeepSideB");
    accept_split_and_wait(world);
}

#[then("the third option is selected")]
fn verify_third_keep_side_selected(world: &mut ApplicationWorld) {
    let preview = world.state().split_preview.expect("expected split preview");
    assert_eq!(preview.request.keep_side, SplitKeepSide::SideB);
}

#[then("the work plane and solid selections are still present")]
fn verify_work_plane_and_solid_retained(world: &mut ApplicationWorld) {
    let preview = world.state().split_preview.expect("expected split preview");
    assert!(matches!(
        preview.request.tool,
        Some(SplitTool::WorkPlane(_))
    ));
    assert!(preview.request.body.is_some());
    let text = super::super::world::squeezed(&world.window_text(SPLIT_WINDOW));
    assert!(
        text.contains("1WorkPlane") && text.contains("1Solid"),
        "expected retained 1 Work Plane and 1 Solid, saw {text}"
    );
}

#[given(
    "the user creates Split1 in Faces mode using the crossing sketch line and the top planar face"
)]
fn create_split1_faces_mode_top_face(world: &mut ApplicationWorld) {
    open_split_ribbon(world);
    click_visible_sketch_line_tool(world);
    click_top_planar_face(world);
    let preview = world.state().split_preview.expect("expected split preview");
    assert_eq!(preview.result, GuiSplitPreview::Ready);
    accept_split_and_wait(world);
}

#[given("Split1 is open in Faces mode with one curve and one face")]
fn reopen_split1_one_curve_one_face(world: &mut ApplicationWorld) {
    double_click_split1(world);
    let preview = world.state().split_preview.expect("expected split preview");
    assert_eq!(preview.request.mode, SplitMode::Faces);
    assert!(matches!(
        preview.request.tool,
        Some(SplitTool::SketchCurve { .. })
    ));
    assert_eq!(preview.request.faces.len(), 1);
}

#[when("the user clears the tool")]
fn clear_split_tool(world: &mut ApplicationWorld) {
    click_clear_on_split_row(world, "Tool");
}

#[then("Split requests a tool and OK is disabled")]
fn split_requests_tool_and_ok_disabled(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(SPLIT_WINDOW));
    assert!(
        text.contains("SelectTool"),
        "expected Select Tool prompt after clearing tool, saw {text}"
    );
    assert!(!world.run_is_enabled("OK"), "expected OK disabled");
}

#[then("Tool shows one curve and OK is enabled")]
fn tool_shows_one_curve_and_ok_enabled(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(SPLIT_WINDOW));
    assert!(
        text.contains("1Curve"),
        "expected 1 Curve after reselecting sketch line, saw {text}"
    );
    assert!(world.run_is_enabled("OK"), "expected OK enabled");
}

#[when("the user clears the selected face")]
fn clear_split_faces(world: &mut ApplicationWorld) {
    click_clear_on_split_row(world, "Faces");
}

#[then("Split requests faces and OK is disabled")]
fn split_requests_faces_and_ok_disabled(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(SPLIT_WINDOW));
    assert!(
        text.contains("SelectFaces"),
        "expected Select Faces prompt after clearing faces, saw {text}"
    );
    assert!(!world.run_is_enabled("OK"), "expected OK disabled");
}

#[then("the Split panel closes")]
fn split_panel_closes(_world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(SPLIT_WINDOW).is_none(),
        "expected Split panel to close"
    );
}
