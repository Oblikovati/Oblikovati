// SPDX-License-Identifier: GPL-2.0-only
//! Viewport navigation, occlusion avoidance, and face-discovery helpers for Replace Face BDD scenarios (#3918).
use super::super::super::*;
use oblikovati_martensite::PrismSurfaceKind;
use oblikovati_viewport::ViewportNavigation;

pub(super) fn pan_clear_of_panel(world: &mut ApplicationWorld) {
    if crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.35 },
    )
    .is_ok()
    {
        let _ = crate::HeadWindow::navigate_view(
            &mut world.app().window.native,
            ViewportNavigation::Pan([360, 0]),
        );
    }
    world.pump(Vec::new());
}

fn open_canvas_bounds() -> ([f32; 2], [i32; 2]) {
    let mut min_x = 20.0_f32;
    let mut max_x = 1650.0_f32;
    let mut min_y = 200.0_f32;
    let mut max_y = 1150.0_f32;
    for rect in super::super::super::diagnostics::window_occlusions() {
        let width = rect[2] - rect[0];
        let height = rect[3] - rect[1];
        // Left side panels (Browser / Properties)
        if rect[0] < 950.0 && height > 200.0 && rect[2] < 1150.0 {
            min_x = min_x.max(rect[2] + 24.0);
        }
        // Right side panels / ViewCube / NavBar
        if rect[0] > 1200.0 {
            max_x = max_x.min(rect[0] - 24.0);
        }
        // Top ribbon
        if rect[1] <= 10.0 && width > 400.0 && rect[3] < 400.0 {
            min_y = min_y.max(rect[3] + 24.0);
        }
        // Bottom bar
        if rect[1] > 900.0 && width > 400.0 {
            max_y = max_y.min(rect[1] - 24.0);
        }
    }
    if max_x <= min_x + 80.0 {
        min_x = 940.0;
        max_x = 1640.0;
    }
    if max_y <= min_y + 80.0 {
        min_y = 220.0;
        max_y = 1140.0;
    }
    let target = [
        ((min_x + max_x) * 0.5) as i32,
        ((min_y + max_y) * 0.5) as i32,
    ];
    ([max_x, max_y], target)
}

pub(super) fn pixel_is_occluded(pixel: [i32; 2]) -> bool {
    let x = pixel[0] as f32;
    let y = pixel[1] as f32;
    if !(20.0..=2100.0).contains(&x) || !(200.0..=1300.0).contains(&y) {
        return true;
    }
    super::super::super::diagnostics::window_occlusions()
        .into_iter()
        .any(|rect| {
            x >= rect[0] - 6.0 && x <= rect[2] + 6.0 && y >= rect[1] - 6.0 && y <= rect[3] + 6.0
        })
}

pub(super) fn ensure_point_visible_and_unoccluded(
    world: &mut ApplicationWorld,
    point: [f64; 3],
) -> [i32; 2] {
    let mut pixel = crate::HeadWindow::model_pixel(&world.app().window.native, point).unwrap();
    for _ in 0..4 {
        if !pixel_is_occluded(pixel) {
            return pixel;
        }
        let (_, target) = open_canvas_bounds();
        let dx = target[0] - pixel[0];
        let dy = target[1] - pixel[1];
        let _ = crate::HeadWindow::navigate_view(
            &mut world.app().window.native,
            ViewportNavigation::Pan([dx, dy]),
        );
        world.pump(Vec::new());
        let next = crate::HeadWindow::model_pixel(&world.app().window.native, point).unwrap();
        let moved_x = next[0] - pixel[0];
        let moved_y = next[1] - pixel[1];
        if dx.abs() > 10 && moved_x.signum() == -dx.signum() {
            let _ = crate::HeadWindow::navigate_view(
                &mut world.app().window.native,
                ViewportNavigation::Pan([-2 * dx, 0]),
            );
            world.pump(Vec::new());
        }
        if dy.abs() > 10 && moved_y.signum() == -dy.signum() {
            let _ = crate::HeadWindow::navigate_view(
                &mut world.app().window.native,
                ViewportNavigation::Pan([0, -2 * dy]),
            );
            world.pump(Vec::new());
        }
        pixel = crate::HeadWindow::model_pixel(&world.app().window.native, point).unwrap();
    }
    pixel
}

pub(super) fn click_model_point(world: &mut ApplicationWorld, point: [f64; 3]) -> bool {
    let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) else {
        return false;
    };
    let pixel = if pixel_is_occluded(pixel) {
        ensure_point_visible_and_unoccluded(world, point)
    } else {
        pixel
    };
    world.click(pixel);
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
    true
}

