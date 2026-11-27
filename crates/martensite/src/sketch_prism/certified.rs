// SPDX-License-Identifier: GPL-2.0-only
//! Convert current document-owned sketch identities to the released profile operation.
use super::*;
use martensite_tkbo::profile_extrusion::{
    ProfileCurve, ProfileCurveId, ProfileVertex, ProfileVertexId, Resolution,
};
use oblikovati_model::feature::FeatureId;
use oblikovati_model::feature::sketch_prism::ResolvedSketchPrism;
use std::collections::BTreeMap;

pub(crate) fn evaluate_feature(
    input: ResolvedSketchPrism<'_>,
    resolution: crate::PrismResolutionPolicy,
    feature: FeatureId,
) -> Result<(crate::CertifiedPrismSnapshot, crate::PrismResolution), EvaluationFailure> {
    let sketch = input.sketch.sketch.id();
    let resolution = resolution
        .for_sketch(&input.sketch)
        .map_err(|error| error.for_feature(feature))?;
    let result = evaluate(input, resolution.kernel())
        .map_err(|error| error.with_sketch_identity(sketch).for_feature(feature))?;
    Ok((result, resolution))
}

/// Evaluate a current, converged saved prism with an explicit modelling resolution.
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
    let plane = input.sweep_start;
    let (profile, sources) = super::regions::input::profile(input.sketch.sketch.geometry(), plane)?;
    let vector = crate::prism::placement::model_displacement(plane, input.displacement)?;
    let taper = crate::prism::PrismTaper::new(plane, vector, input.taper_radians);
    let mut result = crate::prism::retain_certified_tapered_prism(
        &profile,
        input.selection,
        [vector.x(), vector.y(), vector.z()],
        resolution,
        taper,
    )?;
    result.vertex_sources = sources;
    Ok(result)
}

pub(super) fn profile_line(
    sketch: &oblikovati_model::sketch::Sketch2D,
    directed: SketchDirectedCurve,
    coincidence: &super::incidence::CoincidentPoints,
    vertices: &mut BTreeMap<ProfileVertexId, ProfileVertex>,
) -> Result<ProfileCurve, EvaluationFailure> {
    let line = boundary::require_boundary_line(sketch, directed.curve)?;
    let [a, b] = line.endpoints;
    let start = profile_vertex(coincidence, vertices, a)?;
    let end = profile_vertex(coincidence, vertices, b)?;
    Ok(ProfileCurve::Line {
        id: ProfileCurveId(line.id.0),
        start,
        end,
    })
}
pub(super) fn profile_vertex(
    coincidence: &super::incidence::CoincidentPoints,
    vertices: &mut BTreeMap<ProfileVertexId, ProfileVertex>,
    id: oblikovati_model::sketch::SketchEntityId,
) -> Result<ProfileVertexId, EvaluationFailure> {
    let point = coincidence.vertex(id)?;
    let identity = ProfileVertexId(point.id.0);
    vertices.insert(
        identity,
        ProfileVertex {
            id: identity,
            point: point.position,
        },
    );
    Ok(identity)
}
