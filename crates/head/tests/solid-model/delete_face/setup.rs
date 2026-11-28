// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Part Modify Delete Face scenarios (delete-face-01..05, #3918).
use super::super::super::*;
use oblikovati_viewport::ViewportNavigation;

pub(super) const DELETE_FACE_WINDOW: &str = "Delete Face";
pub(super) const FAILURE_MODAL_WINDOW: &str = "Create DelFace feature failed";

pub(super) fn click_delete_face_ribbon(world: &mut ApplicationWorld) {
    super::super::super::ribbon_panel_expansion::click_expansion_arrow(world, "Modify");
    world.pump(Vec::new());
    world.pump(Vec::new());
    let run = world
        .run_in_window("##ribbon-additional-PartModify", "DeleteFace")
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(run));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(DELETE_FACE_WINDOW).is_some(),
        "expected Delete Face panel to open, saw {}",
        world.text()
    );
}

pub(super) fn click_panel_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window(DELETE_FACE_WINDOW, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_modal_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window(FAILURE_MODAL_WINDOW, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn set_heal_remaining_faces(world: &mut ApplicationWorld, desired: bool) {
    let current = world
        .state()
        .delete_face_preview
        .as_ref()
        .is_some_and(|p| p.request.heal_remaining_faces);
    if current != desired {
        click_panel_control(world, "HealRemainingFaces");
    }
    let after = world
        .state()
        .delete_face_preview
        .as_ref()
        .is_some_and(|p| p.request.heal_remaining_faces);
    assert_eq!(
        after, desired,
        "expected Heal Remaining Faces={desired}, got {after}"
    );
}

pub(super) fn frame_part_clear_of_panel(world: &mut ApplicationWorld) {
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.35 },
    )
    .unwrap();
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Pan([360, 0]),
    )
    .unwrap();
    world.pump(Vec::new());
}

pub(super) fn scene_bounds(world: &mut ApplicationWorld) -> ([f32; 3], [f32; 3]) {
    let mut min = [f32::INFINITY; 3];
    let mut max = [f32::NEG_INFINITY; 3];
    for tri in &world.app().window.scene_triangles {
        for vertex in tri.positions {
            for axis in 0..3 {
                min[axis] = min[axis].min(vertex[axis]);
                max[axis] = max[axis].max(vertex[axis]);
            }
        }
    }
    (min, max)
}

pub(super) fn click_top_face_of_single_solid(world: &mut ApplicationWorld) {
    frame_part_clear_of_panel(world);
    let (min, max) = scene_bounds(world);
    let cx = f64::from((min[0] + max[0]) * 0.5);
    let cy = f64::from((min[1] + max[1]) * 0.5);
    let z_top = f64::from(max[2]);
    let before = selected_face_count(world);
    for offset in [[0.0, 0.0], [0.1, 0.0], [-0.1, 0.0], [0.0, 0.1]] {
        let point = [cx + offset[0], cy + offset[1], z_top];
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if selected_face_count(world) > before {
            return;
        }
    }
    panic!(
        "expected top face click to select a face; preview={:?}",
        world.state().delete_face_preview
    );
}

pub(super) fn click_curved_side_face_of_cylinder(world: &mut ApplicationWorld) {
    frame_part_clear_of_panel(world);
    let (min, max) = scene_bounds(world);
    let cx = f64::from((min[0] + max[0]) * 0.5);
    let cy = f64::from((min[1] + max[1]) * 0.5);
    let cz = f64::from((min[2] + max[2]) * 0.5);
    let rx = f64::from((max[0] - min[0]) * 0.5);
    let ry = f64::from((max[1] - min[1]) * 0.5);
    let before = selected_face_count(world);
    let candidates = [
        [cx + rx * 0.707, cy - ry * 0.707, cz],
        [cx + rx, cy, cz],
        [cx, cy - ry, cz],
        [cx - rx * 0.707, cy - ry * 0.707, cz],
    ];
    for point in candidates {
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if selected_face_count(world) > before {
            return;
        }
    }
    panic!(
        "expected curved side face click to select a face; preview={:?}",
        world.state().delete_face_preview
    );
}

pub(super) fn click_exposed_lower_circular_face(world: &mut ApplicationWorld) {
    // Orient to look upward at the exposed cylinder end cap centered at (10, 15, -10) mm = (1.0, 1.5, -1.0) cm.
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.35 },
    )
    .unwrap();
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Orbit {
            yaw: 0.0,
            pitch: -1.45,
        },
    )
    .unwrap();
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Pan([340, 0]),
    )
    .unwrap();
    world.pump(Vec::new());

    let center = [1.0, 1.5, -1.0];
    let offsets = [
        [0.0, 0.0],
        [0.12, 0.0],
        [-0.12, 0.0],
        [0.0, 0.12],
        [0.0, -0.12],
    ];
    for [dx, dy] in offsets {
        let point = [center[0] + dx, center[1] + dy, center[2]];
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if selected_face_count(world) == 1 && selected_face_near(world, center) {
            return;
        }
        if selected_face_count(world) > 0 {
            click_panel_control(world, "x");
        }
    }
    panic!(
        "expected lower circular face at (10, 15, -10) mm to be selected; preview={:?}",
        world.state().delete_face_preview
    );
}

pub(super) fn selected_face_count(world: &mut ApplicationWorld) -> usize {
    world
        .state()
        .delete_face_preview
        .as_ref()
        .map_or(0, |p| p.request.faces.len())
}

fn selected_face_near(world: &mut ApplicationWorld, expected_cm: [f64; 3]) -> bool {
    let Some(preview) = world.state().delete_face_preview else {
        return false;
    };
    let Some(face_key) = preview.request.faces.first() else {
        return false;
    };
    let doc = world.active_id();
    let app = world.app();
    let ws = app.owner.documents().workspace();
    let session = ws.session(doc).unwrap();
    let guid = ws.identity(doc).unwrap().internal_name;
    let eval = app.owner.part_scenes_current(doc).unwrap();
    for body in eval.ordered_bodies() {
        if let Ok(center) = body.face_bounds_center(session.history(), guid, face_key) {
            let dist = ((center[0] - expected_cm[0]).powi(2)
                + (center[1] - expected_cm[1]).powi(2)
                + (center[2] - expected_cm[2]).powi(2))
            .sqrt();
            if dist < 0.05 {
                return true;
            }
        }
    }
    false
}

pub(super) fn has_highlighted_face_triangles(world: &mut ApplicationWorld) -> bool {
    let Some(scheme) = world.state().color_scheme else {
        return false;
    };
    let _ = scheme;
    // Highlighted faces use secondary_select color, which differs from the uniform part body color.
    let tris = &world.app().window.scene_triangles;
    let Some(first_color) = tris.first().map(|t| t.color) else {
        return false;
    };
    tris.iter().any(|t| t.color != first_color)
}
