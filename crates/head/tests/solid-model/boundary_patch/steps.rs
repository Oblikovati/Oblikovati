// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Surface Boundary Patch scenarios (boundary-patch-01..06, #3918).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiBoundaryCondition;
use setup::{
    PATCH_WINDOW, add_visible_diagonal_to_sketch1, click_condition_icon, click_model_point,
    click_patch_control, create_zero_offset_surface_on_face, double_click_boundary_patch1,
    ensure_sketch_visible, enter_weight_value, frame_block_back_iso, frame_block_front_iso,
    frame_sketch9_clear_of_panel, frame_top_clear_of_panel, has_ready_preview,
    has_warm_amber_preview_triangles, open_boundary_patch_ribbon, open_fixture_copy,
};

#[given(
    "a disposable copy of the native fixture \"fixtures/cad/revolve/revolve-profile.opd\" is open with its rectangular sketch visible"
)]
fn open_revolve_profile_rectangular(world: &mut ApplicationWorld) {
    open_fixture_copy(world, "fixtures/cad/revolve/revolve-profile.opd");
    let browser = world.state().browser.expect("expected Model browser");
    assert!(
        browser
            .sketches
            .iter()
            .any(|s| s.name == "Sketch1" && s.visible),
        "expected visible Sketch1 in revolve-profile.opd"
    );
    frame_top_clear_of_panel(world);
}

#[given("Boundary Patch is open with Automatic Edge Chain enabled")]
fn boundary_patch_open_auto_chain(world: &mut ApplicationWorld) {
    open_boundary_patch_ribbon(world);
    let preview = world
        .state()
        .boundary_patch_preview
        .expect("expected boundary patch preview state");
    assert!(
        preview.request.automatic_edge_chain,
        "expected Automatic Edge Chain enabled by default"
    );
}

#[when("I select an edge of the rectangular boundary")]
fn select_edge_of_rectangular_boundary(world: &mut ApplicationWorld) {
    // Lower edge of revolve-profile.opd rectangle: (-0.6, 0.0) -> (0.6, 0.0).
    click_model_point(world, [0.0, 0.0, 0.0]);
}

#[then("Edges1 reports four selected edges and OK and Apply become available")]
fn edges1_four_edges_ok_apply_available(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .boundary_patch_preview
        .expect("expected boundary patch preview");
    assert_eq!(preview.request.edge_sets.len(), 1);
    assert_eq!(preview.request.edge_sets[0].edges.len(), 4);
    assert!(has_ready_preview(world));
    let text = super::super::world::squeezed(&world.window_text(PATCH_WINDOW));
    assert!(
        text.contains("Edges1:4Edges"),
        "expected Edges1: 4 Edges in panel, saw {text}"
    );
    assert!(world.run_is_enabled("OK"), "expected OK enabled");
    assert!(world.run_is_enabled("Apply"), "expected Apply enabled");
}

#[given("the four-edge boundary is selected in Boundary Patch")]
fn four_edge_boundary_selected(world: &mut ApplicationWorld) {
    boundary_patch_open_auto_chain(world);
    select_edge_of_rectangular_boundary(world);
    edges1_four_edges_ok_apply_available(world);
}

#[when("I choose the second condition icon and enter a weight of 0.7")]
fn choose_second_condition_and_weight(world: &mut ApplicationWorld) {
    click_condition_icon(world, 2);
    enter_weight_value(world, "0.7");
}

#[then("the condition icon and weight field reflect those choices")]
fn condition_and_weight_reflected(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .boundary_patch_preview
        .expect("expected boundary patch preview");
    assert_eq!(
        preview.request.edge_sets[0].condition,
        GuiBoundaryCondition::Tangent
    );
    assert!((preview.request.edge_sets[0].weight - 0.7).abs() < 1e-6);
    let text = super::super::world::squeezed(&world.window_text(PATCH_WINDOW));
    assert!(
        text.contains("0.7"),
        "expected weight 0.7 displayed in Boundary Patch panel, saw {text}"
    );
}

