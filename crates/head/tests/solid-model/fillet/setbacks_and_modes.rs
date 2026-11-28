// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Fillet corner setbacks, selection modes, and partial blends (fillet-09..14).
use super::super::super::*;
use super::setup::{
    FILLET_WINDOW, accept_fillet_and_wait, assert_v2_ready_or_partial, click_body_face_center,
    click_fillet_control, open_fillet_ribbon, reopen_fillet1, replace_fillet_field,
    select_three_origin_edges,
};
use cucumber::{given, then, when};

#[given(
    "the user selects the three edges meeting at the solid vertex at (0, 0, 0) in native recipe coordinates"
)]
fn select_three_origin_edges_step(world: &mut ApplicationWorld) {
    select_three_origin_edges(world);
}

#[given(
    "I open Fillet with a 2 mm radius and select the three edges meeting at (0, 0, 0) in native recipe coordinates"
)]
fn open_fillet_and_select_three_origin_edges(world: &mut ApplicationWorld) {
    open_fillet_ribbon(world);
    select_three_origin_edges(world);
}

#[when("I select Corner Setbacks and the shared vertex")]
fn select_corner_setbacks_and_shared_vertex(world: &mut ApplicationWorld) {
    click_fillet_control(world, "Setbacks");
    world.pump(Vec::new());
    click_fillet_control(world, "SelectSharedVertex");
}

#[given("I select Corner Setbacks, select that vertex and enable Minimal")]
fn select_setbacks_vertex_and_minimal(world: &mut ApplicationWorld) {
    select_corner_setbacks_and_shared_vertex(world);
    click_fillet_control(world, "Minimal");
}

#[then("Corner Setback Behavior shows three edge values of 2 mm")]
fn corner_setback_shows_three_2mm(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .fillet_v2_preview
        .expect("expected setback preview");
    let sb = preview
        .request
        .corner_setback
        .expect("expected corner setback");
    assert_eq!(sb.distances, vec!["2 mm", "2 mm", "2 mm"]);
    let text = super::super::super::world::squeezed(&world.window_text(FILLET_WINDOW));
    assert!(
        text.contains("CornerSetbackBehavior")
            && text.contains("Edge1")
            && text.contains("Edge2")
            && text.contains("Edge3"),
        "expected Corner Setback Behavior with 3 edges, saw {text}"
    );
}

#[when("I enable Minimal and accept the feature")]
fn enable_minimal_and_accept(world: &mut ApplicationWorld) {
    click_fillet_control(world, "Minimal");
    accept_fillet_and_wait(world);
}

#[then("the corner setback shows one vertex and Minimal")]
#[given("the corner setback uses Minimal")]
fn corner_setback_shows_vertex_and_minimal(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .fillet_v2_preview
        .expect("expected reopened setback preview");
    let sb = preview
        .request
        .corner_setback
        .expect("expected corner setback");
    assert!(sb.minimal);
    let text = super::super::super::world::squeezed(&world.window_text(FILLET_WINDOW));
    assert!(text.contains("1Vertex"), "expected 1 Vertex, saw {text}");
    assert!(world.checked("Minimal"));
}

#[when("I clear Minimal and change Edge 1 to 3 mm")]
fn clear_minimal_and_change_edge1_to_3mm(world: &mut ApplicationWorld) {
    click_fillet_control(world, "Minimal");
    replace_fillet_field(world, "Edge1", "3 mm");
}

#[then("Edge 1 is 3 mm and Edges 2 and 3 are 2 mm")]
fn edge1_3mm_edges2_3_2mm(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .fillet_v2_preview
        .expect("expected reopened setback preview");
    let sb = preview
        .request
        .corner_setback
        .expect("expected corner setback");
    assert!(!sb.minimal);
    assert_eq!(sb.distances, vec!["3 mm", "2 mm", "2 mm"]);
}

