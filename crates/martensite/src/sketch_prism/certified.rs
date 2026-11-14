// SPDX-License-Identifier: GPL-2.0-only
//! Convert current document-owned sketch identities to the released profile operation.
use super::*;
use martensite_tkbo::profile_extrusion::{
    ProfileCurve, ProfileCurveId, ProfileVertex, ProfileVertexId, Resolution,
};
use oblikovati_model::feature::sketch_prism::ResolvedSketchPrism;
use oblikovati_model::feature::{FeatureId, part::SketchPrismEvaluator};
use std::collections::BTreeMap;

/// Certified replay for candidate validation with an explicit modelling resolution.
pub struct CertifiedPrismEvaluator {
    resolution: crate::PrismResolutionPolicy,
}
impl CertifiedPrismEvaluator {
    /// Bind the caller's resolution; example: `CertifiedPrismEvaluator::new(resolution)`.
    pub fn new(resolution: crate::PrismResolution) -> Self {
        Self {
            resolution: crate::PrismResolutionPolicy::Fixed(resolution),
        }
    }
    /// Bind per-sketch policy; example: `CertifiedPrismEvaluator::with_policy(policy)`.
    pub fn with_policy(resolution: crate::PrismResolutionPolicy) -> Self {
        Self { resolution }
    }
}
impl SketchPrismEvaluator for CertifiedPrismEvaluator {
    type Body = crate::CertifiedPrismSnapshot;
    type Failure = EvaluationFailure;
    fn verify_body_reference(&self, _body: &Self::Body, key: &[u8]) -> Result<(), String> {
        Err(format!(
            "certified prism reference {key:?}: expected a document-scoped named evaluator"
        ))
    }
    fn evaluate_prism(
        &mut self,
        feature: FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<Self::Body, Self::Failure> {
        evaluate_feature(input, self.resolution, feature)
    }
}

pub(crate) fn evaluate_feature(
    input: ResolvedSketchPrism<'_>,
    resolution: crate::PrismResolutionPolicy,
    feature: FeatureId,
) -> Result<crate::CertifiedPrismSnapshot, EvaluationFailure> {
    let sketch = input.sketch.sketch.id();
    let resolution = resolution
        .for_sketch(&input.sketch)
        .map_err(|error| error.for_feature(feature))?;
    evaluate(input, resolution.kernel())
        .map_err(|error| error.with_sketch_identity(sketch).for_feature(feature))
}

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
    let (profile, sources) = super::regions::input::profile(input.sketch.sketch.geometry(), plane)?;
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

pub(super) fn profile_line(
    sketch: &oblikovati_model::sketch::Sketch2D,
    directed: SketchDirectedLine,
    coincidence: &super::incidence::CoincidentPoints,
    vertices: &mut BTreeMap<ProfileVertexId, ProfileVertex>,
) -> Result<ProfileCurve, EvaluationFailure> {
    let line = boundary::require_boundary_line(sketch, directed.line)?;
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
