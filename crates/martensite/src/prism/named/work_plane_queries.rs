// SPDX-License-Identifier: GPL-2.0-only
//! Exact analytic B-rep topology queries for Work Plane creation and picking (#3918).

use super::*;
use martensite_tkbrep::{
    brep_adaptor::{BRepAdaptorCurve, BRepAdaptorSurface},
    topods,
};
use martensite_tkg3d::adaptor3d::{Adaptor3dCurve, Adaptor3dSurface};
use martensite_tkmath::geom_abs::{CurveType, SurfaceType};
use oblikovati_model::{
    feature::work_axis::WorkAxisLine,
    identity::ReferenceEnvelope,
    sketch::{SketchPlane, SketchPlaneOptions},
};

/// One named B-rep vertex hit along a viewport picking ray.
#[derive(Clone, Debug, PartialEq)]
pub struct NamedKeyedVertexHit {
    pub key: Vec<u8>,
    pub point: [f64; 3],
    pub distance: f64,
    pub reach: f64,
}

impl NamedSolid {
    /// Query named vertices (kind 3) near a forward view ray in deterministic distance order.
    pub fn named_vertex_hits_on_ray(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        [origin, direction]: [[f64; 3]; 2],
        tolerance: f64,
    ) -> Result<Vec<NamedKeyedVertexHit>, String> {
        if !tolerance.is_finite() || tolerance <= 0.0 {
            return Err(format!(
                "vertex pick tolerance {tolerance}: expected a finite positive length"
            ));
        }
        let dir = normalize3(direction).ok_or_else(|| {
            format!("vertex ray direction {direction:?}: expected nonzero vector")
        })?;
        let mut hits = Vec::new();
        for key in self.subentity_reference_keys(history, document)? {
            let envelope = ReferenceEnvelope::decode(&key).map_err(|cause| cause.to_string())?;
            if envelope.kind != 3 {
                continue;
            }
            let point = self.subentity_point(history, document, &key)?;
            let rel = sub3(point, origin);
            let reach = dot3(rel, dir);
            if !reach.is_finite() || reach < 0.0 {
                continue;
            }
            let closest = [
                origin[0] + dir[0] * reach,
                origin[1] + dir[1] * reach,
                origin[2] + dir[2] * reach,
            ];
            let distance = len3(sub3(point, closest));
            if distance.is_finite() && distance <= tolerance {
                hits.push(NamedKeyedVertexHit {
                    key,
                    point,
                    distance,
                    reach,
                });
            }
        }
        hits.sort_by(|a, b| {
            a.distance
                .total_cmp(&b.distance)
                .then(a.reach.total_cmp(&b.reach))
                .then(a.key.cmp(&b.key))
        });
        Ok(hits)
    }

