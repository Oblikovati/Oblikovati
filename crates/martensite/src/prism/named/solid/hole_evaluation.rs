// SPDX-License-Identifier: GPL-2.0-only
//! Construct a named Hole cut during ordered part replay (#4093).
use super::*;
use martensite_tkmath::bnd::BoundingBox3;
use martensite_tktopalgo::brep_bnd_lib::BRepBndLib;
use oblikovati_model::feature::hole::{HoleTerminationSurface, ResolvedHole};

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

    /// Build display triangles for the Hole cut volume (including flipped-direction & Extend Start previews).
    pub fn hole_preview_triangles(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        face: &[u8],
        input: &ResolvedHole,
        options: crate::PrismMeshOptions,
    ) -> Result<Vec<crate::DisplayTriangle>, String> {
        let resolution = match self {
            NamedSolid::Prism(body) => body.certification.as_ref().map(|value| value.resolution),
            NamedSolid::Derived(body) => body.resolution(),
        }
        .ok_or_else(|| {
            "hole preview: expected the source body's certified resolution".to_string()
        })?;
        let shape = self
            .native_shape(history, document)
            .map_err(|cause| cause.to_string())?;
        let bounds = analytic_bounds(shape, "hole preview")?;
        let mut axis = self.hole_axis(history, document, face, input.position, resolution)?;
        let inward_depth = through_depth(bounds, axis).unwrap_or(input.diameter * 2.0);
        if input.direction_flipped {
            axis[1] = axis[1].map(|component| -component);
        }
        let (depth, finite_depth) = match (input.depth, input.termination_surface.as_ref()) {
            (Some(depth), _) => (depth, true),
            (None, Some(surface)) => (
                surface_depth(history, document, self, shape, axis, surface)
                    .unwrap_or(inward_depth),
                false,
            ),
            (None, None) => (through_depth(bounds, axis).unwrap_or(inward_depth), false),
        };
        let request = blend::HoleRequest {
            feature: FeatureId(1),
            face,
            axis,
            diameter: input.diameter,
            depth,
            seat: input.seat,
            drill_point: input.drill_point,
            extend_start: input.extend_start,
            finite_depth,
        };
        let tool = blend::build_hole_tool(document, &request, Some(resolution))?;
        Ok(
            crate::prism::mesh::body_face_display_triangles(&tool.shape, options)?
                .into_iter()
                .flatten()
                .collect(),
        )
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
        let mut axis = body
            .hole_axis(
                self.history,
                self.document,
                face,
                input.position,
                resolution,
            )
            .map_err(|message| hole_failure(feature, message))?;
        if input.direction_flipped {
            axis[1] = axis[1].map(|component| -component);
        }
        let (depth, finite_depth) = match (input.depth, input.termination_surface.as_ref()) {
            (Some(depth), _) => (depth, true),
            (None, Some(surface)) => {
                let d = surface_depth(self.history, self.document, body, shape, axis, surface)
                    .map_err(|message| hole_failure(feature, message))?;
                (d, false)
            }
            (None, None) => (
                through_depth(bounds, axis).map_err(|message| hole_failure(feature, message))?,
                false,
            ),
        };
        let request = blend::HoleRequest {
            feature,
            face,
            axis,
            diameter: input.diameter,
            depth,
            seat: input.seat,
            drill_point: input.drill_point,
            extend_start: input.extend_start,
            finite_depth,
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
pub(super) fn through_depth(bounds: [f64; 6], axis: [[f64; 3]; 2]) -> Result<f64, String> {
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

pub(super) fn surface_depth(
    history: &DocumentEditHistory,
    document: DocumentGuid,
    body: &NamedSolid,
    shape: &Shape,
    axis: [[f64; 3]; 2],
    surface: &HoleTerminationSurface,
) -> Result<f64, String> {
    let (plane_origin, plane_normal) = match surface {
        HoleTerminationSurface::OriginPlane(plane) => plane.frame(),
        HoleTerminationSurface::Face(face_key) => {
            let target = body.native_face(history, document, face_key)?;
            let frame = super::super::face_topology::planar_shape_frame(shape, target)?;
            (frame.recipe().origin, frame.normal())
        }
    };
    let denom: f64 = axis[1]
        .into_iter()
        .zip(plane_normal)
        .map(|(a, b)| a * b)
        .sum();
    if denom.abs() <= 1e-9 {
        return Err("hole terminating surface: expected a plane intersecting the hole axis".into());
    }
    let numer: f64 = plane_origin
        .into_iter()
        .zip(axis[0])
        .zip(plane_normal)
        .map(|((p, o), n)| (p - o) * n)
        .sum();
    let depth = numer / denom;
    if !depth.is_finite() || depth <= 1e-9 {
        return Err(format!(
            "hole terminating surface depth {depth}: expected a surface ahead of the placement face"
        ));
    }
    Ok(depth)
}
