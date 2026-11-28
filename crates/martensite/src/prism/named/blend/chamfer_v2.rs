// SPDX-License-Identifier: GPL-2.0-only
//! Extended v2 Chamfer construction (distance-and-angle, two-distances, partial-edge cut wedges, #4046).
use super::*;
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tkmath::gp::{Point3, Vector3};
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakePrism;
use martensite_tktopalgo::brep_builder_api::{BRepBuilderAPIMakeFace, BRepBuilderAPIMakePolygon};
use oblikovati_model::{
    feature::chamfer_options::{ChamferMode, ChamferPartialSpec, ResolvedChamfer},
    identity::DerivedReferenceName,
};

impl NamedPrismSnapshot {
    pub(in crate::prism::named) fn chamfer_v2_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        edges: &[Vec<u8>],
        resolved: &ResolvedChamfer,
    ) -> Result<BlendCandidate, String> {
        let source = self.blend_sources(history, document)?;
        let source_body = self.body_reference_key(history, document)?.to_vec();
        build_chamfer_v2_candidate(
            BlendInputTopology {
                body: &self.snapshot.shape,
                source,
                source_bodies: vec![source_body.clone()],
                body_parent: source_body.clone(),
                primary_body: source_body,
                inherited_deleted: Vec::new(),
                inherited_splits: Default::default(),
                resolution: self.certification.as_ref().map(|value| value.resolution),
            },
            history,
            document,
            feature,
            edges,
            resolved,
        )
    }
}

impl BlendNamedTopology {
    pub(in crate::prism::named) fn chamfer_v2_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        edges: &[Vec<u8>],
        resolved: &ResolvedChamfer,
    ) -> Result<BlendCandidate, String> {
        let source = self.source_topology(history, document)?;
        let body_parent = self.body_reference_key(history, document)?.to_vec();
        build_chamfer_v2_candidate(
            BlendInputTopology {
                body: &source.body,
                source: source.entities,
                source_bodies: source.body_keys,
                body_parent: body_parent.clone(),
                primary_body: body_parent,
                inherited_deleted: self.deleted_keys().to_vec(),
                inherited_splits: self.split_keys().clone(),
                resolution: self.resolution(),
            },
            history,
            document,
            feature,
            edges,
            resolved,
        )
    }
}

impl NamedSolid {
    /// Measure a current named edge's endpoints and length in centimeters.
    pub fn edge_span(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        edge_key: &[u8],
    ) -> Result<([f64; 3], [f64; 3], f64), String> {
        let segs = self.edge_display_segments(history, document, edge_key, 0.02)?;
        let start = segs
            .first()
            .ok_or("chamfer edge span: expected at least one edge segment")?[0];
        let end = segs
            .last()
            .ok_or("chamfer edge span: expected at least one edge segment")?[1];
        let mut length = 0.0;
        for [a, b] in &segs {
            length += dist3(*a, *b);
        }
        Ok((start, end, length))
    }
}

fn build_chamfer_v2_candidate(
    input: BlendInputTopology<'_>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    feature: FeatureId,
    edges: &[Vec<u8>],
    resolved: &ResolvedChamfer,
) -> Result<BlendCandidate, String> {
    validate_inputs(edges, resolved.distance)?;
    validate_selected_edges(&input.source, edges)?;
    if let Some(partial) = resolved.partial {
        return build_partial_chamfer_candidate(
            input, history, document, feature, edges, resolved, partial,
        );
    }
    let mut copier = crate::checked_shape::copy_shape(input.body, "chamfer v2 input")
        .map_err(|cause| cause.to_string())?;
    let copied = copy_named_sources(&mut copier, input.source)?;
    let shape = copier.shape().map_err(|cause| cause.to_string())?.clone();
    let (result_shape, history_records) = construct_chamfer_v2(&shape, &copied, edges, resolved)?;
    Ok(BlendCandidate {
        resolution: input.resolution,
        achieved_tolerance: None,
        shape: result_shape,
        history: history_records,
        source_bodies: input.source_bodies,
        body_parent: input.body_parent,
        primary_body: input.primary_body,
        inherited_deleted: input.inherited_deleted,
        inherited_splits: input.inherited_splits,
        document,
        scope: history.evaluation_scope(),
    })
}

