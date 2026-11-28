// SPDX-License-Identifier: GPL-2.0-only
//! Planar face boundary extraction and half-plane edge extension geometry (#3918).
use super::thicken_offset_chain::collect_shapes;
use martensite_tkbrep::{
    brep_adaptor::{BRepAdaptorCurve, BRepAdaptorSurface},
    topods::{self, Shape},
};
use martensite_tkg3d::{
    adaptor3d::{Adaptor3dCurve, Adaptor3dSurface},
    top_abs::{Orientation, ShapeEnum},
};
use martensite_tkmath::{geom_abs::SurfaceType, gp::Point3};
use martensite_tktopalgo::brep_builder_api::{BRepBuilderAPIMakeFace, BRepBuilderAPIMakePolygon};
use oblikovati_model::feature::extend_surface::{ExtendSurfaceMethod, ResolvedExtendSurfaceExtent};

pub(super) const ENDPOINT_TOL: f64 = 1.0e-4;

struct PlaneBasis {
    origin: [f64; 3],
    u: [f64; 3],
    v: [f64; 3],
}

impl PlaneBasis {
    fn from_planar_face(face: &Shape) -> Result<Self, String> {
        let native = topods::face(face).map_err(|cause| cause.to_string())?;
        let adaptor =
            BRepAdaptorSurface::with_face(&native, false).map_err(|cause| cause.to_string())?;
        if adaptor.get_type().map_err(|cause| cause.to_string())? != SurfaceType::Plane {
            return Err("extend surface: only planar surface faces are supported".into());
        }
        let plane = adaptor.plane().map_err(|cause| cause.to_string())?;
        let origin = *plane.location().coordinates();
        let mut normal = *plane.position().direction().coordinates();
        let u = *plane.x_axis().direction().coordinates();
        if face.orientation() == Orientation::Reversed {
            normal = [-normal[0], -normal[1], -normal[2]];
        }
        let v = cross(normal, u);
        let v_len = length3(v);
        if v_len <= 1.0e-12 {
            return Err("extend surface plane: degenerate face frame".into());
        }
        Ok(Self {
            origin,
            u,
            v: [v[0] / v_len, v[1] / v_len, v[2] / v_len],
        })
    }

    fn project(&self, pt: [f64; 3]) -> [f64; 2] {
        let rel = sub3(pt, self.origin);
        [dot3(rel, self.u), dot3(rel, self.v)]
    }

    fn unproject(&self, uv: [f64; 2]) -> [f64; 3] {
        [
            self.origin[0] + uv[0] * self.u[0] + uv[1] * self.v[0],
            self.origin[1] + uv[0] * self.u[1] + uv[1] * self.v[1],
            self.origin[2] + uv[0] * self.u[2] + uv[1] * self.v[2],
        ]
    }
}

