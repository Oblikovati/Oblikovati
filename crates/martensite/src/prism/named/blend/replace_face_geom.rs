// SPDX-License-Identifier: GPL-2.0-only
//! Analytic plane, bounding-box, and vector helpers for B-rep Replace Face (#3918).
use martensite_tkbrep::{
    brep_adaptor::BRepAdaptorSurface,
    topods::{self, Shape},
};
use martensite_tkg3d::adaptor3d::Adaptor3dSurface;
use martensite_tkgeombase::gprop::GProps;
use martensite_tkmath::{bnd::BoundingBox3, geom_abs::SurfaceType, gp::Point3};
use martensite_tktopalgo::{
    brep_bnd_lib::BRepBndLib,
    brep_builder_api::{BRepBuilderAPIMakeFace, BRepBuilderAPIMakePolygon},
    brep_gprop::BRepGProp,
};

pub(super) type PlaneOriginNormal = ([f64; 3], [f64; 3]);

pub(super) fn face_plane_point_normal(face: &Shape) -> Result<Option<PlaneOriginNormal>, String> {
    let native = topods::face(face).map_err(|cause| cause.to_string())?;
    let adaptor =
        BRepAdaptorSurface::with_face(&native, false).map_err(|cause| cause.to_string())?;
    if adaptor.get_type().map_err(|cause| cause.to_string())? != SurfaceType::Plane {
        return Ok(None);
    }
    let plane = adaptor.plane().map_err(|cause| cause.to_string())?;
    let origin = *plane.position().location().coordinates();
    let dir = *plane.position().direction().coordinates();
    Ok(Some((origin, dir)))
}

pub(super) fn build_large_plane_face(
    origin: [f64; 3],
    normal: [f64; 3],
    span: f64,
) -> Result<Shape, String> {
    let (u, v) = tangent_basis(normal);
    let corners = [
        add(add(origin, scale(u, -span)), scale(v, -span)),
        add(add(origin, scale(u, span)), scale(v, -span)),
        add(add(origin, scale(u, span)), scale(v, span)),
        add(add(origin, scale(u, -span)), scale(v, span)),
    ];
    let mut poly = BRepBuilderAPIMakePolygon::new();
    for corner in corners {
        poly.add_point(Point3::new(corner[0], corner[1], corner[2]))
            .map_err(|cause| cause.to_string())?;
    }
    poly.close().map_err(|cause| cause.to_string())?;
    let wire = poly.wire().map_err(|cause| cause.to_string())?;
    let face = BRepBuilderAPIMakeFace::from_wire(&wire, true)
        .map_err(|cause| cause.to_string())?
        .face()
        .map_err(|cause| cause.to_string())?;
    Ok(face.into_shape())
}

pub(super) fn solid_center_of_mass(solid: &Shape) -> Result<[f64; 3], String> {
    let mut props = GProps::default();
    BRepGProp::volume_properties(solid, &mut props, true, false, false)
        .map_err(|cause| cause.to_string())?;
    let com = props.centre_of_mass();
    Ok([com.x(), com.y(), com.z()])
}

pub(super) fn shape_bounds(shape: &Shape) -> Result<[f64; 6], String> {
    let mut bbox = BoundingBox3::default();
    BRepBndLib::add_optimal(shape, &mut bbox, false, false).map_err(|cause| cause.to_string())?;
    bbox.get().map_err(|cause| cause.to_string())
}

pub(super) fn combined_bounds(shapes: &[Shape]) -> Result<[f64; 6], String> {
    let mut bbox = BoundingBox3::default();
    for shape in shapes {
        BRepBndLib::add_optimal(shape, &mut bbox, false, false)
            .map_err(|cause| cause.to_string())?;
    }
    bbox.get().map_err(|cause| cause.to_string())
}

pub(super) fn boxes_overlap_in_tangent_plane(a: [f64; 6], b: [f64; 6], normal: [f64; 3]) -> bool {
    let (u, v) = tangent_basis(normal);
    let (au0, au1) = projected_interval(a, u);
    let (bu0, bu1) = projected_interval(b, u);
    let (av0, av1) = projected_interval(a, v);
    let (bv0, bv1) = projected_interval(b, v);
    let tol = 1.0e-4;
    au1 + tol >= bu0 && bu1 + tol >= au0 && av1 + tol >= bv0 && bv1 + tol >= av0
}

pub(super) fn signed_plane_distances(
    bbox: [f64; 6],
    p_new: [f64; 3],
    n_new: [f64; 3],
    alignment: f64,
) -> (f64, f64) {
    let mut t_min = f64::INFINITY;
    let mut t_max = f64::NEG_INFINITY;
    for &x in &[bbox[0], bbox[3]] {
        for &y in &[bbox[1], bbox[4]] {
            for &z in &[bbox[2], bbox[5]] {
                let t = dot(sub(p_new, [x, y, z]), n_new) / alignment;
                t_min = t_min.min(t);
                t_max = t_max.max(t);
            }
        }
    }
    (t_min, t_max)
}

pub(super) fn projected_extent(bbox: [f64; 6], axis: [f64; 3]) -> f64 {
    let (lo, hi) = projected_interval(bbox, axis);
    (hi - lo).max(0.0)
}

pub(super) fn box_diagonal(b: [f64; 6]) -> f64 {
    let dx = b[3] - b[0];
    let dy = b[4] - b[1];
    let dz = b[5] - b[2];
    (dx * dx + dy * dy + dz * dz).sqrt()
}

pub(super) fn box_center(b: [f64; 6]) -> [f64; 3] {
    [
        0.5 * (b[0] + b[3]),
        0.5 * (b[1] + b[4]),
        0.5 * (b[2] + b[5]),
    ]
}

fn projected_interval(bbox: [f64; 6], axis: [f64; 3]) -> (f64, f64) {
    let mut lo = f64::INFINITY;
    let mut hi = f64::NEG_INFINITY;
    for &x in &[bbox[0], bbox[3]] {
        for &y in &[bbox[1], bbox[4]] {
            for &z in &[bbox[2], bbox[5]] {
                let val = dot([x, y, z], axis);
                lo = lo.min(val);
                hi = hi.max(val);
            }
        }
    }
    (lo, hi)
}

fn tangent_basis(normal: [f64; 3]) -> ([f64; 3], [f64; 3]) {
    let helper = if normal[2].abs() < 0.9 {
        [0.0, 0.0, 1.0]
    } else {
        [1.0, 0.0, 0.0]
    };
    let u = normalize(cross(helper, normal));
    let v = normalize(cross(normal, u));
    (u, v)
}

pub(super) fn dot(a: [f64; 3], b: [f64; 3]) -> f64 {
    a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
}

pub(super) fn sub(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [a[0] - b[0], a[1] - b[1], a[2] - b[2]]
}

fn add(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [a[0] + b[0], a[1] + b[1], a[2] + b[2]]
}

fn scale(a: [f64; 3], s: f64) -> [f64; 3] {
    [a[0] * s, a[1] * s, a[2] * s]
}

fn cross(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [
        a[1] * b[2] - a[2] * b[1],
        a[2] * b[0] - a[0] * b[2],
        a[0] * b[1] - a[1] * b[0],
    ]
}

fn normalize(v: [f64; 3]) -> [f64; 3] {
    let len = dot(v, v).sqrt();
    if len > 1.0e-12 {
        scale(v, 1.0 / len)
    } else {
        [1.0, 0.0, 0.0]
    }
}
