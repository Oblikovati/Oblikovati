// SPDX-License-Identifier: GPL-2.0-only
//! Tangent-continuous face chaining and analytic surface normal queries for Thicken/Offset (#3918).
use super::rigid_motion::{RigidMotion, compose};
use super::*;
use martensite_tkbrep::{
    brep_adaptor::{BRepAdaptorCurve, BRepAdaptorSurface},
    topexp,
    topods::{self, Builder, Face, Shape, Solid},
    toptools::IndexedMapOfShape,
};
use martensite_tkfillet::{brep_fillet_api::BRepFilletAPIMakeFillet, chfi3d::FilletShape};
use martensite_tkg3d::{
    adaptor3d::{Adaptor3dCurve, Adaptor3dSurface},
    top_abs::{Orientation, ShapeEnum},
};
use martensite_tkmath::{
    bnd::BoundingBox3,
    geom_abs::SurfaceType,
    gp::{Axis2, Direction3, Point3},
};
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakeCylinder;
use martensite_tktopalgo::{
    brep_bnd_lib::BRepBndLib,
    brep_builder_api::{BRepBuilderAPIMakeShape, BRepBuilderAPITransform},
};
use std::collections::{BTreeSet, VecDeque};

impl NamedSolid {
    /// Expand a seed face to all edge-adjacent G1 tangent-continuous faces of matching curvature class.
    /// Example: `body.tangent_face_chain(&history, document, &seed_face)?`.
    pub fn tangent_face_chain(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        seed_face: &[u8],
    ) -> Result<Vec<Vec<u8>>, String> {
        let keys = self.face_reference_keys(history, document)?;
        let Some(seed_idx) = keys.iter().position(|key| key.as_slice() == seed_face) else {
            return Err("thicken face chain: expected a live named face on the target body".into());
        };
        let faces: Vec<Shape> = keys
            .iter()
            .map(|key| self.native_face(history, document, key).cloned())
            .collect::<Result<_, _>>()?;
        let seed_curved = is_curved_face(&faces[seed_idx])?;
        let face_edges: Vec<Vec<Shape>> = faces
            .iter()
            .map(|face| collect_shapes(face, ShapeEnum::Edge))
            .collect::<Result<_, _>>()?;

        let mut visited = BTreeSet::new();
        let mut queue = VecDeque::new();
        visited.insert(seed_idx);
        queue.push_back(seed_idx);

        while let Some(current) = queue.pop_front() {
            for (other, other_edges) in face_edges.iter().enumerate() {
                if visited.contains(&other) || is_curved_face(&faces[other])? != seed_curved {
                    continue;
                }
                let Some(shared_edge) = face_edges[current]
                    .iter()
                    .find(|edge| other_edges.iter().any(|candidate| edge.is_same(candidate)))
                else {
                    continue;
                };
                if faces_tangent_at_edge(&faces[current], &faces[other], shared_edge)? {
                    visited.insert(other);
                    queue.push_back(other);
                }
            }
        }
        let mut ordered = vec![keys[seed_idx].clone()];
        for idx in visited {
            if idx != seed_idx {
                ordered.push(keys[idx].clone());
            }
        }
        Ok(ordered)
    }
}

pub(super) fn planar_outward_normal(face: &Shape) -> Result<Option<[f64; 3]>, String> {
    let native = topods::face(face).map_err(|cause| cause.to_string())?;
    let adaptor =
        BRepAdaptorSurface::with_face(&native, false).map_err(|cause| cause.to_string())?;
    if adaptor.get_type().map_err(|cause| cause.to_string())? != SurfaceType::Plane {
        return Ok(None);
    }
    let plane = adaptor.plane().map_err(|cause| cause.to_string())?;
    let dir = *plane.position().direction().coordinates();
    let sign = match face.orientation() {
        Orientation::Reversed => -1.0,
        _ => 1.0,
    };
    Ok(Some([dir[0] * sign, dir[1] * sign, dir[2] * sign]))
}