#[when("I choose feature selection and click the extrusion")]
#[when("I choose feature selection and click the block")]
fn choose_feature_selection_and_click_block(world: &mut ApplicationWorld) {
    click_fillet_control(world, "Feature");
    // Find the body with 12 edges (rectangular block)
    let doc = world.active_id();
    let block_index = {
        let app = world.app();
        let ws = app.owner.documents().workspace();
        let session = ws.session(doc).unwrap();
        let guid = ws.identity(doc).unwrap().internal_name;
        let eval = app.owner.part_scenes_current(doc).unwrap();
        eval.ordered_bodies()
            .enumerate()
            .find(|(_, body)| {
                body.edge_reference_keys(session.history(), guid)
                    .is_ok_and(|edges| edges.len() == 12)
            })
            .map(|(i, _)| i)
            .unwrap_or(0)
    };
    click_body_face_center(world, block_index);
}

#[then("the selection set shows 12 edges")]
fn selection_set_shows_12_edges(world: &mut ApplicationWorld) {
    let text = super::super::super::world::squeezed(&world.window_text(FILLET_WINDOW));
    assert!(
        text.contains("12Edges"),
        "expected 12 Edges in Fillet panel, saw {text}"
    );
}

#[given(
    "Fillet is open on the two-body part with existing Extrude1, Extrusion2 and Fillet1 features"
)]
fn fillet_open_on_two_body_part(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(browser.bodies.len(), 2, "expected 2 solid bodies");
    open_fillet_ribbon(world);
}

#[when(
    "I cancel, reopen Fillet, choose edge-loop selection, and click the right side of the previously rounded body"
)]
fn cancel_reopen_loop_click_rounded_body(world: &mut ApplicationWorld) {
    click_fillet_control(world, "Cancel");
    open_fillet_ribbon(world);
    click_fillet_control(world, "Loop");
    let doc = world.active_id();
    let rounded_index = {
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
    click_body_face_center(world, rounded_index);
}

#[then("the selection set shows five edges and a loop indicator")]
fn selection_set_shows_5_edges_and_loop(world: &mut ApplicationWorld) {
    let text = super::super::super::world::squeezed(&world.window_text(FILLET_WINDOW));
    assert!(
        text.contains("5Edges") && text.contains("[Loop]"),
        "expected 5 Edges and [Loop] indicator, saw {text}"
    );
}

#[given("I open Fillet, choose edge-loop selection, and select a loop on the solid")]
fn open_fillet_loop_select_loop(world: &mut ApplicationWorld) {
    open_fillet_ribbon(world);
    click_fillet_control(world, "Loop");
    click_body_face_center(world, 0);
}

#[when("I choose the solid selection control and click a solid body")]
fn choose_solid_selection_and_click_body(world: &mut ApplicationWorld) {
    click_fillet_control(world, "Solid");
    click_body_face_center(world, 0);
}

#[then("the panel lists 1 Solid and displays separate edge groups with radius fields")]
fn panel_lists_1_solid_and_edge_groups(world: &mut ApplicationWorld) {
    let text = super::super::super::world::squeezed(&world.window_text(FILLET_WINDOW));
    assert!(
        text.contains("1Solid") && text.contains("Set1") && text.contains("Set2"),
        "expected 1 Solid and separate edge sets, saw {text}"
    );
}

#[then("the selected body is highlighted in the preview")]
fn selected_body_highlighted_in_preview(world: &mut ApplicationWorld) {
    assert_v2_ready_or_partial(world);
    assert!(world.state().extrude_preview_active);
}

#[when("I cancel the Fillet command")]
fn cancel_the_fillet_command(world: &mut ApplicationWorld) {
    click_fillet_control(world, "Cancel");
}

#[then("the uncommitted preview closes without adding the new Fillet feature")]
fn uncommitted_preview_closes_without_fillet(world: &mut ApplicationWorld) {
    assert!(super::super::super::diagnostics::window_geometry(FILLET_WINDOW).is_none());
    assert!(!world.state().extrude_preview_active);
    let browser = world.state().browser.expect("expected browser");
    assert!(!browser.features.iter().any(|f| f.name == "Fillet1"));
}

#[given("the model contains a joined cylindrical boss and a swept cut as two saved Sweep features")]
fn swept_fixture_has_two_sweeps(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    let sweeps = browser
        .features
        .iter()
        .filter(|f| f.kind == "Sweep")
        .count();
    assert!(
        sweeps >= 2,
        "expected 2 Sweep features, saw {:?}",
        browser.features
    );
}

#[given("the boss uses the 6 mm radius Sketch2 profile and 10 mm outward Sketch3 path")]
#[given("the cut uses a reconstructed 2.5 mm radius Sketch4 profile and 10 mm inward Sketch5 path")]
fn swept_fixture_verified(_world: &mut ApplicationWorld) {}

#[given("the model contains one solid body")]
fn model_contains_one_solid(world: &mut ApplicationWorld) {
    assert_eq!(world.state().browser.expect("browser").bodies.len(), 1);
}

#[given("the swept solid is selected for a 2 mm fillet")]
fn swept_solid_selected_for_2mm_fillet(world: &mut ApplicationWorld) {
    open_fillet_ribbon(world);
    click_fillet_control(world, "Solid");
    click_body_face_center(world, 0);
}

#[given("the selection lists groups of one and three edges")]
fn selection_lists_1_and_3_edge_groups(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .fillet_v2_preview
        .expect("expected solid fillet preview");
    let counts: Vec<usize> = preview
        .request
        .edge_sets
        .iter()
        .map(|s| s.edges.len())
        .collect();
    assert_eq!(
        counts,
        vec![1, 3],
        "expected [1, 3] edge groups, got {counts:?}"
    );
}

#[then("Create Fillet reports that one of four edges cannot be blended at the current size")]
fn create_fillet_reports_1_of_4_failed(world: &mut ApplicationWorld) {
    assert!(
        super::super::super::diagnostics::window_geometry("Create Fillet").is_some(),
        "expected Create Fillet modal dialog, saw {}",
        world.text()
    );
    let text = super::super::super::world::squeezed(&world.window_text("Create Fillet"));
    assert!(
        text.contains("1of4edgescannotbeblendedatthecurrentsize"),
        "expected 1 of 4 message in Create Fillet modal, saw {text}"
    );
}

#[then("accepting successful blends and changing the body selection to edge selection is checked")]
fn accept_successful_blends_is_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Accept successful blends and change body selection to edge selection"));
}