#[given(
    "a disposable copy of `fixtures/cad/boundary-patch/six-solids-five-loops-candidate.opd` is open with Sketch9 visible"
)]
#[given(
    "a disposable copy of the native fixture \"fixtures/cad/boundary-patch/six-solids-five-loops-candidate.opd\" is open with Sketch9 visible"
)]
fn open_six_solids_five_loops(world: &mut ApplicationWorld) {
    open_fixture_copy(
        world,
        "fixtures/cad/boundary-patch/six-solids-five-loops-candidate.opd",
    );
    ensure_sketch_visible(world, "Sketch9");
    let browser = world.state().browser.expect("expected Model browser");
    assert_eq!(browser.bodies.len(), 6, "expected six solid bodies");
    assert!(
        browser
            .sketches
            .iter()
            .any(|s| s.name == "Sketch9" && s.visible),
        "expected Sketch9 visible"
    );
    frame_sketch9_clear_of_panel(world);
}

#[given(
    "the user creates four separate surface patches from the three upper rectangular loops and the lower-left loop of Sketch9"
)]
fn create_four_patches_on_sketch9(world: &mut ApplicationWorld) {
    // Midpoints of lower edge of each loop in Sketch9 (XY plane, z = 0):
    // upper-left, upper-mid, upper-right, lower-left.
    let loop_points = [
        [-2.2, 0.8, 0.0],
        [0.0, 0.8, 0.0],
        [2.2, 0.8, 0.0],
        [-1.1, -0.8, 0.0],
    ];
    for point in loop_points {
        ensure_sketch_visible(world, "Sketch9");
        open_boundary_patch_ribbon(world);
        let px = crate::HeadWindow::model_pixel(&world.app().window.native, point).unwrap();
        click_model_point(world, point);
        assert!(
            has_ready_preview(world),
            "expected ready preview for {point:?} at px={px:?}, preview={:?}",
            world.state().boundary_patch_preview
        );
        click_patch_control(world, "OK");
        assert!(
            super::super::diagnostics::window_geometry(PATCH_WINDOW).is_none(),
            "expected Boundary Patch panel to close after commit"
        );
    }
    let browser = world.state().browser.expect("expected Model browser");
    assert_eq!(
        browser.surface_bodies.len(),
        4,
        "expected 4 surface bodies before fifth patch"
    );
}

#[given("Sketch9 is made visible again, leaving the lower-right rectangular loop available")]
fn make_sketch9_visible_again(world: &mut ApplicationWorld) {
    ensure_sketch_visible(world, "Sketch9");
}

#[given("Boundary Patch has automatically chained all four edges of that lower-right loop")]
fn chain_lower_right_loop(world: &mut ApplicationWorld) {
    open_boundary_patch_ribbon(world);
    // Lower-right loop of Sketch9: [0.5..1.7, -0.8..-0.3], lower-edge midpoint [1.1, -0.8, 0.0].
    click_model_point(world, [1.1, -0.8, 0.0]);
    let preview = world
        .state()
        .boundary_patch_preview
        .expect("expected boundary patch preview");
    assert_eq!(preview.request.edge_sets.len(), 1);
    assert_eq!(preview.request.edge_sets[0].edges.len(), 4);
    assert!(has_ready_preview(world));
}

#[given("the second boundary condition is selected with weight 0.7")]
fn second_condition_selected_with_weight_0_7(world: &mut ApplicationWorld) {
    choose_second_condition_and_weight(world);
    condition_and_weight_reflected(world);
}

#[then(
    "the dialog closes and Srf5 appears under Surface Bodies, increasing the count from four to five"
)]
fn dialog_closes_and_srf5_appears(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(PATCH_WINDOW).is_none(),
        "expected Boundary Patch dialog closed after OK"
    );
    let browser = world.state().browser.expect("expected Model browser");
    assert_eq!(
        browser.surface_bodies.len(),
        5,
        "expected 5 surface bodies, saw {:?}",
        browser.surface_bodies
    );
    assert!(
        browser.surface_bodies.iter().any(|s| s.name == "Srf5"),
        "expected Srf5 under Surface Bodies, saw {:?}",
        browser.surface_bodies
    );
    let model = super::super::world::squeezed(&world.window_text("Model"));
    assert!(
        model.contains("SurfaceBodies(5)") && model.contains("Srf5"),
        "expected Surface Bodies (5) and Srf5 in Model browser, saw {model}"
    );
}