pub(super) fn collect_shapes(shape: &Shape, kind: ShapeEnum) -> Result<Vec<Shape>, String> {
    let mut map = IndexedMapOfShape::default();
    topexp::map_shapes_of_type(shape, kind, &mut map).map_err(|cause| cause.to_string())?;
    (1..=map.extent())
        .map(|index| {
            map.find_key(index)
                .cloned()
                .map_err(|cause| cause.to_string())
        })
        .collect()
}

fn is_curved_face(face: &Shape) -> Result<bool, String> {
    let native = topods::face(face).map_err(|cause| cause.to_string())?;
    let adaptor =
        BRepAdaptorSurface::with_face(&native, false).map_err(|cause| cause.to_string())?;
    let kind = adaptor.get_type().map_err(|cause| cause.to_string())?;
    Ok(kind != SurfaceType::Plane)
}

fn faces_tangent_at_edge(face_a: &Shape, face_b: &Shape, edge: &Shape) -> Result<bool, String> {
    let native_edge = topods::edge(edge).map_err(|cause| cause.to_string())?;
    let curve = BRepAdaptorCurve::with_edge(&native_edge).map_err(|cause| cause.to_string())?;
    let first = curve.first_parameter().map_err(|cause| cause.to_string())?;
    let last = curve.last_parameter().map_err(|cause| cause.to_string())?;
    let point = curve
        .value(0.5 * (first + last))
        .map_err(|cause| cause.to_string())?;
    let Some(na) = face_normal_near(face_a, point)? else {
        return Ok(false);
    };
    let Some(nb) = face_normal_near(face_b, point)? else {
        return Ok(false);
    };
    let dot = (na[0] * nb[0] + na[1] * nb[1] + na[2] * nb[2]).abs();
    Ok(dot > 0.995)
}

fn face_normal_near(face: &Shape, point: Point3) -> Result<Option<[f64; 3]>, String> {
    let native: Face = topods::face(face).map_err(|cause| cause.to_string())?;
    let adaptor =
        BRepAdaptorSurface::with_face(&native, false).map_err(|cause| cause.to_string())?;
    let kind = adaptor.get_type().map_err(|cause| cause.to_string())?;
    let raw = match kind {
        SurfaceType::Plane => *adaptor
            .plane()
            .map_err(|cause| cause.to_string())?
            .position()
            .direction()
            .coordinates(),
        SurfaceType::Cylinder => {
            let cyl = adaptor.cylinder().map_err(|cause| cause.to_string())?;
            let axis_loc = *cyl.position().location().coordinates();
            let axis_dir = *cyl.position().direction().coordinates();
            let rel = [
                point.x() - axis_loc[0],
                point.y() - axis_loc[1],
                point.z() - axis_loc[2],
            ];
            let along = rel[0] * axis_dir[0] + rel[1] * axis_dir[1] + rel[2] * axis_dir[2];
            let Some(unit) = normalize([
                rel[0] - along * axis_dir[0],
                rel[1] - along * axis_dir[1],
                rel[2] - along * axis_dir[2],
            ]) else {
                return Ok(None);
            };
            unit
        }
        SurfaceType::Torus => {
            let torus = adaptor.torus().map_err(|cause| cause.to_string())?;
            let center = *torus.position().location().coordinates();
            let axis = *torus.position().direction().coordinates();
            let major = torus.major_radius();
            let rel = [
                point.x() - center[0],
                point.y() - center[1],
                point.z() - center[2],
            ];
            let along = rel[0] * axis[0] + rel[1] * axis[1] + rel[2] * axis[2];
            let Some(r_unit) = normalize([
                rel[0] - along * axis[0],
                rel[1] - along * axis[1],
                rel[2] - along * axis[2],
            ]) else {
                return Ok(None);
            };
            let tube_center = [
                center[0] + r_unit[0] * major,
                center[1] + r_unit[1] * major,
                center[2] + r_unit[2] * major,
            ];
            let Some(unit) = normalize([
                point.x() - tube_center[0],
                point.y() - tube_center[1],
                point.z() - tube_center[2],
            ]) else {
                return Ok(None);
            };
            unit
        }
        SurfaceType::Sphere => {
            let sphere = adaptor.sphere().map_err(|cause| cause.to_string())?;
            let center = *sphere.location().coordinates();
            let Some(unit) = normalize([
                point.x() - center[0],
                point.y() - center[1],
                point.z() - center[2],
            ]) else {
                return Ok(None);
            };
            unit
        }
        _ => return Ok(None),
    };
    Ok(Some(raw))
}