fn construct_chamfer_v2(
    body: &Shape,
    sources: &[(Vec<u8>, Shape)],
    edges: &[Vec<u8>],
    resolved: &ResolvedChamfer,
) -> Result<(Shape, Vec<BlendHistoryRecord>), String> {
    let mut maker = BRepFilletAPIMakeChamfer::new(body).map_err(|cause| cause.to_string())?;
    let explicit_face = resolved
        .reference_face
        .as_deref()
        .and_then(|key| named_edge_source(sources, key).ok())
        .and_then(|shape| martensite_tkbrep::topods::face(shape).ok());

    for key in edges {
        let shape = named_edge_source(sources, key)?;
        let edge = martensite_tkbrep::topods::edge(shape).map_err(|cause| cause.to_string())?;
        match resolved.mode {
            ChamferMode::EqualDistance => {
                maker
                    .add_with_dist(resolved.distance, &edge)
                    .map_err(|cause| {
                        format!(
                            "chamfer edge {key:?}, distance {}: {cause}",
                            resolved.distance
                        )
                    })?;
            }
            ChamferMode::DistanceAndAngle => {
                let face =
                    resolve_incident_face(body, &edge, explicit_face.as_ref(), resolved.reversed)?;
                let d2 = resolved.distance * resolved.angle_rad.tan();
                if maker
                    .add_dist_angle(resolved.distance, resolved.angle_rad, &edge, &face)
                    .is_err()
                {
                    maker
                        .add_with_dists(resolved.distance, d2, &edge, &face)
                        .map_err(|cause| format!("chamfer distance-angle {key:?}: {cause}"))?;
                }
            }
            ChamferMode::TwoDistances => {
                let d2 = resolved
                    .distance2
                    .ok_or("chamfer two-distance: expected Distance2")?;
                let face = resolve_incident_face(body, &edge, explicit_face.as_ref(), false)?;
                maker
                    .add_with_dists(resolved.distance, d2, &edge, &face)
                    .map_err(|cause| format!("chamfer two-distances {key:?}: {cause}"))?;
            }
        }
    }
    finish_blend(&mut maker, sources, "chamfer", resolved.distance)
}

fn build_partial_chamfer_candidate(
    input: BlendInputTopology<'_>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    feature: FeatureId,
    edges: &[Vec<u8>],
    resolved: &ResolvedChamfer,
    partial: ChamferPartialSpec,
) -> Result<BlendCandidate, String> {
    partial.validate()?;
    let d1 = resolved.distance;
    let d2 = match resolved.mode {
        ChamferMode::EqualDistance => d1,
        ChamferMode::DistanceAndAngle => d1 * resolved.angle_rad.tan(),
        ChamferMode::TwoDistances => resolved.distance2.unwrap_or(d1),
    };
    let mut tools = Vec::new();
    for (idx, key) in edges.iter().enumerate() {
        let shape = named_edge_source(&input.source, key)?;
        let edge = martensite_tkbrep::topods::edge(shape).map_err(|cause| cause.to_string())?;
        let wedge = build_partial_wedge_solid(input.body, &edge, d1, d2, partial)?;
        let faces = super::solid_boolean::entities(&wedge, ShapeEnum::Face)?;
        let mut sources = Vec::with_capacity(faces.len());
        for (fi, f) in faces.into_iter().enumerate() {
            let role = format!("chamfer-partial-{idx}-face-{fi}");
            let name = DerivedReferenceName::new(
                document,
                feature.0,
                1,
                &role,
                std::slice::from_ref(key),
            )?
            .encode()?;
            sources.push((name, f));
        }
        tools.push(super::solid_boolean::BooleanTool {
            shape: wedge,
            sources,
            resolution: input.resolution,
        });
    }
    super::solid_boolean::build_boolean_candidate(
        input,
        history,
        document,
        tools,
        oblikovati_model::feature::sketch_prism::PrismBoolean::Cut,
        "chamfer partial",
    )
    .map_err(|failure| failure.to_string())
}

fn build_partial_wedge_solid(
    body: &Shape,
    edge: &martensite_tkbrep::topods::Edge,
    d1: f64,
    d2: f64,
    partial: ChamferPartialSpec,
) -> Result<Shape, String> {
    let verts = super::solid_boolean::entities(&edge.to_shape(), ShapeEnum::Vertex)?;
    if verts.len() < 2 {
        return Err("chamfer partial: expected a straight edge with two vertices".into());
    }
    let p0 = vertex_xyz(&verts[0])?;
    let p1 = vertex_xyz(&verts[verts.len() - 1])?;
    let len = dist3(p0, p1);
    if len <= 1e-7 {
        return Err("chamfer partial: degenerate edge".into());
    }
    let u = [
        (p1[0] - p0[0]) / len,
        (p1[1] - p0[1]) / len,
        (p1[2] - p0[2]) / len,
    ];
    let mid = [
        (p0[0] + p1[0]) * 0.5,
        (p0[1] + p1[1]) * 0.5,
        (p0[2] + p1[2]) * 0.5,
    ];
    let faces = incident_faces(body, edge)?;
    if faces.len() < 2 {
        return Err("chamfer partial: expected two faces incident to edge".into());
    }
    let t1 = inward_face_tangent(&faces[0], mid, u)?;
    let t2 = inward_face_tangent(&faces[1], mid, u)?;

    let eps = 1.0e-3_f64.min(d1 * 0.1).min(d2 * 0.1);
    let s_start = if partial.to_start_cm <= 1e-6 {
        -eps
    } else {
        partial.to_start_cm
    };
    let s_end_raw = partial.to_start_cm + partial.chamfer_length_cm;
    let s_end = if s_end_raw >= len - 1e-6 {
        len + eps
    } else {
        s_end_raw
    };
    let span = s_end - s_start;
    if span <= 1e-6 {
        return Err("chamfer partial: non-positive extrusion span".into());
    }
    let base = add3(p0, scale3(u, s_start));
    let q0 = add3(base, add3(scale3(t1, -eps), scale3(t2, -eps)));
    let q1 = add3(
        base,
        add3(scale3(t1, d1 + (d1 / d2) * eps), scale3(t2, -eps)),
    );
    let q2 = add3(
        base,
        add3(scale3(t1, -eps), scale3(t2, d2 + (d2 / d1) * eps)),
    );

    let mut poly = BRepBuilderAPIMakePolygon::new();
    poly.add_point(Point3::new(q0[0], q0[1], q0[2]))
        .map_err(|cause| cause.to_string())?;
    poly.add_point(Point3::new(q1[0], q1[1], q1[2]))
        .map_err(|cause| cause.to_string())?;
    poly.add_point(Point3::new(q2[0], q2[1], q2[2]))
        .map_err(|cause| cause.to_string())?;
    poly.close().map_err(|cause| cause.to_string())?;
    let wire = poly.wire().map_err(|cause| cause.to_string())?;
    let face: Shape = BRepBuilderAPIMakeFace::from_wire(&wire, true)
        .map_err(|cause| cause.to_string())?
        .face()
        .map_err(|cause| cause.to_string())?
        .into();
    let vec = Vector3::new(u[0] * span, u[1] * span, u[2] * span);
    let mut prism =
        BRepPrimAPIMakePrism::new(&face, vec, false, true).map_err(|cause| cause.to_string())?;
    Ok(prism.shape().map_err(|cause| cause.to_string())?.clone())
}