#[given(
    "the user creates four separate zero-offset surfaces from four distinct planar block faces"
)]
fn create_four_zero_offset_surfaces(world: &mut ApplicationWorld) {
    // Four distinct planar faces of Extrusion1 block ([-4..0, -2.5..0, 0..2]):
    // top (z=2), front (y=-2.5), right (x=0), and left (x=-4).
    frame_block_front_iso(world);
    for point in [[-2.0, -1.25, 2.0], [-2.0, -2.5, 1.0], [0.0, -1.25, 1.0]] {
        create_zero_offset_surface_on_face(world, point);
    }
    frame_block_back_iso(world);
    create_zero_offset_surface_on_face(world, [-4.0, -1.25, 1.0]);
    let browser = world.state().browser.expect("expected Model browser");
    assert_eq!(
        browser.surface_bodies.len(),
        4,
        "expected 4 zero-offset surfaces before Boundary Patch1"
    );
}

#[given(
    "the user creates Boundary Patch1 from the lower-right four-edge loop of Sketch9 using the second condition and weight 0.7"
)]
fn create_boundary_patch1_lower_right(world: &mut ApplicationWorld) {
    ensure_sketch_visible(world, "Sketch9");
    frame_sketch9_clear_of_panel(world);
    chain_lower_right_loop(world);
    second_condition_selected_with_weight_0_7(world);
    click_patch_control(world, "OK");
    assert!(super::super::diagnostics::window_geometry(PATCH_WINDOW).is_none());
}

#[given("Boundary Patch1 retains that condition and weight")]
fn boundary_patch1_retains_condition_and_weight(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected Model browser");
    let feat = browser
        .features
        .iter()
        .find(|f| f.name == "Boundary Patch1")
        .expect("expected Boundary Patch1 in browser");
    let seed = browser
        .boundary_patch_edits
        .get(&feat.id)
        .expect("expected Boundary Patch1 edit seed");
    assert_eq!(seed.edge_sets[0].condition, GuiBoundaryCondition::Tangent);
    assert!((seed.edge_sets[0].weight - 0.7).abs() < 1e-6);
}

#[when("I edit it, choose the first condition icon, and cancel")]
fn edit_patch1_choose_first_and_cancel(world: &mut ApplicationWorld) {
    double_click_boundary_patch1(world);
    click_condition_icon(world, 1);
    let preview = world
        .state()
        .boundary_patch_preview
        .expect("expected edited preview before cancel");
    assert_eq!(
        preview.request.edge_sets[0].condition,
        GuiBoundaryCondition::Free
    );
    click_patch_control(world, "Cancel");
    assert!(
        super::super::diagnostics::window_geometry(PATCH_WINDOW).is_none(),
        "expected Boundary Patch panel closed after Cancel"
    );
}

#[then("reopening it shows the second condition icon and weight 0.7")]
fn reopening_shows_second_condition_and_weight(world: &mut ApplicationWorld) {
    double_click_boundary_patch1(world);
    condition_and_weight_reflected(world);
}

#[given("a visible diagonal crosses the area inside the boundary")]
fn visible_diagonal_crosses_boundary(world: &mut ApplicationWorld) {
    add_visible_diagonal_to_sketch1(world);
    frame_top_clear_of_panel(world);
}

#[given("a new Boundary Patch is open with Automatic Edge Chain cleared")]
fn open_boundary_patch_auto_chain_cleared(world: &mut ApplicationWorld) {
    open_boundary_patch_ribbon(world);
    click_patch_control(world, "Automatic Edge Chain");
    let preview = world
        .state()
        .boundary_patch_preview
        .expect("expected boundary patch preview");
    assert!(
        !preview.request.automatic_edge_chain,
        "expected Automatic Edge Chain cleared"
    );
}

#[when("I select the lower edge")]
fn select_lower_edge(world: &mut ApplicationWorld) {
    click_model_point(world, [0.0, 0.0, 0.0]);
}