pub(super) fn extend_planar_face(
    face: &Shape,
    selected_edges: &[Shape],
    extent: &ResolvedExtendSurfaceExtent,
    method: ExtendSurfaceMethod,
) -> Result<Vec<Shape>, String> {
    let basis = PlaneBasis::from_planar_face(face)?;
    let raw_edges = collect_shapes(face, ShapeEnum::Edge)?;
    let (mut pts_2d, mut edge_selected) = ordered_ccw_polygon(&basis, &raw_edges, selected_edges)?;
    let n = pts_2d.len();
    if n < 3 {
        return Err(format!(
            "extend surface polygon vertices {n}: expected at least 3"
        ));
    }
    if signed_area_2d(&pts_2d) < 0.0 {
        pts_2d.reverse();
        let old = edge_selected;
        edge_selected = (0..n).map(|k| old[(2 * n - 2 - k) % n]).collect();
    }
    let mut normals = Vec::with_capacity(n);
    let mut offsets = Vec::with_capacity(n);
    for k in 0..n {
        let p0 = pts_2d[k];
        let p1 = pts_2d[(k + 1) % n];
        let dx = p1[0] - p0[0];
        let dy = p1[1] - p0[1];
        let len = dx.hypot(dy);
        if len <= 1.0e-12 {
            return Err("extend surface edge: degenerate zero-length boundary edge".into());
        }
        let n_out = [dy / len, -dx / len];
        let delta = if edge_selected[k] {
            resolve_edge_distance(&basis, p0, p1, n_out, extent)?
        } else {
            0.0
        };
        normals.push(n_out);
        offsets.push(n_out[0] * p0[0] + n_out[1] * p0[1] + delta);
    }
    let mut new_pts_2d = Vec::with_capacity(n);
    for k in 0..n {
        let prev = (k + n - 1) % n;
        let pt = intersect_lines_2d(
            normals[prev],
            offsets[prev],
            normals[k],
            offsets[k],
            pts_2d[k],
        );
        new_pts_2d.push(pt);
    }
    let orig_3d: Vec<[f64; 3]> = pts_2d.iter().map(|uv| basis.unproject(*uv)).collect();
    let new_3d: Vec<[f64; 3]> = new_pts_2d.iter().map(|uv| basis.unproject(*uv)).collect();
    match method {
        ExtendSurfaceMethod::Stretch => Ok(vec![make_planar_polygon_face(&new_3d)?]),
        ExtendSurfaceMethod::Extend => {
            let mut out = Vec::with_capacity(1 + edge_selected.iter().filter(|&&s| s).count());
            out.push(make_planar_polygon_face(&orig_3d)?);
            for k in 0..n {
                if !edge_selected[k] {
                    continue;
                }
                let next = (k + 1) % n;
                let quad = [orig_3d[k], orig_3d[next], new_3d[next], new_3d[k]];
                out.push(make_planar_polygon_face(&quad)?);
            }
            Ok(out)
        }
    }
}

fn resolve_edge_distance(
    basis: &PlaneBasis,
    p0: [f64; 2],
    p1: [f64; 2],
    n_out_2d: [f64; 2],
    extent: &ResolvedExtendSurfaceExtent,
) -> Result<f64, String> {
    match extent {
        ResolvedExtendSurfaceExtent::Distance(distance) => {
            if !distance.is_finite() || *distance <= 0.0 {
                return Err(format!(
                    "extend surface distance {distance}: expected a positive finite length"
                ));
            }
            Ok(*distance)
        }
        ResolvedExtendSurfaceExtent::ToPlane(target) => {
            let mid_uv = [0.5 * (p0[0] + p1[0]), 0.5 * (p0[1] + p1[1])];
            let mid_3d = basis.unproject(mid_uv);
            let dir_3d = [
                n_out_2d[0] * basis.u[0] + n_out_2d[1] * basis.v[0],
                n_out_2d[0] * basis.u[1] + n_out_2d[1] * basis.v[1],
                n_out_2d[0] * basis.u[2] + n_out_2d[1] * basis.v[2],
            ];
            let denom = dot3(dir_3d, target.normal());
            if denom.abs() <= 1.0e-6 {
                return Err(
                    "extend surface To target: selected edge is parallel to the termination plane"
                        .into(),
                );
            }
            let t = dot3(sub3(target.recipe().origin, mid_3d), target.normal()) / denom;
            if !t.is_finite() || t <= 1.0e-6 {
                return Err(
                    "extend surface To target: termination plane does not lie ahead of the selected edge"
                        .into(),
                );
            }
            Ok(t)
        }
    }
}