pub(super) fn click_face_around_point(world: &mut ApplicationWorld, point: [f64; 3]) -> bool {
    for d in [[0.0, 0.0], [0.05, 0.0], [-0.05, 0.0], [0.0, 0.05]] {
        if click_model_point(world, [point[0] + d[0], point[1] + d[1], point[2]])
            && world
                .state()
                .thicken_offset_preview
                .as_ref()
                .is_some_and(|p| !p.request.faces.is_empty())
        {
            return true;
        }
    }
    false
}

pub(super) fn find_highest_planar_solid_face_center(world: &mut ApplicationWorld) -> [f64; 3] {
    let doc = world.active_id();
    let app = world.app();
    let ws = app.owner.documents().workspace();
    let session = ws.session(doc).unwrap();
    let guid = ws.identity(doc).unwrap().internal_name;
    let eval = app.owner.part_scenes_current(doc).unwrap();
    let mut best: Option<(f64, [f64; 3])> = None;
    for body in eval.ordered_bodies() {
        if body.is_surface() {
            continue;
        }
        for face in body.face_reference_keys(session.history(), guid).unwrap() {
            let Ok(topo) = body.face_topology(session.history(), guid, &face) else {
                continue;
            };
            if topo.surface != PrismSurfaceKind::Plane {
                continue;
            }
            let Ok(center) = body.face_bounds_center(session.history(), guid, &face) else {
                continue;
            };
            if best.is_none_or(|(z, _)| center[2] > z) {
                best = Some((center[2], center));
            }
        }
    }
    best.expect("expected an exposed planar solid face").1
}

pub(super) fn find_torus_face_candidates(world: &mut ApplicationWorld) -> Vec<[f64; 3]> {
    let doc = world.active_id();
    let app = world.app();
    let ws = app.owner.documents().workspace();
    let session = ws.session(doc).unwrap();
    let guid = ws.identity(doc).unwrap().internal_name;
    let eval = app.owner.part_scenes_current(doc).unwrap();
    let mut diag = Vec::new();
    for body in eval.ordered_bodies() {
        if body.is_surface() {
            continue;
        }
        let faces = body
            .face_reference_keys(session.history(), guid)
            .unwrap_or_default();
        diag.push(faces.len());
        let torus_face = if faces.len() == 1 {
            faces.into_iter().next()
        } else {
            faces.into_iter().find(
                |face| match body.face_topology(session.history(), guid, face) {
                    Ok(topo) => topo.surface == PrismSurfaceKind::Torus,
                    Err(msg) => msg.contains("SurfaceOfRevolution"),
                },
            )
        };
        if let Some(face) = torus_face {
            let meshes = body
                .named_face_meshes(
                    session.history(),
                    guid,
                    oblikovati_martensite::PrismMeshOptions {
                        chord_error: 0.1,
                        angular_error: 0.5,
                    },
                )
                .expect("expected torus face meshes");
            if let Some(mesh) = meshes.into_iter().find(|m| m.key == face) {
                let centroids: Vec<[f64; 3]> = mesh
                    .triangles
                    .iter()
                    .map(|t| {
                        [
                            (t[0][0] + t[1][0] + t[2][0]) / 3.0,
                            (t[0][1] + t[1][1] + t[2][1]) / 3.0,
                            (t[0][2] + t[1][2] + t[2][2]) / 3.0,
                        ]
                    })
                    .collect();
                let mut picks = Vec::new();
                if let Some(&p) = centroids.iter().max_by(|a, b| a[2].total_cmp(&b[2])) {
                    picks.push(p);
                }
                if let Some(&p) = centroids.iter().max_by(|a, b| a[0].total_cmp(&b[0])) {
                    picks.push(p);
                }
                if let Some(&p) = centroids.iter().max_by(|a, b| a[1].total_cmp(&b[1])) {
                    picks.push(p);
                }
                let step = (centroids.len() / 8).max(1);
                for pt in centroids.iter().step_by(step).take(8) {
                    picks.push(*pt);
                }
                if !picks.is_empty() {
                    return picks;
                }
            }
        }
    }
    panic!("expected a toroidal face in the model (solid face counts={diag:?})");
}

pub(super) fn find_body_face_center(
    world: &mut ApplicationWorld,
    want_surface: bool,
    predicate: impl Fn([f64; 3]) -> bool,
) -> [f64; 3] {
    let doc = world.active_id();
    let app = world.app();
    let ws = app.owner.documents().workspace();
    let session = ws.session(doc).unwrap();
    let guid = ws.identity(doc).unwrap().internal_name;
    let eval = app.owner.part_scenes_current(doc).unwrap();
    for body in eval.ordered_bodies() {
        if body.is_surface() != want_surface {
            continue;
        }
        for face in body.face_reference_keys(session.history(), guid).unwrap() {
            if let Ok(center) = body.face_bounds_center(session.history(), guid, &face)
                && predicate(center)
            {
                return center;
            }
        }
    }
    panic!("expected matching face (want_surface={want_surface})");
}