    /// Resolve a named B-rep vertex (kind 3) to its exact 3D model coordinates.
    pub fn resolve_topology_point(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<[f64; 3], String> {
        let envelope = ReferenceEnvelope::decode(key).map_err(|cause| cause.to_string())?;
        if envelope.kind != 3 {
            return Err(format!(
                "vertex reference kind {}: expected vertex kind 3",
                envelope.kind
            ));
        }
        self.subentity_point(history, document, key)
    }

    /// Resolve a named straight B-rep edge (kind 2) to an analytic axis line.
    pub fn resolve_straight_edge(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<WorkAxisLine, String> {
        let shape = self.native_edge(history, document, key)?;
        let edge = topods::edge(shape).map_err(|cause| cause.to_string())?;
        let curve = BRepAdaptorCurve::with_edge(&edge).map_err(|cause| cause.to_string())?;
        let kind = curve.get_type().map_err(|cause| cause.to_string())?;
        if kind != CurveType::Line {
            return Err(format!(
                "edge curve {kind:?}: expected a straight line edge"
            ));
        }
        let first = curve.first_parameter().map_err(|cause| cause.to_string())?;
        let last = curve.last_parameter().map_err(|cause| cause.to_string())?;
        let p0 = curve.value(first).map_err(|cause| cause.to_string())?;
        let p1 = curve.value(last).map_err(|cause| cause.to_string())?;
        WorkAxisLine::through_points([p0.x(), p0.y(), p0.z()], [p1.x(), p1.y(), p1.z()])
            .map_err(|cause| cause.to_string())
    }

    /// Resolve a toroidal face to its analytic midplane.
    pub fn resolve_torus_midplane(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<SketchPlane, String> {
        let face_shape = self.native_face(history, document, key)?;
        let native = topods::face(face_shape).map_err(|cause| cause.to_string())?;
        let adaptor =
            BRepAdaptorSurface::with_face(&native, false).map_err(|cause| cause.to_string())?;
        let kind = adaptor.get_type().map_err(|cause| cause.to_string())?;
        if kind != SurfaceType::Torus {
            return Err(format!("face surface {kind:?}: expected a torus surface"));
        }
        let torus = adaptor.torus().map_err(|cause| cause.to_string())?;
        let center = *torus.position().location().coordinates();
        let axis = *torus.position().direction().coordinates();
        SketchPlane::from_origin_normal(center, axis, SketchPlaneOptions::default())
            .map_err(|cause| cause.to_string())
    }

    /// Resolve an analytic plane tangent to a curved face and passing through `point`.
    pub fn resolve_tangent_plane_through_point(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        face_key: &[u8],
        point: [f64; 3],
    ) -> Result<SketchPlane, String> {
        let face_shape = self.native_face(history, document, face_key)?;
        let native = topods::face(face_shape).map_err(|cause| cause.to_string())?;
        let adaptor =
            BRepAdaptorSurface::with_face(&native, false).map_err(|cause| cause.to_string())?;
        match adaptor.get_type().map_err(|cause| cause.to_string())? {
            SurfaceType::Cylinder => {
                let cyl = adaptor.cylinder().map_err(|cause| cause.to_string())?;
                tangent_cylinder_point(
                    *cyl.position().location().coordinates(),
                    *cyl.position().direction().coordinates(),
                    cyl.radius(),
                    point,
                )
            }
            SurfaceType::Sphere => {
                let sphere = adaptor.sphere().map_err(|cause| cause.to_string())?;
                tangent_sphere_point(*sphere.location().coordinates(), sphere.radius(), point)
            }
            SurfaceType::Torus => {
                let torus = adaptor.torus().map_err(|cause| cause.to_string())?;
                tangent_torus_point(
                    *torus.position().location().coordinates(),
                    *torus.position().direction().coordinates(),
                    torus.major_radius(),
                    torus.minor_radius(),
                    point,
                )
            }
            other => Err(format!(
                "tangent surface {other:?}: expected a cylindrical, spherical, or toroidal face"
            )),
        }
    }

    /// Resolve an analytic plane tangent to a curved face and containing `line`.
    pub fn resolve_tangent_plane_through_line(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        face_key: &[u8],
        line: WorkAxisLine,
    ) -> Result<SketchPlane, String> {
        let face_shape = self.native_face(history, document, face_key)?;
        let native = topods::face(face_shape).map_err(|cause| cause.to_string())?;
        let adaptor =
            BRepAdaptorSurface::with_face(&native, false).map_err(|cause| cause.to_string())?;
        match adaptor.get_type().map_err(|cause| cause.to_string())? {
            SurfaceType::Cylinder => {
                let cyl = adaptor.cylinder().map_err(|cause| cause.to_string())?;
                let axis_dir = *cyl.position().direction().coordinates();
                if dot3(axis_dir, line.direction()).abs() < 1.0 - 1e-6 {
                    return Err(
                        "tangent through edge: expected edge parallel to cylinder axis".into(),
                    );
                }
                tangent_cylinder_point(
                    *cyl.position().location().coordinates(),
                    axis_dir,
                    cyl.radius(),
                    line.origin(),
                )
            }
            SurfaceType::Sphere => {
                let sphere = adaptor.sphere().map_err(|cause| cause.to_string())?;
                let center = *sphere.location().coordinates();
                let rel = sub3(center, line.origin());
                let along = dot3(rel, line.direction());
                let closest = add3(line.origin(), scale3(line.direction(), along));
                tangent_sphere_point(center, sphere.radius(), closest)
            }
            other => Err(format!(
                "tangent surface {other:?}: expected a cylindrical or spherical face"
            )),
        }
    }

    /// Resolve an analytic plane tangent to a curved face and parallel to `plane`.
    pub fn resolve_tangent_plane_parallel(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        face_key: &[u8],
        plane: SketchPlane,
    ) -> Result<SketchPlane, String> {
        let face_shape = self.native_face(history, document, face_key)?;
        let native = topods::face(face_shape).map_err(|cause| cause.to_string())?;
        let adaptor =
            BRepAdaptorSurface::with_face(&native, false).map_err(|cause| cause.to_string())?;
        let normal = plane.normal();
        let origin = match adaptor.get_type().map_err(|cause| cause.to_string())? {
            SurfaceType::Cylinder => {
                let cyl = adaptor.cylinder().map_err(|cause| cause.to_string())?;
                let axis = *cyl.position().direction().coordinates();
                if dot3(normal, axis).abs() > 1e-6 {
                    return Err(
                        "tangent parallel plane: expected reference plane parallel to cylinder axis"
                            .into(),
                    );
                }
                add3(
                    *cyl.position().location().coordinates(),
                    scale3(normal, cyl.radius()),
                )
            }
            SurfaceType::Sphere => {
                let sphere = adaptor.sphere().map_err(|cause| cause.to_string())?;
                add3(
                    *sphere.location().coordinates(),
                    scale3(normal, sphere.radius()),
                )
            }
            SurfaceType::Torus => {
                let torus = adaptor.torus().map_err(|cause| cause.to_string())?;
                let center = *torus.position().location().coordinates();
                let axis = *torus.position().direction().coordinates();
                let axial = dot3(normal, axis).abs();
                if axial >= 1.0 - 1e-6 {
                    add3(center, scale3(normal, torus.minor_radius()))
                } else {
                    let radial = normalize3(sub3(normal, scale3(axis, dot3(normal, axis))))
                        .ok_or("torus tangent parallel: degenerate normal")?;
                    add3(
                        add3(center, scale3(radial, torus.major_radius())),
                        scale3(normal, torus.minor_radius()),
                    )
                }
            }
            other => {
                return Err(format!(
                    "tangent surface {other:?}: expected a cylindrical, spherical, or toroidal face"
                ));
            }
        };
        let mut recipe = plane.recipe();
        recipe.origin = origin;
        SketchPlane::from_recipe(recipe, plane.options()).map_err(|cause| cause.to_string())
    }
}

fn tangent_cylinder_point(
    axis_origin: [f64; 3],
    axis_dir: [f64; 3],
    radius: f64,
    point: [f64; 3],
) -> Result<SketchPlane, String> {
    let rel = sub3(point, axis_origin);
    let along = dot3(rel, axis_dir);
    let radial = sub3(rel, scale3(axis_dir, along));
    let dist = len3(radial);
    if dist < radius - 1e-6 {
        return Err("tangent through point: point lies inside cylinder".into());
    }
    let u_r = normalize3(radial).ok_or("tangent through point: point lies on cylinder axis")?;
    let normal = if (dist - radius).abs() <= 1e-5 {
        u_r
    } else {
        let cos_t = (radius / dist).clamp(-1.0, 1.0);
        let sin_t = (1.0 - cos_t * cos_t).max(0.0).sqrt();
        let u_t = normalize3(cross3(axis_dir, u_r))
            .ok_or("tangent through point: degenerate radial frame")?;
        add3(scale3(u_r, cos_t), scale3(u_t, sin_t))
    };
    SketchPlane::from_origin_normal(point, normal, SketchPlaneOptions::default())
        .map_err(|cause| cause.to_string())
}

fn tangent_sphere_point(
    center: [f64; 3],
    radius: f64,
    point: [f64; 3],
) -> Result<SketchPlane, String> {
    let radial = sub3(point, center);
    let dist = len3(radial);
    if dist < radius - 1e-6 {
        return Err("tangent through point: point lies inside sphere".into());
    }
    let normal = normalize3(radial).ok_or("tangent through point: point is at sphere center")?;
    SketchPlane::from_origin_normal(point, normal, SketchPlaneOptions::default())
        .map_err(|cause| cause.to_string())
}

fn tangent_torus_point(
    center: [f64; 3],
    axis: [f64; 3],
    major: f64,
    minor: f64,
    point: [f64; 3],
) -> Result<SketchPlane, String> {
    let rel = sub3(point, center);
    let along = dot3(rel, axis);
    let radial = sub3(rel, scale3(axis, along));
    let r_unit = normalize3(radial).ok_or("tangent through point: point lies on torus axis")?;
    let tube_center = add3(center, scale3(r_unit, major));
    tangent_sphere_point(tube_center, minor, point)
}

fn dot3(a: [f64; 3], b: [f64; 3]) -> f64 {
    a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
}

fn sub3(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [a[0] - b[0], a[1] - b[1], a[2] - b[2]]
}

fn add3(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [a[0] + b[0], a[1] + b[1], a[2] + b[2]]
}

fn scale3(a: [f64; 3], s: f64) -> [f64; 3] {
    [a[0] * s, a[1] * s, a[2] * s]
}

fn cross3(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [
        a[1] * b[2] - a[2] * b[1],
        a[2] * b[0] - a[0] * b[2],
        a[0] * b[1] - a[1] * b[0],
    ]
}

fn len3(v: [f64; 3]) -> f64 {
    dot3(v, v).sqrt()
}

fn normalize3(v: [f64; 3]) -> Option<[f64; 3]> {
    let length = len3(v);
    (length.is_finite() && length > 1e-12).then(|| scale3(v, 1.0 / length))
}
