// SPDX-License-Identifier: GPL-2.0-only
//! Eligibility queries, B-rep connected face chaining, and curve imprint overlays for Part Modify Mark (#3918).
use super::NamedSolid;
use crate::DisplayTriangle;
use martensite_tkbrep::{
    brep_adaptor::BRepAdaptorSurface,
    topexp,
    topods::{self, Shape},
    toptools::IndexedMapOfShape,
};
use martensite_tkg3d::{
    adaptor3d::Adaptor3dSurface,
    top_abs::{Orientation, ShapeEnum},
};
use martensite_tkmath::gp::Point3;
use martensite_tktopalgo::{
    brep_builder_api::{BRepBuilderAPIMakeShape, BRepBuilderAPIMakeVertex},
    brep_extrema::BRepExtremaDistShapeShape,
};
use oblikovati_model::{
    feature::mark::{MarkType, ResolvedMark},
    history::DocumentEditHistory,
    identity::DocumentGuid,
};
use std::collections::VecDeque;

const MIN_SOLID_VOLUME_CM3: f64 = 1e-9;
const MARK_RIBBON_HALF_WIDTH_CM: f64 = 0.015;
const MARK_SURFACE_LIFT_CM: f64 = 0.004;

/// Mark command eligibility evaluated against the active part's live B-rep bodies.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum MarkEligibility {
    /// Exactly one positive-volume solid body exists.
    Eligible,
    /// More than one solid body exists (`mark-01`).
    MultipleSolidBodies { count: usize },
    /// No positive-volume solid body exists (`mark-05`).
    NoVolume,
}

/// Evaluate Mark eligibility from exact B-rep solid counts and analytic volumes.
/// Example: `check_mark_eligibility(history, document, evaluation.ordered_bodies())`.
pub fn check_mark_eligibility<'a>(
    history: &DocumentEditHistory,
    document: DocumentGuid,
    bodies: impl IntoIterator<Item = &'a NamedSolid>,
) -> MarkEligibility {
    let mut positive_solids = 0usize;
    for body in bodies {
        if body.is_surface() {
            continue;
        }
        if let Ok(volume) = body.volume(history, document)
            && volume.is_finite()
            && volume > MIN_SOLID_VOLUME_CM3
        {
            positive_solids += 1;
        }
    }
    match positive_solids {
        0 => MarkEligibility::NoVolume,
        1 => MarkEligibility::Eligible,
        count => MarkEligibility::MultipleSolidBodies { count },
    }
}

/// Traverse edge-adjacent faces starting from `seed_face` in deterministic BFS order.
pub(crate) fn connected_face_chain_shapes(
    all_faces: &[(Vec<u8>, Shape)],
    seed_face: &[u8],
) -> Result<Vec<Vec<u8>>, String> {
    let Some(seed_idx) = all_faces
        .iter()
        .position(|(key, _)| key.as_slice() == seed_face)
    else {
        return Err("mark face chain: seed face not found on solid".into());
    };
    let mut face_edges = Vec::with_capacity(all_faces.len());
    for (_, face) in all_faces {
        face_edges.push(extract_face_edges(face)?);
    }
    let mut visited = vec![false; all_faces.len()];
    let mut queue = VecDeque::new();
    visited[seed_idx] = true;
    queue.push_back(seed_idx);
    let mut chain = Vec::new();
    while let Some(current) = queue.pop_front() {
        chain.push(all_faces[current].0.clone());
        let edges = &face_edges[current];
        for (neighbor, neighbor_edges) in face_edges.iter().enumerate() {
            if visited[neighbor] {
                continue;
            }
            let shares_edge = edges
                .iter()
                .any(|edge| neighbor_edges.iter().any(|other| edge.is_same(other)));
            if shares_edge {
                visited[neighbor] = true;
                queue.push_back(neighbor);
            }
        }
    }
    Ok(chain)
}

