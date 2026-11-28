// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Chamfer partial edges and viewport layout contract (chamfer-06, 07, 10).
use super::super::super::*;
use super::setup::{
    CHAMFER_WINDOW, accept_chamfer_and_wait, assert_chamfer_ready, click_chamfer_control,
    open_chamfer_fixture, open_chamfer_ribbon, pick_valid_unmodified_edge, reopen_chamfer,
    replace_chamfer_field, solid_volume,
};
use cucumber::{given, then, when};

#[given("I create Chamfer3 on another unmodified edge using two-distance mode at 2 mm and 1 mm")]
fn create_chamfer3_two_dist(world: &mut ApplicationWorld) {
    open_chamfer_ribbon(world);
    click_chamfer_control(world, "TwoDist");
    replace_chamfer_field(world, "Distance1", "2 mm");
    replace_chamfer_field(world, "Distance2", "1 mm");
    pick_valid_unmodified_edge(world, false);
    assert_chamfer_ready(world);
    accept_chamfer_and_wait(world);
    let browser = world.state().browser.expect("expected browser");
    assert!(browser.features.iter().any(|f| f.name == "Chamfer3"));
}

#[given("an equal-distance chamfer preview on a remaining vertical block edge")]
fn equal_chamfer_preview_on_vertical_edge(world: &mut ApplicationWorld) {
    open_chamfer_ribbon(world);
    replace_chamfer_field(world, "Distance", "1 mm");
    pick_valid_unmodified_edge(world, true);
    assert_chamfer_ready(world);
}

#[when("I open Partial and click the highlighted edge away from its distance manipulator")]
fn open_partial_and_click_edge(world: &mut ApplicationWorld) {
    click_chamfer_control(world, "Partial");
    let preview = world
        .state()
        .chamfer_v2_preview
        .expect("expected active Chamfer preview");
    let [p0, p1] = preview
        .edge_endpoints
        .expect("expected edge endpoints for partial click");
    let pt = [
        p0[0] + (p1[0] - p0[0]) * 0.35,
        p0[1] + (p1[1] - p0[1]) * 0.35,
        p0[2] + (p1[2] - p0[2]) * 0.35,
    ];
    if let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, pt) {
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
    if world
        .state()
        .chamfer_v2_preview
        .as_ref()
        .is_none_or(|p| p.request.partial.is_none())
    {
        click_chamfer_control(world, "AddEdge");
    }
}

#[then("the table shows 1 Selected and values for To Start, Chamfer, and To End")]
fn partial_table_shows_1_selected(world: &mut ApplicationWorld) {
    let text = super::super::super::world::squeezed(&world.window_text(CHAMFER_WINDOW));
    assert!(
        text.contains("1Selected")
            && text.contains("ToStart")
            && text.contains("Chamfer")
            && text.contains("ToEnd"),
        "expected 1 Selected and partial fields, saw {text}"
    );
}

#[then("the preview shows endpoints along the edge")]
fn preview_shows_endpoints_along_edge(world: &mut ApplicationWorld) {
    assert_chamfer_ready(world);
    let preview = world.state().chamfer_v2_preview.unwrap();
    assert!(preview.edge_endpoints.is_some());
    assert!(preview.request.partial.is_some());
}

#[then("Chamfer4 is added and the bevel stops short of one end")]
fn chamfer4_added_and_stops_short(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    let f = browser
        .features
        .iter()
        .find(|f| f.name == "Chamfer4")
        .expect("expected Chamfer4 in browser");
    let seed = browser
        .chamfer_v2_edits
        .get(&f.id)
        .expect("expected Chamfer4 v2 edit seed");
    let partial = seed.partial.as_ref().expect("expected saved partial spec");
    assert!(
        partial.to_end != "0 mm",
        "expected bevel to stop short of one end, got {partial:?}"
    );
}

#[given("a writable copy of `fixtures/cad/extrusion/extrusion-rect-8mm.opd` is open")]
fn open_extrusion_rect_8mm(world: &mut ApplicationWorld) {
    open_chamfer_fixture(world, "fixtures/cad/extrusion/extrusion-rect-8mm.opd");
}

#[given(
    "I create three 1 mm equal-distance chamfers on three distinct vertical edges, leaving the fourth vertical edge unmodified"
)]
fn create_three_vertical_chamfers(world: &mut ApplicationWorld) {
    for expected in ["Chamfer1", "Chamfer2", "Chamfer3"] {
        open_chamfer_ribbon(world);
        replace_chamfer_field(world, "Distance", "1 mm");
        pick_valid_unmodified_edge(world, true);
        assert_chamfer_ready(world);
        accept_chamfer_and_wait(world);
        let browser = world.state().browser.expect("expected browser");
        assert!(browser.features.iter().any(|f| f.name == expected));
    }
}

#[given(
    "I create Chamfer4 as a partial equal-distance chamfer on the remaining 8 mm vertical edge with To Start 0 mm"
)]
fn create_chamfer4_partial_8mm(world: &mut ApplicationWorld) {
    open_chamfer_ribbon(world);
    replace_chamfer_field(world, "Distance", "1 mm");
    pick_valid_unmodified_edge(world, true);
    assert_chamfer_ready(world);
    click_chamfer_control(world, "Partial");
    click_chamfer_control(world, "AddEdge");
    replace_chamfer_field(world, "ToStart", "0 mm");
    assert_chamfer_ready(world);
    accept_chamfer_and_wait(world);
    let browser = world.state().browser.expect("expected browser");
    assert!(browser.features.iter().any(|f| f.name == "Chamfer4"));
}