fn normalize(v: [f64; 3]) -> Option<[f64; 3]> {
    let len = (v[0] * v[0] + v[1] * v[1] + v[2] * v[2]).sqrt();
    (len > 1.0e-12).then(|| [v[0] / len, v[1] / len, v[2] / len])
}

/// Exact analytic offset for a coaxial rounded cylinder (Cylinder + tangent Torus fillet)
/// when Automatic Blending thickens the curved chain, avoiding BOP sliver seams during meshing.
pub(super) fn try_coaxial_rounded_cylinder_blend(
    body_shape: &Shape,
    selected_faces: &[Shape],
    signed_offset: f64,
) -> Result<Option<Shape>, String> {
    if selected_faces.len() != 2 || !signed_offset.is_finite() || signed_offset == 0.0 {
        return Ok(None);
    }
    let mut cylinder = None;
    let mut torus = None;
    for face in selected_faces {
        let native = topods::face(face).map_err(|cause| cause.to_string())?;
        let adaptor =
            BRepAdaptorSurface::with_face(&native, false).map_err(|cause| cause.to_string())?;
        match adaptor.get_type().map_err(|cause| cause.to_string())? {
            SurfaceType::Cylinder => {
                cylinder = Some(adaptor.cylinder().map_err(|cause| cause.to_string())?);
            }
            SurfaceType::Torus => {
                torus = Some(adaptor.torus().map_err(|cause| cause.to_string())?);
            }
            _ => return Ok(None),
        }
    }
    let (Some(cyl), Some(tor)) = (cylinder, torus) else {
        return Ok(None);
    };
    let cyl_loc = *cyl.position().location().coordinates();
    let cyl_dir = *cyl.position().direction().coordinates();
    let tor_loc = *tor.position().location().coordinates();
    let tor_dir = *tor.position().direction().coordinates();
    if (cyl_dir[0] * tor_dir[0] + cyl_dir[1] * tor_dir[1] + cyl_dir[2] * tor_dir[2]).abs() < 0.999 {
        return Ok(None);
    }
    let new_radius = cyl.radius() + signed_offset;
    let new_fillet = tor.minor_radius() + signed_offset;
    if new_radius <= 1.0e-5 || new_fillet <= 1.0e-5 || new_fillet >= new_radius - 1.0e-5 {
        return Ok(None);
    }
    // Compute axial extent of the source body along `cyl_dir`.
    let mut bbox = BoundingBox3::default();
    BRepBndLib::add_optimal(body_shape, &mut bbox, false, false)
        .map_err(|cause| cause.to_string())?;
    let [xmin, ymin, zmin, xmax, ymax, zmax] = bbox.get().map_err(|cause| cause.to_string())?;
    let mut t_min = f64::INFINITY;
    let mut t_max = f64::NEG_INFINITY;
    for &x in &[xmin, xmax] {
        for &y in &[ymin, ymax] {
            for &z in &[zmin, zmax] {
                let t = (x - cyl_loc[0]) * cyl_dir[0]
                    + (y - cyl_loc[1]) * cyl_dir[1]
                    + (z - cyl_loc[2]) * cyl_dir[2];
                t_min = t_min.min(t);
                t_max = t_max.max(t);
            }
        }
    }
    let tor_t = (tor_loc[0] - cyl_loc[0]) * cyl_dir[0]
        + (tor_loc[1] - cyl_loc[1]) * cyl_dir[1]
        + (tor_loc[2] - cyl_loc[2]) * cyl_dir[2];
    let fillet_at_top = tor_t >= 0.5 * (t_min + t_max);
    let (base_t, height) = if fillet_at_top {
        (t_min, (t_max - t_min) + signed_offset)
    } else {
        (t_min - signed_offset, (t_max - t_min) + signed_offset)
    };
    if height <= new_fillet + 1.0e-5 {
        return Ok(None);
    }
    let origin = Point3::new(
        cyl_loc[0] + cyl_dir[0] * base_t,
        cyl_loc[1] + cyl_dir[1] * base_t,
        cyl_loc[2] + cyl_dir[2] * base_t,
    );
    let dir = Direction3::new(cyl_dir[0], cyl_dir[1], cyl_dir[2]).map_err(|e| e.to_string())?;
    let axis = Axis2::new(origin, dir).map_err(|e| e.to_string())?;
    let mut cyl_maker = BRepPrimAPIMakeCylinder::with_axes(axis, new_radius, height);
    let cyl_shape = cyl_maker.shape().map_err(|e| e.to_string())?.clone();
    let target_t = if fillet_at_top {
        base_t + height
    } else {
        base_t
    };
    let edges = collect_shapes(&cyl_shape, ShapeEnum::Edge)?;
    let fillet_edge = edges.into_iter().find(|edge| {
        let Ok(native) = topods::edge(edge) else {
            return false;
        };
        let Ok(curve) = BRepAdaptorCurve::with_edge(&native) else {
            return false;
        };
        let Ok(first) = curve.first_parameter() else {
            return false;
        };
        let Ok(last) = curve.last_parameter() else {
            return false;
        };
        let Ok(pt) = curve.value(0.5 * (first + last)) else {
            return false;
        };
        let t = (pt.x() - cyl_loc[0]) * cyl_dir[0]
            + (pt.y() - cyl_loc[1]) * cyl_dir[1]
            + (pt.z() - cyl_loc[2]) * cyl_dir[2];
        (t - target_t).abs() < 1.0e-4
    });
    let Some(edge_shape) = fillet_edge else {
        return Ok(None);
    };
    let native_edge = topods::edge(&edge_shape).map_err(|e| e.to_string())?;
    let mut fillet = BRepFilletAPIMakeFillet::new(&cyl_shape, FilletShape::Rational)
        .map_err(|e| e.to_string())?;
    fillet
        .add_with_radius(new_fillet, &native_edge)
        .map_err(|e| e.to_string())?;
    let result = fillet.shape().map_err(|e| e.to_string())?.clone();
    Ok(Some(result))
}

