// SPDX-License-Identifier: GPL-2.0-only
//! Desktop BDD bindings for Part Create Unwrap scenarios (unwrap-01..unwrap-06).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiUnwrapPreview;
use oblikovati_model::feature::unwrap::UnwrapAlignment;
use setup::{
    UNWRAP_WINDOW, click_unwrap_control, click_unwrap_ribbon, click_unwrap_row_button,
    open_disposable_unwrap_fixture, scene_bounds, select_lateral_face,
};

#[given("the part is displayed in the 3D Model environment")]
fn part_displayed_in_3d_model(world: &mut ApplicationWorld) {
    assert!(!world.app().window.scene_triangles.is_empty());
    assert!(
        world.shows("Unwrap"),
        "expected Unwrap button on 3D Model ribbon"
    );
}

#[when("Unwrap is opened")]
fn unwrap_is_opened(world: &mut ApplicationWorld) {
    click_unwrap_ribbon(world);
    assert!(
        super::super::diagnostics::window_geometry(UNWRAP_WINDOW).is_some(),
        "expected Unwrap panel to open, saw {}",
        world.text()
    );
}

#[then("Faces is empty and OK is disabled")]
fn faces_empty_ok_disabled(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(UNWRAP_WINDOW));
    assert!(
        text.contains("SelectFaces"),
        "expected empty Faces selector, saw {text}"
    );
    assert!(
        world.run_in_window(UNWRAP_WINDOW, "OK").alpha < 200,
        "expected OK disabled when Faces is empty"
    );
}

#[then("Auto Face Chain and Merge Result Body are checked")]
fn auto_chain_and_merge_checked(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .unwrap_preview
        .expect("expected unwrap preview state");
    assert!(preview.request.auto_face_chain);
    assert!(preview.request.merge_result_body);
}

#[given(expr = "a disposable copy of `{word}` is open in 3D Model")]
fn disposable_backtick_in_3d_model(world: &mut ApplicationWorld, relative: String) {
    open_disposable_unwrap_fixture(world, &relative);
}

#[given("the user opens Unwrap with Auto Face Chain enabled")]
fn user_opens_unwrap_with_chain(world: &mut ApplicationWorld) {
    unwrap_is_opened(world);
    let preview = world
        .state()
        .unwrap_preview
        .expect("expected unwrap preview state");
    assert!(preview.request.auto_face_chain);
}

#[when("the user selects a lateral face of the prism")]
fn user_selects_prism_lateral_face(world: &mut ApplicationWorld) {
    select_lateral_face(world);
}

#[then("Faces shows 45 Faces and the MCAD Software displays an unwrap preview.")]
fn faces_shows_45_and_preview(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(UNWRAP_WINDOW));
    assert!(
        text.contains("45Faces"),
        "expected 45 Faces in Unwrap panel, saw {text}"
    );
    assert!(
        world.state().extrude_preview_active,
        "expected live unwrap preview in viewport"
    );
}

#[then("the alignment vertex and edge selection controls appear.")]
fn alignment_vertex_and_edge_controls_appear(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(UNWRAP_WINDOW));
    for needle in [
        "AlignmentVertex",
        "1Vertex",
        "LinearResult",
        "RigidResult",
        "SeamEdges",
    ] {
        assert!(
            text.contains(needle),
            "expected {needle:?} in Unwrap panel, saw {text}"
        );
    }
}

#[given("Unwrap is open with Auto Face Chain disabled and the face selection cleared")]
fn unwrap_open_chain_disabled_cleared(world: &mut ApplicationWorld) {
    unwrap_is_opened(world);
    click_unwrap_control(world, "AutoFaceChain");
    let preview = world
        .state()
        .unwrap_preview
        .expect("expected unwrap preview state");
    assert!(!preview.request.auto_face_chain);
    assert!(preview.request.faces.is_empty());
}

#[when("the user selects the cylindrical side face")]
fn user_selects_cylindrical_side_face(world: &mut ApplicationWorld) {
    select_lateral_face(world);
}

