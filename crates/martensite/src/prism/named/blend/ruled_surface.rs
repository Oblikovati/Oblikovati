// SPDX-License-Identifier: GPL-2.0-only
//! Ruled surface B-rep construction, boundary edge chaining, and preview direction arrows (#3918).
use super::extrusion_boolean::BooleanFailure;
use super::thicken_offset_chain::{collect_shapes, planar_outward_normal};
use super::*;
use martensite_tkbrep::{
    brep_adaptor::{BRepAdaptorCurve, BRepAdaptorSurface},
    topods::{self, Builder, Shape, Shell},
};
use martensite_tkg3d::{
    adaptor3d::{Adaptor3dCurve, Adaptor3dSurface},
    top_abs::{Orientation, ShapeEnum},
};
use martensite_tkmath::{
    bnd::BoundingBox3,
    geom_abs::SurfaceType,
    gp::{Point3, Vector3},
};
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakePrism;
use martensite_tktopalgo::{brep_bnd_lib::BRepBndLib, brep_builder_api::BRepBuilderAPIMakeShape};
use oblikovati_model::feature::{
    FeatureId,
    ruled_surface::{ResolvedRuledSurface, RuledSurfaceType},
};
use std::collections::{BTreeSet, VecDeque};

/// World-space direction arrow for live Ruled Surface preview feedback.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct RuledSurfaceArrow {
    pub origin: [f64; 3],
    pub tip: [f64; 3],
    pub direction: [f64; 3],
}

impl NamedSolid {
    /// Chain vertex-connected boundary edges starting from `seed_edge` in deterministic order.
    /// Example: `body.boundary_edge_chain(&history, document, &edge_key)?`.
    pub fn boundary_edge_chain(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        seed_edge: &[u8],
    ) -> Result<Vec<Vec<u8>>, String> {
        let mut edge_keys: Vec<Vec<u8>> = self
            .subentity_reference_keys(history, document)?
            .into_iter()
            .filter(|key| {
                oblikovati_model::identity::ReferenceEnvelope::decode(key)
                    .is_ok_and(|env| env.kind == 2)
            })
            .collect();
        edge_keys.sort();
        let Some(seed_idx) = edge_keys.iter().position(|k| k.as_slice() == seed_edge) else {
            return Err("ruled surface edge chain: expected a live named edge on the body".into());
        };
        let edges: Vec<Shape> = edge_keys
            .iter()
            .map(|k| self.native_edge(history, document, k).cloned())
            .collect::<Result<_, _>>()?;
        let body_shape = self.native_shape(history, document)?;
        let faces = collect_shapes(body_shape, ShapeEnum::Face)?;
        let face_edges: Vec<Vec<Shape>> = faces
            .iter()
            .map(|face| collect_shapes(face, ShapeEnum::Edge))
            .collect::<Result<_, _>>()?;
        let incident_face_count = |edge: &Shape| -> usize {
            face_edges
                .iter()
                .filter(|fe| fe.iter().any(|candidate| candidate.is_same(edge)))
                .count()
        };
        let seed_is_boundary = incident_face_count(&edges[seed_idx]) <= 1;
        let endpoints: Vec<Vec<[f64; 3]>> = edges
            .iter()
            .map(edge_endpoint_positions)
            .collect::<Result<_, _>>()?;

        let mut visited = BTreeSet::new();
        let mut queue = VecDeque::new();
        let mut ordered = Vec::new();
        visited.insert(seed_idx);
        queue.push_back(seed_idx);
        ordered.push(edge_keys[seed_idx].clone());

        while let Some(current) = queue.pop_front() {
            for other in 0..edges.len() {
                if visited.contains(&other) {
                    continue;
                }
                if seed_is_boundary && incident_face_count(&edges[other]) > 1 {
                    continue;
                }
                if shares_endpoint(&endpoints[current], &endpoints[other]) {
                    visited.insert(other);
                    queue.push_back(other);
                    ordered.push(edge_keys[other].clone());
                }
            }
        }
        Ok(ordered)
    }

