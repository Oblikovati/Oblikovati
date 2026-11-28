// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Part Surface Stitch scenarios (stitch-01..05).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiStitchPreview;
use setup::{
    STITCH_WINDOW, click_browser_surface, click_model_point, click_stitch_control,
    create_zero_offset_surface_on_point, double_click_stitch_feature, expect_ready_preview,
    hide_solid_body_in_browser, open_stitch_ribbon, pan_clear_of_panel, replace_stitch_field,
};

#[given(
    "the user creates separate zero-offset surface bodies from two adjacent planar faces of the block"
)]
fn create_two_adjacent_zero_offset_surfaces(world: &mut ApplicationWorld) {
    // Block in body-properties-start.opd spans [-4..0, -2.5..0, 0..2.0].
    // Create Srf1 on top face (+Z at z=2.0) and Srf2 on adjacent +X face (x=0.0).
    create_zero_offset_surface_on_point(world, [-2.0, -1.25, 2.0]);
    create_zero_offset_surface_on_point(world, [0.0, -1.25, 1.0]);
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(
        browser.surface_bodies.len(),
        2,
        "expected 2 zero-offset surface bodies, saw {:?}",
        browser.surface_bodies
    );
}

#[given("the user hides the original solid so the two surfaces are selectable")]
fn hide_original_solid(world: &mut ApplicationWorld) {
    hide_solid_body_in_browser(world, "Solid1");
    let browser = world.state().browser.expect("expected browser");
    assert!(
        !browser.bodies[0].visible,
        "expected Solid1 hidden, notice={:?}, saw {:?}",
        world.app().window.native.command_notice,
        browser.bodies
    );
}

#[given("Stitch is open without selected surfaces")]
fn stitch_open_without_selected_surfaces(world: &mut ApplicationWorld) {
    open_stitch_ribbon(world);
    let text = super::super::world::squeezed(&world.window_text(STITCH_WINDOW));
    assert!(
        text.contains("0Selected"),
        "expected 0 Selected, saw {text}"
    );
}

#[when("the user selects one surface")]
fn select_one_surface(world: &mut ApplicationWorld) {
    pan_clear_of_panel(world);
    // Pick Srf2 on the +X face at [0.0, -1.25, 1.0]; fallback to browser Srf2 if occluded.
    let _ = click_model_point(world, [0.0, -1.25, 1.0]);
    let text = super::super::world::squeezed(&world.window_text(STITCH_WINDOW));
    if !text.contains("1Selected") {
        click_browser_surface(world, "Srf2");
    }
}

#[then("the count is one and Apply remains disabled")]
fn count_is_one_and_apply_disabled(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(STITCH_WINDOW));
    assert!(
        text.contains("1Selected"),
        "expected 1 Selected, saw {text}"
    );
    assert!(
        !world.run_is_enabled("Apply"),
        "expected Apply disabled with 1 surface"
    );
}

#[when("the user selects Srf1 as a second surface")]
fn select_srf1_as_second_surface(world: &mut ApplicationWorld) {
    click_browser_surface(world, "Srf1");
}

#[then("the count is two and Apply becomes available")]
fn count_is_two_and_apply_available(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(STITCH_WINDOW));
    assert!(
        text.contains("2Selected"),
        "expected 2 Selected, saw {text}"
    );
    expect_ready_preview(world);
    assert!(
        world.run_is_enabled("Apply"),
        "expected Apply enabled with 2 surfaces"
    );
}

#[given("the user opens Stitch and its Analyze tab")]
fn open_stitch_and_analyze_tab(world: &mut ApplicationWorld) {
    open_stitch_ribbon(world);
    click_stitch_control(world, "Analyze");
}

#[when("the user enables Show Edge Conditions and Show Near Tangent")]
fn enable_edge_conditions_and_near_tangent(world: &mut ApplicationWorld) {
    click_stitch_control(world, "ShowEdgeConditions");
    click_stitch_control(world, "ShowNearTangent");
}

#[then("both options are checked and their color legends remain visible")]
fn both_analyze_options_checked_with_legends(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .stitch_preview
        .expect("expected stitch preview state on Analyze tab");
    assert!(
        preview.request.show_edge_conditions,
        "expected Show Edge Conditions checked"
    );
    assert!(
        preview.request.show_near_tangent,
        "expected Show Near Tangent checked"
    );
    let text = super::super::world::squeezed(&world.window_text(STITCH_WINDOW));
    for legend in ["FreeEdges", "StitchedEdges", "NearTangent"] {
        assert!(
            text.contains(legend),
            "expected color legend {legend} visible in Stitch Analyze tab, saw {text}"
        );
    }
}

#[given("the part contains three surface bodies and a separate rounded body")]
fn part_has_three_surfaces_and_rounded_solid(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(
        browser.surface_bodies.len(),
        3,
        "expected 3 surface bodies, saw {:?}",
        browser.surface_bodies
    );
    assert_eq!(
        browser.bodies.len(),
        1,
        "expected 1 rounded solid body, saw {:?}",
        browser.bodies
    );
}

