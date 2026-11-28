// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Work Plane scenarios (work-plane-01..12, #3918).

use super::super::super::*;
use oblikovati_api::client::Client;

pub(super) fn choose_plane_menu_item(world: &mut ApplicationWorld, label: &str) {
    let plane = world.run_in_window("##application-ribbon", "Plane").clone();
    let arrow = world
        .frame
        .runs
        .iter()
        .filter(|run| run.list == plane.list && matches!(run.text.as_str(), "▼" | "v"))
        .filter(|run| {
            run.minimum[0] > plane.maximum[0] && (run.center()[1] - plane.center()[1]).abs() < 18.0
        })
        .min_by(|a, b| a.minimum[0].total_cmp(&b.minimum[0]))
        .unwrap_or_else(|| panic!("expected Plane drop-down arrow, saw {}", world.text()))
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(arrow));
    world.pump(Vec::new());
    let squeezed = super::super::super::world::squeezed(label);
    let item = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| {
            run.list != plane.list && super::super::super::world::squeezed(&run.text) == squeezed
        })
        .unwrap_or_else(|| {
            panic!(
                "expected {label:?} ({squeezed:?}) in Plane menu, saw {}",
                world.text()
            )
        })
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(item));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn ensure_origin_expanded(world: &mut ApplicationWorld) {
    if !world.shows("XY Plane") {
        let origin = world.run("Origin").clone();
        let arrow = [origin.minimum[0] - 8.0, origin.center()[1]];
        world.click(super::super::super::diagnostics::pointer_pixels(arrow));
        world.pump(Vec::new());
        if !world.shows("XY Plane") {
            let center = origin.center();
            world.click(super::super::super::diagnostics::pointer_pixels(center));
            world.pump(Vec::new());
        }
    }
}

pub(super) fn query_topology(world: &mut ApplicationWorld) -> oblikovati_api::wire::BodyTopology {
    let keys = Client::new(world.app().owner.document_host())
        .model()
        .reference_keys()
        .expect("expected body reference keys");
    keys.bodies
        .and_then(|mut bodies| bodies.drain(..).next())
        .expect("expected at least one solid body")
}

pub(super) fn click_model_point(world: &mut ApplicationWorld, point: [f64; 3]) {
    let pixel = world
        .app()
        .window
        .native
        .view
        .model_pixel(point)
        .unwrap_or_else(|cause| panic!("failed to project {point:?}: {cause}"));
    world.click(pixel);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn planar_top_face_point(world: &mut ApplicationWorld) -> [f64; 3] {
    let body = query_topology(world);
    let faces = body.faces.expect("expected solid faces");
    faces
        .iter()
        .filter(|face| face.kind == "plane")
        .filter_map(|face| {
            let pt = face.point.as_ref()?;
            (pt.len() == 3).then_some([pt[0], pt[1], pt[2]])
        })
        .max_by(|a, b| a[2].total_cmp(&b[2]))
        .expect("expected a planar top face on the solid")
}

pub(super) fn three_non_collinear_vertex_points(world: &mut ApplicationWorld) -> [[f64; 3]; 3] {
    let body = query_topology(world);
    let vertices: Vec<[f64; 3]> = body
        .vertices
        .expect("expected solid vertices")
        .into_iter()
        .filter_map(|v| {
            let pt = v.point?;
            (pt.len() == 3).then_some([pt[0], pt[1], pt[2]])
        })
        .collect();
    assert!(
        vertices.len() >= 3,
        "expected at least 3 vertices, saw {}",
        vertices.len()
    );
    for i in 0..vertices.len() {
        for j in (i + 1)..vertices.len() {
            for k in (j + 1)..vertices.len() {
                if non_collinear(vertices[i], vertices[j], vertices[k]) {
                    return [vertices[i], vertices[j], vertices[k]];
                }
            }
        }
    }
    panic!("expected three non-collinear vertices on solid");
}

pub(super) fn two_adjoining_edge_points(world: &mut ApplicationWorld) -> [[f64; 3]; 2] {
    let body = query_topology(world);
    let edges: Vec<[f64; 3]> = body
        .edges
        .expect("expected solid edges")
        .into_iter()
        .filter_map(|e| {
            let pt = e.point?;
            (pt.len() == 3).then_some([pt[0], pt[1], pt[2]])
        })
        .collect();
    // Pick two edges on the top face (max Z) whose midpoints are not collinear/opposite.
    let max_z = edges.iter().map(|p| p[2]).fold(f64::NEG_INFINITY, f64::max);
    let top_edges: Vec<[f64; 3]> = edges
        .into_iter()
        .filter(|p| (p[2] - max_z).abs() < 1e-4)
        .collect();
    assert!(
        top_edges.len() >= 2,
        "expected at least 2 top-face edges, saw {}",
        top_edges.len()
    );
    for i in 0..top_edges.len() {
        for j in (i + 1)..top_edges.len() {
            let dx = (top_edges[i][0] - top_edges[j][0]).abs();
            let dy = (top_edges[i][1] - top_edges[j][1]).abs();
            if dx > 1e-3 && dy > 1e-3 {
                return [top_edges[i], top_edges[j]];
            }
        }
    }
    [top_edges[0], top_edges[1]]
}

pub(super) fn base_edge_midpoint_near_origin_neg_x(world: &mut ApplicationWorld) -> [f64; 3] {
    // body-properties-start.opd: base edge from (0, 0, 0) to (-40, 0, 0) mm = (-4, 0, 0) cm -> midpoint (-2, 0, 0) cm.
    let body = query_topology(world);
    let edges = body.edges.expect("expected solid edges");
    edges
        .into_iter()
        .filter_map(|e| {
            let pt = e.point?;
            (pt.len() == 3).then_some([pt[0], pt[1], pt[2]])
        })
        .min_by(|a, b| {
            let da = (a[0] + 2.0).powi(2) + a[1].powi(2) + a[2].powi(2);
            let db = (b[0] + 2.0).powi(2) + b[1].powi(2) + b[2].powi(2);
            da.total_cmp(&db)
        })
        .expect("expected straight base edge near (-2, 0, 0) cm")
}

pub(super) fn work_plane_count(world: &mut ApplicationWorld) -> usize {
    world
        .state()
        .browser
        .expect("expected browser")
        .features
        .iter()
        .filter(|feature| feature.kind == "Work Plane")
        .count()
}

pub(super) fn orient_isometric_for_picking(world: &mut ApplicationWorld) {
    world
        .app()
        .window
        .native
        .view
        .set_camera(oblikovati_viewport::ViewportCamera {
            eye: [16.0, 18.0, 14.0],
            target: [0.0, 0.0, 1.0],
            up: [0.0, 0.0, 1.0],
            ..Default::default()
        })
        .expect("expected valid camera");
    world.pump(Vec::new());
    world.pump(Vec::new());
}

fn non_collinear(a: [f64; 3], b: [f64; 3], c: [f64; 3]) -> bool {
    let ab = [b[0] - a[0], b[1] - a[1], b[2] - a[2]];
    let ac = [c[0] - a[0], c[1] - a[1], c[2] - a[2]];
    let cx = ab[1] * ac[2] - ab[2] * ac[1];
    let cy = ab[2] * ac[0] - ab[0] * ac[2];
    let cz = ab[0] * ac[1] - ab[1] * ac[0];
    (cx * cx + cy * cy + cz * cz).sqrt() > 1e-4
}
