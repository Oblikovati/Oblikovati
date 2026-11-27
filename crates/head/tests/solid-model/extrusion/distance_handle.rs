// SPDX-License-Identifier: GPL-2.0-only
use super::super::diagnostics;
use super::direction::preview_depth;
use super::*;

#[given("`fixtures/cad/revolve/revolve-profile.opd` is open in Extrude with a distance preview")]
fn open_preview(world: &mut ApplicationWorld) {
    super::reference_extent::native_fixture_preview(world);
    assert_eq!(preview_depth(world), [0., 1.]);
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        oblikovati_viewport::ViewportNavigation::Home { margin: 0.12 },
    )
    .unwrap();
    world.pump(Vec::new());
}

#[when("the user drags the distance manipulator along the extrusion direction")]
fn drag_distance_a(world: &mut ApplicationWorld) {
    let (origin, size) = diagnostics::window_geometry("##extrude-distance-manipulator")
        .unwrap_or_else(|| {
            panic!(
                "expected a manipulator, with axis {:?}",
                world.state().extrude_handle
            )
        });
    let (panel_origin, panel_size) = diagnostics::window_geometry("Extrude").unwrap();
    assert!(
        (0..2).any(|axis| origin[axis] + size[axis] <= panel_origin[axis]
            || panel_origin[axis] + panel_size[axis] <= origin[axis]),
        "distance handle {origin:?} × {size:?} must clear Properties {panel_origin:?} × {panel_size:?}",
    );
    let handle =
        diagnostics::pointer_pixels([origin[0] + size[0] * 0.5, origin[1] + size[1] * 0.5]);
    let view = &world.app().window.native.view;
    let start = view.model_pixel([0., 0., 0.]).unwrap();
    let end = view.model_pixel([0., 0., 1.]).unwrap();
    let to = std::array::from_fn(|axis| handle[axis] + (end[axis] - start[axis]) / 2);
    if world.app().window.native.view.uses_wayland() {
        world.drag(handle, to);
    } else {
        super::super::desktop_input::drag(world, handle, to);
    }
    world.pump(Vec::new());
}

#[then("Distance A changes with the viewport dimension and the preview depth")]
fn distance_fields_follow_drag(world: &mut ApplicationWorld) {
    let state = world.state();
    let draft = state
        .extrude_draft_preview
        .expect("certified dragged preview");
    let length = draft.distance.parse::<f32>().unwrap_or_else(|_| {
        panic!(
            "drag must update Distance A to a numeric value, got {:?}",
            draft.distance
        )
    });
    assert!((length - 15.).abs() < 0.6, "dragged distance {length} mm");
    world.run_in_window("Extrude", &draft.distance);
    world.expect_text(&format!("A {} mm", draft.distance));
    assert!((preview_depth(world)[1] * 10. - length).abs() < 0.001);
    assert!(world.run_is_enabled("OK"));
}

#[when("the user enters 8 mm in Distance A")]
fn set_distance_a_from_panel(world: &mut ApplicationWorld) {
    replace_extrusion_distance(world, "8");
    world.pump(Vec::new());
}

#[then("the viewport dimension reads 8 mm and the preview returns to that depth")]
fn handle_follows_typed_distance(world: &mut ApplicationWorld) {
    world.expect_text("A 8 mm");
    assert_eq!(preview_depth(world), [0., 0.8]);
    assert_eq!(super::acceptance::feature_count(world), 0);
}