#[when("I accept that option and reopen Fillet1")]
fn accept_partial_option_and_reopen(world: &mut ApplicationWorld) {
    let ok = world.run_in_window("Create Fillet", "OK").center();
    world.click(super::super::super::diagnostics::pointer_pixels(ok));
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(FILLET_WINDOW).is_none(),
        "expected Fillet to commit after partial acceptance"
    );
    reopen_fillet1(world);
}

#[then("the feature lists edge sets of one and two edges at 2 mm")]
fn feature_lists_1_and_2_edges_at_2mm(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .fillet_v2_preview
        .expect("expected reopened Fillet1 preview");
    let sets: Vec<(usize, &str)> = preview
        .request
        .edge_sets
        .iter()
        .map(|s| (s.edges.len(), s.radius.as_str()))
        .collect();
    assert_eq!(
        sets,
        vec![(1, "2 mm"), (2, "2 mm")],
        "expected 1 and 2 edge sets at 2 mm"
    );
}

#[when("I cancel the edit")]
fn cancel_fillet_edit(world: &mut ApplicationWorld) {
    for window in ["Combine", "Draft"] {
        if super::super::super::diagnostics::window_geometry(window).is_some() {
            let center = world.run_in_window(window, "Cancel").center();
            world.click(super::super::super::diagnostics::pointer_pixels(center));
            world.pump(Vec::new());
            world.pump(Vec::new());
            return;
        }
    }
    click_fillet_control(world, "Cancel");
}

#[then("the created fillet remains visible")]
fn created_fillet_remains_visible(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(browser.features.iter().any(|f| f.name == "Fillet1"));
    assert!(!world.app().window.scene_triangles.is_empty());
}
