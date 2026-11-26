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
    /// Subtract a prism from a named target. Example: `backend.evaluate_prism_cut(id, body, input)`.
    fn evaluate_prism_cut(
        &mut self,
        _feature: FeatureId,
        _body: &Self::Body,
        _input: ResolvedSketchPrism<'_>,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(
            "extrude Cut: expected a named subtraction evaluator".into(),
        ))
    }
    /// Construct an isolated result from checked inputs; example: `backend.evaluate_prism(feature.id(), input)?`.
    fn evaluate_prism(
        &mut self,
        feature: FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<Self::Body, Self::Failure>;
}