pub(super) fn ensure_positive_solid(shape: Shape, label: &str) -> Result<Shape, String> {
    let solids = collect_shapes(&shape, ShapeEnum::Solid)?;
    let Some(solid) = solids.into_iter().next() else {
        return Err(format!("{label}: expected a closed solid result"));
    };
    let props = crate::checked_shape::analytic_properties(&solid, label)
        .map_err(|cause| cause.to_string())?;
    let solid = if props.volume < 0.0 {
        let mut rebuilt = Solid::default();
        let builder = Builder;
        builder.make_solid(&mut rebuilt);
        let mut rebuilt_shape = rebuilt.into_shape();
        for mut shell in collect_shapes(&solid, ShapeEnum::Shell)? {
            shell.reverse();
            builder
                .add(&mut rebuilt_shape, &shell)
                .map_err(|cause| cause.to_string())?;
        }
        rebuilt_shape
    } else {
        solid
    };
    crate::checked_shape::require_geometry(&solid, label).map_err(|cause| cause.to_string())?;
    Ok(solid)
}

pub(super) fn translate_shape(shape: &Shape, delta: [f64; 3]) -> Result<Shape, String> {
    let trsf = compose(&[RigidMotion::Translation(delta)])?;
    let mut moved = BRepBuilderAPITransform::with_shape(shape, trsf, true, false)
        .map_err(|cause| cause.to_string())?;
    Ok(moved.shape().map_err(|cause| cause.to_string())?.clone())
}