    /// Compute the 3D preview direction arrow on the first selected edge.
    /// Example: `body.ruled_surface_preview_arrow(&history, document, &resolved)?`.
    pub fn ruled_surface_preview_arrow(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        input: &ResolvedRuledSurface,
    ) -> Result<Option<RuledSurfaceArrow>, String> {
        let Some(first_key) = input.edges.first() else {
            return Ok(None);
        };
        let body_shape = self.native_shape(history, document)?;
        let faces = collect_shapes(body_shape, ShapeEnum::Face)?;
        let edge = self.native_edge(history, document, first_key)?;
        let sweep_vector = self.resolve_sweep_vector(history, document, input)?;
        let (mid, tangent) = edge_midpoint_and_tangent(edge)?;
        let adjacent = adjacent_face_for_edge(&faces, edge)?;
        let Some(direction) = resolve_ruled_direction(
            input.ruled_type,
            adjacent,
            mid,
            tangent,
            sweep_vector,
            input.flip,
            input.angle_deg,
        )?
        else {
            return Ok(None);
        };
        let arrow_len = input.distance.clamp(0.5, 2.5);
        let origin = [mid.x(), mid.y(), mid.z()];
        let tip = [
            origin[0] + direction[0] * arrow_len,
            origin[1] + direction[1] * arrow_len,
            origin[2] + direction[2] * arrow_len,
        ];
        Ok(Some(RuledSurfaceArrow {
            origin,
            tip,
            direction,
        }))
    }

    pub(in crate::prism::named) fn evaluate_ruled_surface_result(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        input: &ResolvedRuledSurface,
    ) -> Result<NamedSolid, BooleanFailure> {
        if input.edges.is_empty() {
            return Err(BooleanFailure::Refused(
                "ruled surface: expected at least one selected edge".into(),
            ));
        }
        if !input.distance.is_finite() || input.distance <= 0.0 {
            return Err(BooleanFailure::Refused(format!(
                "ruled surface distance {}: expected a finite positive distance",
                input.distance
            )));
        }
        let resolution = self.body_resolution_for_ruled();
        let body_shape = self
            .native_shape(history, document)
            .map_err(BooleanFailure::Refused)?;
        let faces = collect_shapes(body_shape, ShapeEnum::Face).map_err(BooleanFailure::Refused)?;
        let sweep_vector = self
            .resolve_sweep_vector(history, document, input)
            .map_err(BooleanFailure::Refused)?;
        let mut copier = crate::checked_shape::copy_shape(body_shape, "ruled surface input")
            .map_err(|cause| BooleanFailure::Refused(cause.to_string()))?;

        let mut shell = Shell::default();
        let builder = Builder;
        builder.make_shell(&mut shell);
        let mut shell_shape = shell.into_shape();
        let mut face_count = 0usize;

        for key in &input.edges {
            let source_edge = self
                .native_edge(history, document, key)
                .map_err(BooleanFailure::Refused)?;
            let (mid, tangent) =
                edge_midpoint_and_tangent(source_edge).map_err(BooleanFailure::Refused)?;
            let adjacent =
                adjacent_face_for_edge(&faces, source_edge).map_err(BooleanFailure::Refused)?;
            let Some(direction) = resolve_ruled_direction(
                input.ruled_type,
                adjacent,
                mid,
                tangent,
                sweep_vector,
                input.flip,
                input.angle_deg,
            )
            .map_err(BooleanFailure::Refused)?
            else {
                return Err(BooleanFailure::Refused(
                    "ruled surface: unable to resolve extrusion direction".into(),
                ));
            };
            let copies = copier
                .modified(source_edge)
                .map_err(|cause| BooleanFailure::Refused(cause.to_string()))?;
            let Some(edge_copy) = copies.first() else {
                return Err(BooleanFailure::Refused(format!(
                    "ruled surface edge {key:?}: expected one detached edge copy"
                )));
            };
            let disp = Vector3::new(
                direction[0] * input.distance,
                direction[1] * input.distance,
                direction[2] * input.distance,
            );
            let mut prism =
                BRepPrimAPIMakePrism::new(edge_copy, disp, true, true).map_err(|cause| {
                    BooleanFailure::Refused(format!("ruled surface prism: {cause}"))
                })?;
            let prism_shape = prism
                .shape()
                .map_err(|cause| BooleanFailure::Refused(cause.to_string()))?
                .clone();
            for result_face in
                collect_shapes(&prism_shape, ShapeEnum::Face).map_err(BooleanFailure::Refused)?
            {
                builder
                    .add(&mut shell_shape, &result_face)
                    .map_err(|cause| BooleanFailure::Refused(cause.to_string()))?;
                face_count += 1;
            }
        }
        if face_count == 0 {
            return Err(BooleanFailure::Refused(
                "ruled surface: expected at least one generated surface face".into(),
            ));
        }
        let topology = BlendNamedTopology::from_ruled_surface_shell(
            history,
            document,
            feature,
            shell_shape,
            resolution,
        )
        .map_err(BooleanFailure::Refused)?;
        Ok(NamedSolid::Derived(Box::new(topology)))
    }

