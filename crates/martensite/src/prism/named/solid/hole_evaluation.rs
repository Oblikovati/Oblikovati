// SPDX-License-Identifier: GPL-2.0-only
//! Construct a named Hole cut during ordered part replay (#4093).
use super::*;
use martensite_tkmath::bnd::BoundingBox3;
use martensite_tktopalgo::brep_bnd_lib::BRepBndLib;
use oblikovati_model::feature::hole::ResolvedHole;

impl NamedSolid {
    /// Read measured body tolerances when available; example: `body.achieved_tolerance()` after Hole.
    pub fn achieved_tolerance(&self) -> Option<PrismAchievedTolerance> {
        match self {
            Self::Prism(body) => body
                .certification
                .as_ref()
                .map(|value| value.achieved_tolerance),
            Self::Derived(body) => body.achieved_tolerance(),
        }
    }
}

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_hole(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        face: &[u8],
        input: ResolvedHole,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let resolution = match body {
            NamedSolid::Prism(body) => body.certification.as_ref().map(|value| value.resolution),
            NamedSolid::Derived(body) => body.resolution(),
        }
        .ok_or_else(|| {
            PartFeatureFailure::Unsupported(
                "hole: expected the source body's certified resolution".into(),
            )
        })?;
        let shape = body
            .native_shape(self.history, self.document)
            .map_err(PartFeatureFailure::BodyReference)?;
        let bounds =
            analytic_bounds(shape, "hole").map_err(|message| hole_failure(feature, message))?;
        let axis = body
            .hole_axis(
                self.history,
                self.document,
                face,
                input.position,
                resolution,
            )
            .map_err(|message| hole_failure(feature, message))?;
        let depth = match input.depth {
            Some(depth) => depth,
            None => {
                through_depth(bounds, axis).map_err(|message| hole_failure(feature, message))?
            }
        };
        let request = blend::HoleRequest {
            feature,
            face,
            axis,
            diameter: input.diameter,
            depth,
        };
        let candidate = match body {
            NamedSolid::Prism(body) => body.hole_candidate(self.history, self.document, &request),
            NamedSolid::Derived(body) => body.hole_candidate(self.history, self.document, &request),
        }
        .map_err(|message| hole_failure(feature, message))?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| hole_failure(feature, message))?;
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}

fn hole_failure(feature: FeatureId, message: String) -> PartFeatureFailure<EvaluationFailure> {
    PartFeatureFailure::Kernel(
        Phase::Hole
            .refused(Kind::InvalidResult, Entity::Body, message)
            .for_feature(feature),
    )
}
pub(super) fn analytic_bounds(shape: &Shape, operation: &str) -> Result<[f64; 6], String> {
    let mut bounds = BoundingBox3::default();
    BRepBndLib::add_optimal(shape, &mut bounds, false, false)
        .map_err(|error| format!("{operation} body bounds: expected analytic bounds: {error}"))?;
    let result = bounds
        .get()
        .map_err(|error| format!("{operation} body bounds: expected finite limits: {error}"))?;
    if result.iter().any(|value| !value.is_finite()) {
        return Err(format!(
            "{operation} body bounds {result:?}: expected finite limits"
        ));
    }
    Ok(result)
}
fn through_depth(bounds: [f64; 6], axis: [[f64; 3]; 2]) -> Result<f64, String> {
    let [origin, direction] = axis;
    let depth: f64 = (0..3)
        .map(|index| {
            let extreme = if direction[index] >= 0. {
                bounds[index + 3]
            } else {
                bounds[index]
            };
            (extreme - origin[index]) * direction[index]
        })
        .sum();
    if !depth.is_finite() || depth <= 0. {
        return Err(format!(
            "hole through depth {depth}: expected positive finite span"
        ));
    }
    Ok(depth)
}