/// Project all sketch geometry sets onto target faces and generate crisp imprint ribbon triangles.
pub(crate) fn build_mark_overlay_triangles(
    target_faces: &[Shape],
    input: &ResolvedMark,
) -> Result<Vec<DisplayTriangle>, String> {
    if target_faces.is_empty() {
        return Err("mark target faces: expected at least one face on the solid body".into());
    }
    let mut triangles = Vec::new();
    for set in &input.geometry_sets {
        for polyline in &set.polylines_3d {
            if polyline.len() < 2 {
                continue;
            }
            let dense = densify_polyline(polyline, 0.25);
            let mut primary = Vec::with_capacity(dense.len());
            let mut opposite = Vec::new();
            for &point in &dense {
                if let Some((hit, normal)) = closest_face_projection(point, target_faces)? {
                    primary.push((hit, normal));
                }
                if set.mark_type == MarkType::MarkThrough
                    && let Some((hit, normal)) = farthest_face_projection(point, target_faces)?
                {
                    opposite.push((hit, normal));
                }
            }
            append_ribbon_triangles(&primary, &mut triangles);
            if set.mark_type == MarkType::MarkThrough {
                append_ribbon_triangles(&opposite, &mut triangles);
            }
        }
    }
    Ok(triangles)
}

fn extract_face_edges(face: &Shape) -> Result<Vec<Shape>, String> {
    let mut map = IndexedMapOfShape::default();
    topexp::map_shapes_of_type(face, ShapeEnum::Edge, &mut map)
        .map_err(|cause| cause.to_string())?;
    let mut edges = Vec::with_capacity(map.extent());
    for i in 1..=map.extent() {
        edges.push(map.find_key(i).map_err(|cause| cause.to_string())?.clone());
    }
    Ok(edges)
}

fn densify_polyline(points: &[[f64; 3]], max_step_cm: f64) -> Vec<[f64; 3]> {
    let mut out = Vec::new();
    for pair in points.windows(2) {
        let a = pair[0];
        let b = pair[1];
        let len = dist3(a, b);
        let steps = ((len / max_step_cm).ceil() as usize).clamp(1, 32);
        if out.is_empty() {
            out.push(a);
        }
        for i in 1..=steps {
            let t = i as f64 / steps as f64;
            out.push([
                a[0] + (b[0] - a[0]) * t,
                a[1] + (b[1] - a[1]) * t,
                a[2] + (b[2] - a[2]) * t,
            ]);
        }
    }
    out
}

type SurfaceHit = ([f64; 3], [f64; 3]);

fn closest_face_projection(point: [f64; 3], faces: &[Shape]) -> Result<Option<SurfaceHit>, String> {
    let mut best: Option<(f64, [f64; 3], [f64; 3])> = None;
    for face in faces {
        let Some((dist, projected)) = project_point_to_face(point, face)? else {
            continue;
        };
        if best.is_none_or(|(d, _, _)| dist < d) {
            let normal = face_normal_near(face).unwrap_or([0.0, 0.0, 1.0]);
            best = Some((dist, projected, normal));
        }
    }
    Ok(best.map(|(_, p, n)| (p, n)))
}

fn farthest_face_projection(
    point: [f64; 3],
    faces: &[Shape],
) -> Result<Option<SurfaceHit>, String> {
    if faces.len() < 2 {
        return Ok(None);
    }
    let mut best: Option<(f64, [f64; 3], [f64; 3])> = None;
    for face in faces {
        let Some((dist, projected)) = project_point_to_face(point, face)? else {
            continue;
        };
        if best.is_none_or(|(d, _, _)| dist > d) {
            let normal = face_normal_near(face).unwrap_or([0.0, 0.0, -1.0]);
            best = Some((dist, projected, normal));
        }
    }
    Ok(best.map(|(_, p, n)| (p, n)))
}

fn project_point_to_face(point: [f64; 3], face: &Shape) -> Result<Option<(f64, [f64; 3])>, String> {
    let mut vertex_maker = BRepBuilderAPIMakeVertex::new(Point3::new(point[0], point[1], point[2]))
        .map_err(|cause| cause.to_string())?;
    let vertex = vertex_maker
        .shape()
        .map_err(|cause| cause.to_string())?
        .clone();
    let extrema = BRepExtremaDistShapeShape::with_shapes(&vertex, face, Default::default())
        .map_err(|cause| cause.to_string())?;
    if !extrema.is_done() || extrema.nb_solution() == 0 {
        return Ok(None);
    }
    let dist = extrema.value().map_err(|cause| cause.to_string())?;
    let on_face = extrema
        .point_on_shape2(1)
        .map_err(|cause| cause.to_string())?;
    Ok(Some((dist, [on_face.x(), on_face.y(), on_face.z()])))
}