    fn body_resolution_for_ruled(&self) -> Option<PrismResolution> {
        match self {
            Self::Prism(body) => body.certification.as_ref().map(|c| c.resolution),
            Self::Derived(body) => body.resolution(),
        }
    }

    fn resolve_sweep_vector(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        input: &ResolvedRuledSurface,
    ) -> Result<Option<[f64; 3]>, String> {
        if let Some(dir) = input.vector_direction {
            return Ok(Some(normalize(dir).ok_or_else(|| {
                format!("ruled surface vector {dir:?}: expected a nonzero direction")
            })?));
        }
        if let Some(edge_key) = &input.vector_edge {
            let edge = self.native_edge(history, document, edge_key)?;
            let (_, tangent) = edge_midpoint_and_tangent(edge)?;
            return Ok(Some(tangent));
        }
        Ok(None)
    }
}

fn edge_endpoint_positions(edge: &Shape) -> Result<Vec<[f64; 3]>, String> {
    let native = topods::edge(edge).map_err(|cause| cause.to_string())?;
    let curve = BRepAdaptorCurve::with_edge(&native).map_err(|cause| cause.to_string())?;
    let first = curve.first_parameter().map_err(|cause| cause.to_string())?;
    let last = curve.last_parameter().map_err(|cause| cause.to_string())?;
    let p0 = curve.value(first).map_err(|cause| cause.to_string())?;
    let p1 = curve.value(last).map_err(|cause| cause.to_string())?;
    Ok(vec![[p0.x(), p0.y(), p0.z()], [p1.x(), p1.y(), p1.z()]])
}

fn shares_endpoint(a: &[[f64; 3]], b: &[[f64; 3]]) -> bool {
    const TOL_SQ: f64 = 1.0e-8;
    a.iter().any(|pa| {
        b.iter().any(|pb| {
            let dx = pa[0] - pb[0];
            let dy = pa[1] - pb[1];
            let dz = pa[2] - pb[2];
            dx * dx + dy * dy + dz * dz <= TOL_SQ
        })
    })
}

fn edge_midpoint_and_tangent(edge: &Shape) -> Result<(Point3, [f64; 3]), String> {
    let native = topods::edge(edge).map_err(|cause| cause.to_string())?;
    let curve = BRepAdaptorCurve::with_edge(&native).map_err(|cause| cause.to_string())?;
    let first = curve.first_parameter().map_err(|cause| cause.to_string())?;
    let last = curve.last_parameter().map_err(|cause| cause.to_string())?;
    let mid_param = 0.5 * (first + last);
    let mid = curve.value(mid_param).map_err(|cause| cause.to_string())?;
    let dt = ((last - first) * 1.0e-4).max(1.0e-7);
    let p_a = curve
        .value((mid_param - dt).max(first))
        .map_err(|cause| cause.to_string())?;
    let p_b = curve
        .value((mid_param + dt).min(last))
        .map_err(|cause| cause.to_string())?;
    let sign = if edge.orientation() == Orientation::Reversed {
        -1.0
    } else {
        1.0
    };
    let tangent = normalize([
        (p_b.x() - p_a.x()) * sign,
        (p_b.y() - p_a.y()) * sign,
        (p_b.z() - p_a.z()) * sign,
    ])
    .ok_or_else(|| "ruled surface edge: expected a non-degenerate curve tangent".to_string())?;
    Ok((mid, tangent))
}

fn adjacent_face_for_edge<'a>(
    faces: &'a [Shape],
    edge: &Shape,
) -> Result<Option<&'a Shape>, String> {
    for face in faces {
        let face_edges = collect_shapes(face, ShapeEnum::Edge)?;
        if face_edges.iter().any(|candidate| candidate.is_same(edge)) {
            return Ok(Some(face));
        }
    }
    Ok(None)
}

