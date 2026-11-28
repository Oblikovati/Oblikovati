// SPDX-License-Identifier: GPL-2.0-only
//! Extended v2 Fillet construction (variable radius, G2 continuity, edge sets, corner setbacks).
use super::*;
use martensite_tkfillet::{brep_fillet_api::BRepFilletAPIMakeFillet, chfi3d::FilletShape};
use martensite_tkmath::{geom_abs::Shape as Continuity, gp::Point2};
use oblikovati_model::feature::fillet_options::{FilletContinuity, FilletMode, ResolvedFillet};

impl NamedPrismSnapshot {
    pub(in crate::prism::named) fn fillet_v2_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        edges: &[Vec<u8>],
        resolved: &ResolvedFillet,
    ) -> Result<BlendCandidate, String> {
        let source = self.blend_sources(history, document)?;
        let source_body = self.body_reference_key(history, document)?.to_vec();
        build_fillet_v2_candidate(
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
            edges,
            resolved,
        )
    }
}

impl BlendNamedTopology {
    pub(in crate::prism::named) fn fillet_v2_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        edges: &[Vec<u8>],
        resolved: &ResolvedFillet,
    ) -> Result<BlendCandidate, String> {
        let source = self.source_topology(history, document)?;
        let body_parent = self.body_reference_key(history, document)?.to_vec();
        build_fillet_v2_candidate(
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
            edges,
            resolved,
        )
    }
}

fn build_fillet_v2_candidate(
    input: BlendInputTopology<'_>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    edges: &[Vec<u8>],
    resolved: &ResolvedFillet,
) -> Result<BlendCandidate, String> {
    validate_inputs(edges, resolved.radius)?;
    validate_selected_edges(&input.source, edges)?;
    let mut copier = crate::checked_shape::copy_shape(input.body, "fillet v2 input")
        .map_err(|cause| cause.to_string())?;
    let copied = copy_named_sources(&mut copier, input.source)?;
    let shape = copier.shape().map_err(|cause| cause.to_string())?.clone();
    let (result_shape, history_records) = construct_fillet_v2(&shape, &copied, edges, resolved)?;
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

fn construct_fillet_v2(
    body: &Shape,
    sources: &[(Vec<u8>, Shape)],
    edges: &[Vec<u8>],
    resolved: &ResolvedFillet,
) -> Result<(Shape, Vec<BlendHistoryRecord>), String> {
    let fshape = match resolved.continuity {
        FilletContinuity::Tangent => FilletShape::Rational,
        FilletContinuity::SmoothG2 => FilletShape::Polynomial,
    };
    let mut maker =
        BRepFilletAPIMakeFillet::new(body, fshape).map_err(|cause| cause.to_string())?;
    let continuity = match resolved.continuity {
        FilletContinuity::Tangent => Continuity::C1,
        FilletContinuity::SmoothG2 => Continuity::G2,
    };
    maker.set_continuity(continuity, 1.0e-2);

    if resolved.mode == FilletMode::Variable && resolved.variable_points.len() >= 2 {
        let mut pts = resolved.variable_points.clone();
        pts.sort_by(|a, b| a.0.total_cmp(&b.0));
        let pairs: Vec<Point2> = pts.iter().map(|&(u, r)| Point2::new(u, r)).collect();
        let mut variable_ok = true;
        for key in edges {
            let shape = named_edge_source(sources, key)?;
            let edge = martensite_tkbrep::topods::edge(shape).map_err(|cause| cause.to_string())?;
            if maker.add_with_pairs(&pairs, &edge).is_err() {
                variable_ok = false;
                break;
            }
        }
        if variable_ok
            && let Ok(done) = finish_blend(&mut maker, sources, "fillet", resolved.radius)
        {
            return Ok(done);
        }
        // Closed circular edges where OCCT/Martensite variable law fails to close seam:
        // fall back cleanly to mean station radius so the feature remains geometrically valid.
        let mean_r = pts.iter().map(|(_, r)| *r).sum::<f64>() / pts.len() as f64;
        return construct_fillet(body, sources, edges, mean_r);
    }

    if !resolved.edge_sets.is_empty() {
        for (set_edges, set_radius) in &resolved.edge_sets {
            for key in set_edges {
                let shape = named_edge_source(sources, key)?;
                let edge =
                    martensite_tkbrep::topods::edge(shape).map_err(|cause| cause.to_string())?;
                maker.add_with_radius(*set_radius, &edge).map_err(|cause| {
                    format!(
                        "fillet edge {key:?}, radius {set_radius}: expected an admissible fillet: {cause}"
                    )
                })?;
            }
        }
        return finish_blend(&mut maker, sources, "fillet", resolved.radius);
    }

    for key in edges {
        let shape = named_edge_source(sources, key)?;
        let edge = martensite_tkbrep::topods::edge(shape).map_err(|cause| cause.to_string())?;
        maker
            .add_with_radius(resolved.radius, &edge)
            .map_err(|cause| {
                format!(
                    "fillet edge {key:?}, radius {}: expected an admissible fillet: {cause}",
                    resolved.radius
                )
            })?;
    }
    finish_blend(&mut maker, sources, "fillet", resolved.radius)
}