fn face_normal_near(face: &Shape) -> Result<[f64; 3], String> {
    let native = topods::face(face).map_err(|cause| cause.to_string())?;
    let adaptor =
        BRepAdaptorSurface::with_face(&native, true).map_err(|cause| cause.to_string())?;
    let u0 = adaptor.first_u_parameter().map_err(|e| e.to_string())?;
    let u1 = adaptor.last_u_parameter().map_err(|e| e.to_string())?;
    let v0 = adaptor.first_v_parameter().map_err(|e| e.to_string())?;
    let v1 = adaptor.last_v_parameter().map_err(|e| e.to_string())?;
    let d1 = adaptor
        .d1(0.5 * (u0 + u1), 0.5 * (v0 + v1))
        .map_err(|e| e.to_string())?;
    let du = [d1.d1u.x(), d1.d1u.y(), d1.d1u.z()];
    let dv = [d1.d1v.x(), d1.d1v.y(), d1.d1v.z()];
    let mut n = cross3(du, dv);
    if face.orientation() == Orientation::Reversed {
        n = n.map(|c| -c);
    }
    Ok(normalize3(n))
}

fn append_ribbon_triangles(samples: &[([f64; 3], [f64; 3])], out: &mut Vec<DisplayTriangle>) {
    for pair in samples.windows(2) {
        let (p0, n0) = pair[0];
        let (p1, n1) = pair[1];
        let seg = [p1[0] - p0[0], p1[1] - p0[1], p1[2] - p0[2]];
        if dot3(seg, seg) <= 1e-12 {
            continue;
        }
        let n = normalize3([n0[0] + n1[0], n0[1] + n1[1], n0[2] + n1[2]]);
        let side = normalize3(cross3(n, seg));
        let lift = |p: [f64; 3], sign: f64| {
            [
                p[0] + n[0] * MARK_SURFACE_LIFT_CM + side[0] * MARK_RIBBON_HALF_WIDTH_CM * sign,
                p[1] + n[1] * MARK_SURFACE_LIFT_CM + side[1] * MARK_RIBBON_HALF_WIDTH_CM * sign,
                p[2] + n[2] * MARK_SURFACE_LIFT_CM + side[2] * MARK_RIBBON_HALF_WIDTH_CM * sign,
            ]
        };
        let a = lift(p0, -1.0);
        let b = lift(p0, 1.0);
        let c = lift(p1, 1.0);
        let d = lift(p1, -1.0);
        out.push(DisplayTriangle {
            positions: [a, b, c],
            normals: Some([n, n, n]),
        });
        out.push(DisplayTriangle {
            positions: [a, c, d],
            normals: Some([n, n, n]),
        });
    }
}

fn dot3(a: [f64; 3], b: [f64; 3]) -> f64 {
    a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
}

fn dist3(a: [f64; 3], b: [f64; 3]) -> f64 {
    let d = [b[0] - a[0], b[1] - a[1], b[2] - a[2]];
    dot3(d, d).sqrt()
}

fn cross3(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [
        a[1] * b[2] - a[2] * b[1],
        a[2] * b[0] - a[0] * b[2],
        a[0] * b[1] - a[1] * b[0],
    ]
}

fn normalize3(v: [f64; 3]) -> [f64; 3] {
    let len = dot3(v, v).sqrt();
    if len <= 1e-12 {
        [0.0, 0.0, 1.0]
    } else {
        [v[0] / len, v[1] / len, v[2] / len]
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use martensite_tkprim::brep_prim_api::BRepPrimAPIMakeBox;

    #[test]
    fn box_connected_face_chain_reaches_all_six_faces() {
        let mut maker = BRepPrimAPIMakeBox::from_sizes(2.0, 2.0, 2.0).unwrap();
        let shape = maker.shape().unwrap().clone();
        let mut map = IndexedMapOfShape::default();
        topexp::map_shapes_of_type(&shape, ShapeEnum::Face, &mut map).unwrap();
        let all_faces: Vec<(Vec<u8>, Shape)> = (1..=map.extent())
            .map(|i| (vec![i as u8], map.find_key(i).unwrap().clone()))
            .collect();
        assert_eq!(all_faces.len(), 6);
        let chain = connected_face_chain_shapes(&all_faces, &all_faces[0].0).unwrap();
        assert_eq!(chain.len(), 6);
        assert_eq!(chain[0], all_faces[0].0);
    }
}