fn resolve_ruled_direction(
    ruled_type: RuledSurfaceType,
    adjacent_face: Option<&Shape>,
    mid: Point3,
    tangent: [f64; 3],
    sweep_vector: Option<[f64; 3]>,
    flip: bool,
    angle_deg: f64,
) -> Result<Option<[f64; 3]>, String> {
    let base = match ruled_type {
        RuledSurfaceType::Normal => {
            let Some(face) = adjacent_face else {
                return Err("ruled surface Normal: expected an edge with an adjacent face".into());
            };
            outward_face_normal(face, mid)?
        }
        RuledSurfaceType::Tangent => {
            let Some(face) = adjacent_face else {
                return Err("ruled surface Tangent: expected an edge with an adjacent face".into());
            };
            let normal = outward_face_normal(face, mid)?;
            let mut side = normalize(cross(normal, tangent)).ok_or_else(|| {
                "ruled surface Tangent: edge tangent is collinear with face normal".to_string()
            })?;
            let center = shape_bounds_center(face)?;
            let away = [
                mid.x() - center[0],
                mid.y() - center[1],
                mid.z() - center[2],
            ];
            if dot(side, away) < 0.0 {
                side = [-side[0], -side[1], -side[2]];
            }
            side
        }
        RuledSurfaceType::Sweep => {
            let Some(vec) = sweep_vector else {
                return Ok(None);
            };
            let perp = cross(vec, tangent);
            let perp_len = (perp[0] * perp[0] + perp[1] * perp[1] + perp[2] * perp[2]).sqrt();
            if perp_len <= 1.0e-6 {
                return Err(
                    "ruled surface Sweep: vector direction is collinear with the selected edge"
                        .into(),
                );
            }
            vec
        }
    };
    let oriented = if flip {
        [-base[0], -base[1], -base[2]]
    } else {
        base
    };
    let rotated = if angle_deg.abs() > 1.0e-12 {
        rotate_about_axis(oriented, tangent, angle_deg.to_radians())
    } else {
        oriented
    };
    Ok(normalize(rotated))
}

fn outward_face_normal(face: &Shape, point: Point3) -> Result<[f64; 3], String> {
    if let Some(normal) = planar_outward_normal(face)? {
        return Ok(normal);
    }
    let native = topods::face(face).map_err(|cause| cause.to_string())?;
    let adaptor =
        BRepAdaptorSurface::with_face(&native, false).map_err(|cause| cause.to_string())?;
    let raw = match adaptor.get_type().map_err(|cause| cause.to_string())? {
        SurfaceType::Cylinder => {
            let cyl = adaptor.cylinder().map_err(|cause| cause.to_string())?;
            let loc = *cyl.position().location().coordinates();
            let dir = *cyl.position().direction().coordinates();
            let rel = [point.x() - loc[0], point.y() - loc[1], point.z() - loc[2]];
            let along = dot(rel, dir);
            normalize([
                rel[0] - along * dir[0],
                rel[1] - along * dir[1],
                rel[2] - along * dir[2],
            ])
            .ok_or_else(|| "ruled surface: degenerate cylindrical face normal".to_string())?
        }
        other => {
            return Err(format!(
                "ruled surface face type {other:?}: expected a planar or cylindrical face"
            ));
        }
    };
    let sign = if face.orientation() == Orientation::Reversed {
        -1.0
    } else {
        1.0
    };
    Ok([raw[0] * sign, raw[1] * sign, raw[2] * sign])
}

fn shape_bounds_center(shape: &Shape) -> Result<[f64; 3], String> {
    let mut bbox = BoundingBox3::default();
    BRepBndLib::add_optimal(shape, &mut bbox, false, false).map_err(|cause| cause.to_string())?;
    let [xmin, ymin, zmin, xmax, ymax, zmax] = bbox.get().map_err(|cause| cause.to_string())?;
    Ok([
        0.5 * (xmin + xmax),
        0.5 * (ymin + ymax),
        0.5 * (zmin + zmax),
    ])
}

fn rotate_about_axis(v: [f64; 3], axis: [f64; 3], radians: f64) -> [f64; 3] {
    let (s, c) = radians.sin_cos();
    let k_cross_v = cross(axis, v);
    let k_dot_v = dot(axis, v);
    [
        v[0] * c + k_cross_v[0] * s + axis[0] * k_dot_v * (1.0 - c),
        v[1] * c + k_cross_v[1] * s + axis[1] * k_dot_v * (1.0 - c),
        v[2] * c + k_cross_v[2] * s + axis[2] * k_dot_v * (1.0 - c),
    ]
}

fn dot(a: [f64; 3], b: [f64; 3]) -> f64 {
    a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
}

fn cross(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [
        a[1] * b[2] - a[2] * b[1],
        a[2] * b[0] - a[0] * b[2],
        a[0] * b[1] - a[1] * b[0],
    ]
}

fn normalize(v: [f64; 3]) -> Option<[f64; 3]> {
    let len = (v[0] * v[0] + v[1] * v[1] + v[2] * v[2]).sqrt();
    (len.is_finite() && len > 1.0e-12).then(|| [v[0] / len, v[1] / len, v[2] / len])
}