#[then("Faces shows 1 Face and OK is enabled.")]
fn faces_shows_1_face_ok_enabled(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(UNWRAP_WINDOW));
    assert!(
        text.contains("1Face"),
        "expected 1 Face in Unwrap panel, saw {text}"
    );
    let preview = world
        .state()
        .unwrap_preview
        .expect("expected unwrap preview state");
    assert_eq!(preview.result, GuiUnwrapPreview::Ready);
    assert!(
        world.run_in_window(UNWRAP_WINDOW, "OK").alpha >= 200,
        "expected OK enabled once a face is selected"
    );
}

#[given("the cylindrical side face is selected in Unwrap")]
fn cylindrical_side_face_selected_in_unwrap(world: &mut ApplicationWorld) {
    unwrap_open_chain_disabled_cleared(world);
    select_lateral_face(world);
    faces_shows_1_face_ok_enabled(world);
}

#[when("the user chooses different Alignment buttons")]
fn user_chooses_different_alignment_buttons(world: &mut ApplicationWorld) {
    let before_bounds = scene_bounds(world);
    click_unwrap_control(world, "XZ");
    let xz_bounds = scene_bounds(world);
    assert_ne!(
        before_bounds, xz_bounds,
        "expected XZ Alignment button to change preview orientation"
    );
    click_unwrap_control(world, "YZ");
    let yz_bounds = scene_bounds(world);
    assert_ne!(
        xz_bounds, yz_bounds,
        "expected YZ Alignment button to change preview orientation"
    );
}

#[then("the displayed unwrap preview changes orientation.")]
fn unwrap_preview_changed_orientation(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .unwrap_preview
        .expect("expected unwrap preview state");
    assert_eq!(preview.request.alignment, UnwrapAlignment::OriginYZ);
    assert!(world.state().extrude_preview_active);
}

#[given("the cylindrical side face is selected in Unwrap and Merge Result Body is cleared")]
fn cylindrical_face_selected_merge_cleared(world: &mut ApplicationWorld) {
    cylindrical_side_face_selected_in_unwrap(world);
    click_unwrap_control(world, "MergeResultBody");
    let preview = world
        .state()
        .unwrap_preview
        .expect("expected unwrap preview state");
    assert!(!preview.request.merge_result_body);
}

#[then("the panel closes, Unwrap1 appears in the browser, and the browser shows one surface body.")]
fn panel_closes_unwrap1_and_surface_body(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry(UNWRAP_WINDOW).is_none());
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser
            .features
            .iter()
            .any(|f| f.name == "Unwrap1" && f.kind == "Unwrap"),
        "expected Unwrap1 in browser, saw {:?}",
        browser.features
    );
    assert_eq!(
        browser.surface_bodies.len(),
        1,
        "expected 1 surface body in browser, saw {:?}",
        browser.surface_bodies
    );
    let model = super::super::world::squeezed(&world.window_text("Model"));
    assert!(
        model.contains("SurfaceBodies(1)"),
        "expected Surface Bodies (1) folder in Model browser, saw {model}"
    );
}

#[given(
    "the user opens Unwrap with Auto Face Chain enabled and selects a lateral face of the prism"
)]
fn open_unwrap_chain_and_select_lateral(world: &mut ApplicationWorld) {
    user_opens_unwrap_with_chain(world);
    select_lateral_face(world);
    faces_shows_45_and_preview(world);
}

#[given("the user disables Auto Face Chain, clears the selection and reselects that lateral face")]
fn disable_chain_clear_and_reselect(world: &mut ApplicationWorld) {
    click_unwrap_control(world, "AutoFaceChain");
    click_unwrap_control(world, "Clear");
    assert!(
        world
            .state()
            .unwrap_preview
            .expect("expected preview")
            .request
            .faces
            .is_empty()
    );
    select_lateral_face(world);
    faces_shows_1_face_ok_enabled(world);
}