fn ordered_ccw_polygon(
    basis: &PlaneBasis,
    raw_edges: &[Shape],
    selected_edges: &[Shape],
) -> Result<(Vec<[f64; 2]>, Vec<bool>), String> {
    let mut segs: Vec<([f64; 3], [f64; 3], bool)> = Vec::with_capacity(raw_edges.len());
    for edge in raw_edges {
        let [a, b] = edge_endpoints(edge)?;
        let selected = selected_edges.iter().any(|sel| sel.is_same(edge));
        segs.push((a, b, selected));
    }
    let Some((first_a, first_b, first_sel)) = segs.first().copied() else {
        return Err("extend surface face: expected at least one boundary edge".into());
    };
    let mut used = vec![false; segs.len()];
    used[0] = true;
    let mut pts = vec![basis.project(first_a)];
    let mut sels = vec![first_sel];
    let mut current_end = first_b;
    for _ in 1..segs.len() {
        let next = segs.iter().enumerate().find_map(|(i, (a, b, sel))| {
            if used[i] {
                return None;
            }
            if points_near(current_end, *a) {
                Some((i, *a, *b, *sel))
            } else if points_near(current_end, *b) {
                Some((i, *b, *a, *sel))
            } else {
                None
            }
        });
        let Some((idx, start, end, sel)) = next else {
            return Err("extend surface face: boundary edges do not form a closed loop".into());
        };
        used[idx] = true;
        pts.push(basis.project(start));
        sels.push(sel);
        current_end = end;
    }
    if !points_near(current_end, first_a) {
        return Err("extend surface face: boundary loop is not closed".into());
    }
    Ok((pts, sels))
}

fn make_planar_polygon_face(points: &[[f64; 3]]) -> Result<Shape, String> {
    let mut poly = BRepBuilderAPIMakePolygon::new();
    for pt in points {
        poly.add_point(Point3::new(pt[0], pt[1], pt[2]))
            .map_err(|cause| cause.to_string())?;
    }
    poly.close().map_err(|cause| cause.to_string())?;
    let wire = poly.wire().map_err(|cause| cause.to_string())?;
    let face: Shape = BRepBuilderAPIMakeFace::from_wire(&wire, true)
        .map_err(|cause| cause.to_string())?
        .face()
        .map_err(|cause| cause.to_string())?
        .into();
    Ok(face)
}

pub(super) fn edge_endpoints(edge: &Shape) -> Result<[[f64; 3]; 2], String> {
    let native = topods::edge(edge).map_err(|cause| cause.to_string())?;
    let curve = BRepAdaptorCurve::with_edge(&native).map_err(|cause| cause.to_string())?;
    let first = curve.first_parameter().map_err(|cause| cause.to_string())?;
    let last = curve.last_parameter().map_err(|cause| cause.to_string())?;
    let p0 = curve.value(first).map_err(|cause| cause.to_string())?;
    let p1 = curve.value(last).map_err(|cause| cause.to_string())?;
    Ok([[p0.x(), p0.y(), p0.z()], [p1.x(), p1.y(), p1.z()]])
}

fn intersect_lines_2d(
    n1: [f64; 2],
    c1: f64,
    n2: [f64; 2],
    c2: f64,
    fallback: [f64; 2],
) -> [f64; 2] {
    let det = n1[0] * n2[1] - n1[1] * n2[0];
    if det.abs() <= 1.0e-7 {
        let shift = (c1 - (n1[0] * fallback[0] + n1[1] * fallback[1]))
            .max(c2 - (n2[0] * fallback[0] + n2[1] * fallback[1]));
        return [fallback[0] + shift * n2[0], fallback[1] + shift * n2[1]];
    }
    [
        (c1 * n2[1] - n1[1] * c2) / det,
        (n1[0] * c2 - c1 * n2[0]) / det,
    ]
}

fn signed_area_2d(pts: &[[f64; 2]]) -> f64 {
    let n = pts.len();
    (0..n)
        .map(|i| {
            let a = pts[i];
            let b = pts[(i + 1) % n];
            a[0] * b[1] - b[0] * a[1]
        })
        .sum::<f64>()
        * 0.5
}

pub(super) fn points_near(a: [f64; 3], b: [f64; 3]) -> bool {
    length3(sub3(a, b)) <= ENDPOINT_TOL
}

fn sub3(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [a[0] - b[0], a[1] - b[1], a[2] - b[2]]
}

fn dot3(a: [f64; 3], b: [f64; 3]) -> f64 {
    a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
}

fn cross(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [
        a[1] * b[2] - a[2] * b[1],
        a[2] * b[0] - a[0] * b[2],
        a[0] * b[1] - a[1] * b[0],
    ]
}

fn length3(v: [f64; 3]) -> f64 {
    dot3(v, v).sqrt()
}