#[then("Edges1 shows one selected edge and OK and Apply remain disabled")]
#[then("Edges1 still shows one selected edge and OK and Apply remain disabled")]
fn edges1_one_edge_ok_apply_disabled(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .boundary_patch_preview
        .expect("expected boundary patch preview");
    assert_eq!(preview.request.edge_sets.len(), 1);
    assert_eq!(preview.request.edge_sets[0].edges.len(), 1);
    let text = super::super::world::squeezed(&world.window_text(PATCH_WINDOW));
    assert!(
        text.contains("Edges1:1Edge"),
        "expected Edges1: 1 Edge in panel, saw {text}"
    );
    assert!(!world.run_is_enabled("OK"), "expected OK disabled");
    assert!(!world.run_is_enabled("Apply"), "expected Apply disabled");
}

#[when("I click the visible diagonal")]
fn click_visible_diagonal(world: &mut ApplicationWorld) {
    // Midpoint of diagonal (-0.6, 0.0) -> (0.6, 0.5) is (0.0, 0.25, 0.0).
    click_model_point(world, [0.0, 0.25, 0.0]);
}

#[when("I select the right edge")]
fn select_right_edge(world: &mut ApplicationWorld) {
    // Midpoint of right edge (0.6, 0.0) -> (0.6, 0.5) is (0.6, 0.25, 0.0).
    click_model_point(world, [0.6, 0.25, 0.0]);
}

#[then("Edges2 shows one selected edge, a preview appears, and OK and Apply become available")]
fn edges2_one_edge_preview_and_ok_apply(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .boundary_patch_preview
        .expect("expected boundary patch preview");
    assert_eq!(preview.request.edge_sets.len(), 2);
    assert_eq!(preview.request.edge_sets[1].edges.len(), 1);
    assert!(has_ready_preview(world));
    assert!(
        has_warm_amber_preview_triangles(world),
        "expected translucent Warm-Amber surface preview in viewport"
    );
    let text = super::super::world::squeezed(&world.window_text(PATCH_WINDOW));
    assert!(
        text.contains("Edges2:1Edge"),
        "expected Edges2: 1 Edge in panel, saw {text}"
    );
    assert!(world.run_is_enabled("OK"), "expected OK enabled");
    assert!(world.run_is_enabled("Apply"), "expected Apply enabled");
}

#[given("the user selects the lower and right edges and a Boundary Patch preview is visible")]
fn select_lower_and_right_edges_with_preview(world: &mut ApplicationWorld) {
    select_lower_edge(world);
    select_right_edge(world);
    edges2_one_edge_preview_and_ok_apply(world);
}

#[when("I activate Guide Rails and select the upper edge")]
fn activate_guide_rails_and_select_upper(world: &mut ApplicationWorld) {
    click_patch_control(world, "Guide Rails");
    // Midpoint of upper edge (0.6, 0.5) -> (-0.6, 0.5) is (0.0, 0.5, 0.0).
    click_model_point(world, [0.0, 0.5, 0.0]);
}

#[then("a 1. Edge selection indicator appears")]
fn guide_rail_indicator_appears(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .boundary_patch_preview
        .expect("expected boundary patch preview");
    assert_eq!(preview.request.guide_rails.len(), 1);
    let text = super::super::world::squeezed(&world.window_text(PATCH_WINDOW));
    assert!(
        text.contains("1.Edge"),
        "expected 1. Edge indicator in panel, saw {text}"
    );
}

#[when("I cancel the trial")]
fn cancel_the_trial(world: &mut ApplicationWorld) {
    click_patch_control(world, "Cancel");
}

#[then("the trial is canceled")]
fn trial_is_canceled(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(PATCH_WINDOW).is_none(),
        "expected Boundary Patch panel closed"
    );
    assert!(
        world.state().boundary_patch_preview.is_none(),
        "expected preview cleared on Cancel"
    );
    let browser = world.state().browser.expect("expected Model browser");
    assert!(
        browser.surface_bodies.is_empty(),
        "expected no surface body committed after Cancel"
    );
}