fn incident_faces(
    body: &Shape,
    edge: &martensite_tkbrep::topods::Edge,
) -> Result<Vec<martensite_tkbrep::topods::Face>, String> {
    let edge_shape = edge.to_shape();
    let mut result = Vec::new();
    for fshape in super::solid_boolean::entities(body, ShapeEnum::Face)? {
        let contains = super::solid_boolean::entities(&fshape, ShapeEnum::Edge)?
            .iter()
            .any(|e| e.is_same(&edge_shape));
        if contains && let Ok(face) = martensite_tkbrep::topods::face(&fshape) {
            result.push(face);
        }
    }
    Ok(result)
}

fn resolve_incident_face(
    body: &Shape,
    edge: &martensite_tkbrep::topods::Edge,
    explicit: Option<&martensite_tkbrep::topods::Face>,
    reversed: bool,
) -> Result<martensite_tkbrep::topods::Face, String> {
    let faces = incident_faces(body, edge)?;
    if let Some(exp) = explicit {
        let exp_shape = exp.to_shape();
        if faces.iter().any(|f| f.to_shape().is_same(&exp_shape)) {
            if reversed
                && let Some(other) = faces.iter().find(|f| !f.to_shape().is_same(&exp_shape))
            {
                return Ok(other.clone());
            }
            return Ok(exp.clone());
        }
    }
    let idx = usize::from(reversed && faces.len() > 1);
    faces
        .into_iter()
        .nth(idx)
        .ok_or_else(|| "chamfer: expected an incident face for selected edge".to_string())
}

fn inward_face_tangent(
    face: &martensite_tkbrep::topods::Face,
    mid: [f64; 3],
    u: [f64; 3],
) -> Result<[f64; 3], String> {
    let verts = super::solid_boolean::entities(&face.to_shape(), ShapeEnum::Vertex)?;
    if verts.is_empty() {
        return Err("chamfer partial: face has no vertices".into());
    }
    let mut sum = [0.0; 3];
    for v in &verts {
        let p = vertex_xyz(v)?;
        sum[0] += p[0];
        sum[1] += p[1];
        sum[2] += p[2];
    }
    let n = verts.len() as f64;
    let centroid = [sum[0] / n, sum[1] / n, sum[2] / n];
    let v = [
        centroid[0] - mid[0],
        centroid[1] - mid[1],
        centroid[2] - mid[2],
    ];
    let proj = v[0] * u[0] + v[1] * u[1] + v[2] * u[2];
    let perp = [v[0] - proj * u[0], v[1] - proj * u[1], v[2] - proj * u[2]];
    let plen = (perp[0] * perp[0] + perp[1] * perp[1] + perp[2] * perp[2]).sqrt();
    if plen <= 1e-9 {
        return Err("chamfer partial: degenerate face tangent".into());
    }
    Ok([perp[0] / plen, perp[1] / plen, perp[2] / plen])
}

fn vertex_xyz(shape: &Shape) -> Result<[f64; 3], String> {
    let v = martensite_tkbrep::topods::vertex(shape).map_err(|cause| cause.to_string())?;
    let p = martensite_tkbrep::brep::BRepTool::pnt(&v).map_err(|cause| cause.to_string())?;
    Ok([p.x(), p.y(), p.z()])
}

fn dist3(a: [f64; 3], b: [f64; 3]) -> f64 {
    ((a[0] - b[0]).powi(2) + (a[1] - b[1]).powi(2) + (a[2] - b[2]).powi(2)).sqrt()
}

fn add3(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [a[0] + b[0], a[1] + b[1], a[2] + b[2]]
}

fn scale3(a: [f64; 3], s: f64) -> [f64; 3] {
    [a[0] * s, a[1] * s, a[2] * s]
}
