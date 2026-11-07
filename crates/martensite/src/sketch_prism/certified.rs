// SPDX-License-Identifier: GPL-2.0-only
//! Convert current document-owned sketch identities to the released profile operation.
use super::*;
use martensite_tkbo::profile_extrusion::{
    ProfileCurve, ProfileCurveId, ProfilePlane, ProfileVertex, ProfileVertexId, Resolution,
    SketchProfile,
};
use oblikovati_model::feature::sketch_prism::ResolvedSketchPrism;
use std::collections::BTreeMap;

/// Evaluate a current, fully constrained saved prism with an explicit modelling resolution.
/// Example: `retain_certified_sketch_prism(&document, &definition, resolution)?`.
pub fn retain_certified_sketch_prism(
    document: &DocumentSnapshot,
    definition: &SketchPrismDefinition,
    resolution: Resolution,
) -> Result<crate::CertifiedPrismSnapshot, EvaluationFailure> {
    let resolved = definition.resolve(document).map_err(|cause| {
        Phase::Input
            .refused(Kind::InvalidInput, Entity::Body, cause.to_string())
            .with_sketch_identity(definition.sketch())
    })?;
    let sketch = resolved.sketch.sketch.id();
    evaluate(resolved, resolution)
        .map_err(|cause| cause.for_sketch(sketch, format!("document sketch {sketch:?}")))
}

pub(crate) fn evaluate(
    input: ResolvedSketchPrism<'_>,
    resolution: Resolution,
) -> Result<crate::CertifiedPrismSnapshot, EvaluationFailure> {
    let plane = input.sketch.plane;
    let (profile, sources) =
        selected_profile(input.sketch.sketch.geometry(), input.selection, plane)?;
    let vector = crate::prism::placement::model_displacement(plane, input.displacement)?;
    let mut result = crate::retain_certified_profile_prism(
        &profile,
        input.selection,
        [vector.x(), vector.y(), vector.z()],
        resolution,
    )?;
    result.vertex_sources = sources;
    Ok(result)
}

fn selected_profile(
    sketch: &oblikovati_model::sketch::Sketch2D,
    selection: &SketchPrismSelection,
    plane: oblikovati_model::sketch::SketchPlane,
) -> Result<(SketchProfile, BTreeMap<u64, Vec<u64>>), EvaluationFailure> {
    let coincidence = super::incidence::CoincidentPoints::new(sketch);
    let mut vertices = BTreeMap::new();
    let mut curves = Vec::new();
    for line in std::iter::once(&selection.outer)
        .chain(&selection.holes)
        .flatten()
    {
        curves.push(profile_line(sketch, *line, &coincidence, &mut vertices)?);
    }
    let recipe = plane.recipe();
    let sources = vertices
        .keys()
        .map(|id| {
            (
                id.0,
                coincidence.sources(oblikovati_model::sketch::SketchEntityId(id.0)),
            )
        })
        .collect();
    Ok((
        SketchProfile {
            plane: ProfilePlane {
                origin: recipe.origin,
                x_direction: recipe.x_axis,
                normal: plane.normal(),
            },
            vertices: vertices.into_values().collect(),
            curves,
        },
        sources,
    ))
}

fn profile_line(
    sketch: &oblikovati_model::sketch::Sketch2D,
    directed: SketchDirectedLine,
    coincidence: &super::incidence::CoincidentPoints,
    vertices: &mut BTreeMap<ProfileVertexId, ProfileVertex>,
) -> Result<ProfileCurve, EvaluationFailure> {
    let line = boundary::require_boundary_line(sketch, directed.line)?;
    let endpoints = line
        .endpoints
        .map(|id| coincidence.vertex(id))
        .into_iter()
        .collect::<Result<Vec<_>, _>>()?;
    for point in &endpoints {
        vertices.insert(
            ProfileVertexId(point.id.0),
            ProfileVertex {
                id: ProfileVertexId(point.id.0),
                point: point.position,
            },
        );
    }
    Ok(ProfileCurve::Line {
        id: ProfileCurveId(line.id.0),
        start: ProfileVertexId(endpoints[0].id.0),
        end: ProfileVertexId(endpoints[1].id.0),
    })
}