#[given("the user hides the rounded body so the surfaces are selectable")]
fn hide_rounded_body(world: &mut ApplicationWorld) {
    hide_solid_body_in_browser(world, "Solid1");
    let browser = world.state().browser.expect("expected browser");
    assert!(!browser.bodies[0].visible, "expected Solid1 hidden");
}

#[given(
    "the user opens Stitch and selects the two coaxial surfaces centered at X = -100 mm, spanning Z = 0 to 10 mm and Z = 10 to 20 mm"
)]
fn open_stitch_and_select_coaxial_surfaces(world: &mut ApplicationWorld) {
    open_stitch_ribbon(world);
    // In adjacent-pair-three-surfaces-candidate.opd:
    // Srf1 = SweepSrf1 (X=-10 cm, Z=0..1 cm), Srf2 = SweepSrf3 (X=-10 cm, Z=1..2 cm).
    click_browser_surface(world, "Srf1");
    click_browser_surface(world, "Srf2");
    let text = super::super::world::squeezed(&world.window_text(STITCH_WINDOW));
    assert!(
        text.contains("2Selected"),
        "expected 2 coaxial surfaces selected, saw {text}"
    );
}

#[given("Maintain as surface is checked")]
fn maintain_as_surface_checked(world: &mut ApplicationWorld) {
    let checked = world
        .state()
        .stitch_preview
        .is_some_and(|p| p.request.maintain_as_surface);
    if !checked {
        click_stitch_control(world, "Maintainassurface");
    }
    expect_ready_preview(world);
}

#[then("Stitch Surface1 and Srf4 appear and the surface body count increases from three to four")]
fn stitch_surface1_and_srf4_appear(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "Stitch Surface1"),
        "expected Stitch Surface1 in browser: {:?}",
        browser.features
    );
    assert_eq!(
        browser.surface_bodies.len(),
        4,
        "expected surface body count to increase from 3 to 4, saw {:?}",
        browser.surface_bodies
    );
    assert!(
        browser.surface_bodies.iter().any(|b| b.name == "Srf4"),
        "expected Srf4 in surface bodies: {:?}",
        browser.surface_bodies
    );
}

#[then("remaining free edges are listed")]
#[then("the remaining free-edge list is populated.")]
fn remaining_free_edges_are_listed(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(STITCH_WINDOW));
    assert!(
        text.contains("RemainingFreeEdges") && text.contains("Edge1"),
        "expected Remaining Free Edges populated with Edge 1, saw {text}"
    );
}

#[given(
    "the user opens Stitch, selects both surfaces, checks Maintain as surface and clicks Apply"
)]
fn open_stitch_select_both_maintain_and_apply(world: &mut ApplicationWorld) {
    open_stitch_ribbon(world);
    click_browser_surface(world, "Srf1");
    click_browser_surface(world, "Srf2");
    maintain_as_surface_checked(world);
    click_stitch_control(world, "Apply");
}

#[given("the Stitch dialog remains open")]
fn stitch_dialog_remains_open(_world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(STITCH_WINDOW).is_some(),
        "expected Stitch dialog to remain open after Apply"
    );
}

#[given("the user creates Stitch Surface1 from both surfaces with Maintain as surface checked")]
fn create_stitch_surface1_from_both(world: &mut ApplicationWorld) {
    open_stitch_select_both_maintain_and_apply(world);
    click_stitch_control(world, "Done");
    assert!(super::super::diagnostics::window_geometry(STITCH_WINDOW).is_none());
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "Stitch Surface1"),
        "expected Stitch Surface1 created: {:?}",
        browser.features
    );
}

#[when("the user reopens it, enters 0.5 mm for Maximum Tolerance, and clicks Apply")]
fn reopen_stitch_set_05mm_and_apply(world: &mut ApplicationWorld) {
    double_click_stitch_feature(world, "Stitch Surface1");
    replace_stitch_field(world, "MaximumTolerance", "0.5 mm");
    expect_ready_preview(world);
    click_stitch_control(world, "Apply");
}

#[when("the user clicks Done and reopens Stitch Surface1")]
fn click_done_and_reopen_stitch_surface1(world: &mut ApplicationWorld) {
    click_stitch_control(world, "Done");
    assert!(super::super::diagnostics::window_geometry(STITCH_WINDOW).is_none());
    double_click_stitch_feature(world, "Stitch Surface1");
}

#[then("the dialog displays 0.500 mm and 2 Selected.")]
fn dialog_displays_0500_mm_and_2_selected(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(STITCH_WINDOW));
    assert!(
        text.contains("0.500mm") && text.contains("2Selected"),
        "expected 0.500 mm and 2 Selected in Stitch dialog, saw {text}"
    );
    let preview = world
        .state()
        .stitch_preview
        .expect("expected stitch preview after reopen");
    assert!(
        matches!(preview.result, GuiStitchPreview::Ready { .. }),
        "expected Ready stitch preview after reopen, got {preview:?}"
    );
}
