// SPDX-License-Identifier: GPL-2.0-only
//! Injected ordered part feature evaluation boundary.
use super::{PartBlendKind, PartFeatureFailure};
use crate::feature::{FeatureId, sketch_prism::ResolvedSketchPrism};
/// The model resolves current source inputs; the injected adapter owns kernel construction and validation.
pub trait PartFeatureEvaluator {
    type Body;
    type Failure;
    /// Construct a named shell or report a named decline; example: `backend.evaluate_shell(id, body, faces, input)`.
    fn evaluate_shell(
        &mut self,
        _feature: FeatureId,
        _body: &Self::Body,
        _faces: &[Vec<u8>],
        _input: crate::feature::shell::ResolvedShell,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(
            "shell: expected an available named-face evaluator".into(),
        ))
    }
    /// Verify the complete body name before a body-set edit; example: `backend.verify_body_reference(body, key)?`.
    fn verify_body_reference(&self, body: &Self::Body, key: &[u8]) -> Result<(), String>;
    /// Resolve a current named planar face; example: `backend.resolve_planar_face(body, key)?`.
    fn resolve_planar_face(
        &self,
        body: &Self::Body,
        key: &[u8],
    ) -> Result<crate::sketch::SketchPlane, String>;
    /// Resolve a named vertex or circular edge center; example: `backend.resolve_frame_origin(body, key)?`.
    fn resolve_frame_origin(&self, _body: &Self::Body, key: &[u8]) -> Result<[f64; 3], String> {
        Err(format!(
            "frame origin {key:?}: expected an exact named vertex or circular-edge resolver"
        ))
    }
    /// Construct and validate one blend or return a named decline.
    /// Example: `backend.evaluate_blend(kind, feature, body, edges, distance)`.
    fn evaluate_blend(
        &mut self,
        kind: PartBlendKind,
        feature: FeatureId,
        body: &Self::Body,
        edges: &[Vec<u8>],
        distance: f64,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>>;
    /// Transform a named body; example: `backend.evaluate_move_body(id, body, &motions)`.
    fn evaluate_move_body(
        &mut self,
        _feature: FeatureId,
        _body: &Self::Body,
        _motions: &[crate::feature::move_body::ResolvedBodyMotion],
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(
            "move body: expected a named rigid-motion evaluator".into(),
        ))
    }
    /// Construct a named straight bore; example: `backend.evaluate_hole(id, body, face, input)`.
    fn evaluate_hole(
        &mut self,
        _feature: FeatureId,
        _body: &Self::Body,
        _face: &[u8],
        _input: crate::feature::hole::ResolvedHole,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(
            "hole: expected an available named-face evaluator".into(),
        ))
    }
    /// Resolve the sweep from a sketch plane to the current target body's bounds.
    /// Example: `backend.prism_through_all_span(body, plane, direction)?`.
    fn prism_through_all_span(
        &self,
        _body: &Self::Body,
        _plane: crate::sketch::SketchPlane,
        _direction: crate::feature::sketch_prism::ThroughAllDirection,
    ) -> Result<(f64, f64), String> {
        Err("extrude Through All: expected an analytic body bounds evaluator".into())
    }
    /// Replace a named target. Example: `backend.evaluate_prism_boolean(id, body, input, operation)`.
    fn evaluate_prism_boolean(
        &mut self,
        _feature: FeatureId,
        _body: &Self::Body,
        _input: ResolvedSketchPrism<'_>,
        operation: crate::feature::sketch_prism::PrismBoolean,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(format!(
            "extrude {operation}: expected a named Boolean evaluator"
        )))
    }
    /// Apply the resolved extrusion tools at each grid occurrence.
    /// Example: `backend.evaluate_extrusion_pattern(id, body, &sources, &occurrences)`.
    fn evaluate_extrusion_pattern(
        &mut self,
        _feature: FeatureId,
        _body: &Self::Body,
        _sources: &[crate::feature::extrusion_pattern::PatternPrismSource<'_>],
        _occurrences: &[crate::feature::extrusion_pattern::ExtrusionPatternOccurrence],
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(
            "rectangular pattern: expected a named tool replay backend".into(),
        ))
    }
    /// Construct a named solid from a solved profile and path. Example: `backend.evaluate_sweep(feature, input)?`.
    fn evaluate_sweep(
        &mut self,
        _feature: FeatureId,
        _input: crate::feature::sketch_sweep::ResolvedSketchSweep<'_>,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(
            "sweep: expected an available named sweep evaluator".into(),
        ))
    }
    /// Construct an isolated result from checked inputs; example: `backend.evaluate_prism(feature.id(), input)?`.
    fn evaluate_prism(
        &mut self,
        feature: FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<Self::Body, Self::Failure>;
}