#[given("the user selects the next Alignment option and then the following Alignment option")]
fn select_next_two_alignments(world: &mut ApplicationWorld) {
    click_unwrap_control(world, "XZ");
    click_unwrap_control(world, "YZ");
    assert_eq!(
        world
            .state()
            .unwrap_preview
            .expect("expected preview")
            .request
            .alignment,
        UnwrapAlignment::OriginYZ
    );
}

#[given("the user clears Merge Result Body and accepts to create Unwrap1")]
fn clear_merge_and_accept_unwrap1(world: &mut ApplicationWorld) {
    click_unwrap_control(world, "MergeResultBody");
    click_unwrap_control(world, "OK");
    panel_closes_unwrap1_and_surface_body(world);
}

#[when("the user double-clicks Unwrap1")]
fn user_double_clicks_unwrap1(world: &mut ApplicationWorld) {
    let center = world.run_in_window("Model", "Unwrap1").center();
    world.double_click(super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::diagnostics::window_geometry(UNWRAP_WINDOW).is_some(),
        "expected Unwrap panel to reopen on double-click of Unwrap1"
    );
}

#[then("the properties show 1 Face and 1 Vertex")]
fn properties_show_1_face_and_1_vertex(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(UNWRAP_WINDOW));
    assert!(text.contains("1Face"), "expected 1 Face, saw {text}");
    assert!(text.contains("1Vertex"), "expected 1 Vertex, saw {text}");
}

#[then("Auto Face Chain and Merge Result Body are cleared")]
fn auto_chain_and_merge_cleared(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .unwrap_preview
        .expect("expected unwrap preview state");
    assert!(!preview.request.auto_face_chain);
    assert!(!preview.request.merge_result_body);
}

#[when("the user activates Linear Result")]
fn user_activates_linear_result(world: &mut ApplicationWorld) {
    click_unwrap_row_button(world, "LinearResult");
}

#[then("the status bar requests a contiguous edge set to create one linear edge")]
fn status_bar_requests_linear_edge(world: &mut ApplicationWorld) {
    let status = super::super::world::squeezed(&world.window_text("##status-bar"));
    assert!(
        status.contains("contiguousedgesettocreateonelinearedge"),
        "expected Linear Result prompt in status bar, saw {status}"
    );
}

#[when("the user attempts to select the lower edge of the displayed triangular preview")]
fn user_attempts_select_preview_lower_edge(world: &mut ApplicationWorld) {
    let (min, max) = scene_bounds(world);
    let point = [
        f64::from((min[0] + max[0]) * 0.5),
        f64::from(min[1]),
        f64::from(min[2]),
    ];
    if let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) {
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
}

#[then("the Linear Result selector remains unpopulated")]
fn linear_result_remains_unpopulated(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .unwrap_preview
        .expect("expected unwrap preview state");
    assert!(
        preview.request.linear_result_edges.is_empty(),
        "expected Linear Result to remain unpopulated"
    );
}

#[when("the user activates Rigid Result")]
fn user_activates_rigid_result(world: &mut ApplicationWorld) {
    click_unwrap_row_button(world, "RigidResult");
}

#[then("the status bar requests contiguous planar edges that will not deform")]
fn status_bar_requests_rigid_edges(world: &mut ApplicationWorld) {
    let status = super::super::world::squeezed(&world.window_text("##status-bar"));
    assert!(
        status.contains("contiguousplanaredgesthatwillnotdeform"),
        "expected Rigid Result prompt in status bar, saw {status}"
    );
}

#[when("the user activates Seam Edges")]
fn user_activates_seam_edges(world: &mut ApplicationWorld) {
    click_unwrap_row_button(world, "SeamEdges");
}

#[then("the status bar requests edges to act as seams")]
fn status_bar_requests_seam_edges(world: &mut ApplicationWorld) {
    let status = super::super::world::squeezed(&world.window_text("##status-bar"));
    assert!(
        status.contains("edgestoactasseams"),
        "expected Seam Edges prompt in status bar, saw {status}"
    );
}