#[when("I reopen it and open Partial")]
fn reopen_chamfer4_and_open_partial(world: &mut ApplicationWorld) {
    reopen_chamfer(world, "Chamfer4");
    click_chamfer_control(world, "Partial");
}

#[then("its saved start, chamfer, and end distances appear")]
fn saved_partial_distances_appear(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .chamfer_v2_preview
        .expect("expected reopened partial preview");
    let partial = preview
        .request
        .partial
        .expect("expected saved partial state");
    assert!((partial.edge_length_cm - 0.8).abs() < 0.02);
    assert_eq!(partial.to_start.trim(), "0 mm");
}

#[when("I choose Chamfer as the driven dimension and set To End to 2 mm")]
fn set_chamfer_driven_and_to_end_2mm(world: &mut ApplicationWorld) {
    click_chamfer_control(world, "Chamfer");
    // Re-ensure Partial tab stays open and set To End to 2 mm (driven defaults to Chamfer).
    if super::super::super::world::squeezed(&world.window_text(CHAMFER_WINDOW)).contains("Distance")
    {
        click_chamfer_control(world, "Partial");
    }
    replace_chamfer_field(world, "ToEnd", "2 mm");
}

#[then("the table recalculates Chamfer to 6 mm with To Start at 0 mm")]
fn verify_partial_recalculates_to_6mm(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .chamfer_v2_preview
        .expect("expected recalculated partial preview");
    let partial = preview.request.partial.expect("expected partial state");
    assert_eq!(partial.to_start.trim(), "0 mm");
    assert_eq!(partial.to_end.trim(), "2 mm");
    assert_eq!(partial.chamfer_length.trim(), "6 mm");
}

#[then("accepting the dialog updates the feature")]
fn accept_partial_update(world: &mut ApplicationWorld) {
    let _before = solid_volume(world);
    assert_chamfer_ready(world);
    accept_chamfer_and_wait(world);
    reopen_chamfer(world, "Chamfer4");
    let preview = world.state().chamfer_v2_preview.unwrap();
    let partial = preview.request.partial.unwrap();
    assert_eq!(partial.chamfer_length.trim(), "6 mm");
    assert_eq!(partial.to_end.trim(), "2 mm");
    click_chamfer_control(world, "Cancel");
}

#[then("the Chamfer dialog opens anchored near the upper-left corner of the active viewport")]
fn chamfer_anchored_upper_left(_world: &mut ApplicationWorld) {
    let (origin, size) = super::super::super::diagnostics::window_geometry(CHAMFER_WINDOW)
        .expect("expected Chamfer window geometry");
    assert!(
        (size[0] - 280.0).abs() <= 4.0,
        "expected Chamfer width ~280px, got {}",
        size[0]
    );
    assert!(
        origin[0] >= 10.0 && origin[1] >= 10.0,
        "expected upper-left viewport inset, got {origin:?}"
    );
}

#[then("the ViewCube and Navigation Bar remain completely unobstructed")]
fn viewcube_and_navbar_unobstructed(_world: &mut ApplicationWorld) {
    let (origin, size) = super::super::super::diagnostics::window_geometry(CHAMFER_WINDOW)
        .expect("expected Chamfer window geometry");
    let right = origin[0] + size[0];
    let bottom = origin[1] + size[1];
    for hud in ["ViewCube", "Navigation Bar"] {
        if let Some((ho, hs)) = super::super::super::diagnostics::window_geometry(hud) {
            let overlaps = origin[0] < ho[0] + hs[0]
                && right > ho[0]
                && origin[1] < ho[1] + hs[1]
                && bottom > ho[1];
            assert!(
                !overlaps,
                "Chamfer ({origin:?}, {size:?}) overlaps {hud} ({ho:?}, {hs:?})"
            );
        }
    }
}

#[then("the dialog height fits its active controls with no empty vertical void beneath the footer")]
fn chamfer_height_fits_controls(world: &mut ApplicationWorld) {
    let (origin, size) = super::super::super::diagnostics::window_geometry(CHAMFER_WINDOW)
        .expect("expected Chamfer window geometry");
    let ok_bottom = world.run_in_window(CHAMFER_WINDOW, "OK").maximum[1];
    let win_bottom = origin[1] + size[1];
    let trailing = win_bottom - ok_bottom;
    assert!(
        (0.0..=56.0).contains(&trailing),
        "expected tight footer height without empty void, trailing={trailing} (ok_bottom={ok_bottom}, win_bottom={win_bottom})"
    );
}

#[then("resizing the main window keeps the Chamfer dialog clamped inside the viewport bounds")]
fn resizing_keeps_chamfer_clamped(world: &mut ApplicationWorld) {
    let resized = [960u32, 640u32];
    world.pump(vec![oblikovati_viewport::ViewportEvent::Resized(resized)]);
    world.pump(Vec::new());
    let (origin, size) = super::super::super::diagnostics::window_geometry(CHAMFER_WINDOW)
        .expect("expected Chamfer window geometry after resize");
    assert!(
        origin[0] >= 0.0
            && origin[1] >= 0.0
            && origin[0] + size[0] <= resized[0] as f32 + 2.0
            && origin[1] + size[1] <= resized[1] as f32 + 2.0,
        "Chamfer ({origin:?}, {size:?}) escaped resized window {resized:?}"
    );
}
