// SPDX-License-Identifier: GPL-2.0-only
//! Injected ordered part feature evaluation boundary.
use super::{PartBlendKind, PartFeatureFailure};
use crate::feature::{
    FeatureId,
    sketch_prism::{PrismBoolean, ResolvedSketchPrism},
};
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
    /// Construct and validate an extended v2 Fillet; defaults to `evaluate_blend` for constant radii.
    fn evaluate_fillet_v2(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        edges: &[Vec<u8>],
        input: crate::feature::fillet_options::ResolvedFillet,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.evaluate_blend(PartBlendKind::Fillet, feature, body, edges, input.radius)
    }
    /// Construct and validate an extended v2 Chamfer; defaults to `evaluate_blend` for equal distance.
    fn evaluate_chamfer_v2(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        edges: &[Vec<u8>],
        input: crate::feature::chamfer_options::ResolvedChamfer,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.evaluate_blend(PartBlendKind::Chamfer, feature, body, edges, input.distance)
    }
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
    /// Construct a named machined slot cut; example: `backend.evaluate_slot(id, body, face, input)`.
    fn evaluate_slot(
        &mut self,
        _feature: FeatureId,
        _body: &Self::Body,
        _face: &[u8],
        _input: crate::feature::slot::ResolvedSlot,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(
            "slot: expected an available named-face evaluator".into(),
        ))
    }
    /// Combine a base solid with one or more tool solids; example: `backend.evaluate_combine(id, base, &tools, input)`.
    fn evaluate_combine(
        &mut self,
        _feature: FeatureId,
        _base: &Self::Body,
        _tools: &[&Self::Body],
        _input: crate::feature::combine::ResolvedCombine,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(
            "combine: expected an available solid Boolean evaluator".into(),
        ))
    }
    /// Thicken selected faces into a solid or offset them into a surface shell.
    fn evaluate_thicken_offset(
        &mut self,
        _feature: FeatureId,
        _body: &Self::Body,
        _input: crate::feature::thicken_offset::ResolvedThickenOffset,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(
            "thicken/offset: expected an available face thicken/offset evaluator".into(),
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
    /// Construct a named solid from solved Loft sections. Example: `backend.evaluate_loft(feature, sections)?`.
    fn evaluate_loft(
        &mut self,
        _feature: FeatureId,
        _input: crate::feature::sketch_loft::ResolvedSketchLoft,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(
            "loft: expected an available section evaluator".into(),
        ))
    }
    /// Construct a named solid from a solved Revolve profile and axis. Example: `backend.evaluate_revolve(feature, input)?`.
    fn evaluate_revolve(
        &mut self,
        _feature: FeatureId,
        _input: crate::feature::sketch_revolve::ResolvedSketchRevolve<'_>,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(
            "revolve: expected an available revolution evaluator".into(),
        ))
    }
    /// Replace a named body with a Revolve Boolean result. Example: `backend.evaluate_revolve_boolean(feature, body, input, op)?`.
    fn evaluate_revolve_boolean(
        &mut self,
        _feature: FeatureId,
        _body: &Self::Body,
        _input: crate::feature::sketch_revolve::ResolvedSketchRevolve<'_>,
        operation: PrismBoolean,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(format!(
            "revolve {operation}: expected a named Boolean evaluator"
        )))
    }
    /// Construct a named solid from a solved Coil profile and axis. Example: `backend.evaluate_coil(feature, input)?`.
    fn evaluate_coil(
        &mut self,
        _feature: FeatureId,
        _input: crate::feature::sketch_coil::ResolvedSketchCoil<'_>,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(
            "coil: expected an available coil evaluator".into(),
        ))
    }
    /// Replace a named body with a Coil Boolean result. Example: `backend.evaluate_coil_boolean(feature, body, input, op)?`.
    fn evaluate_coil_boolean(
        &mut self,
        _feature: FeatureId,
        _body: &Self::Body,
        _input: crate::feature::sketch_coil::ResolvedSketchCoil<'_>,
        operation: PrismBoolean,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(format!(
            "coil {operation}: expected a named Boolean evaluator"
        )))
    }
    /// Construct a named derived solid from an external source part.
    fn evaluate_derived_part(
        &mut self,
        _feature: FeatureId,
        _input: crate::feature::derived_part::ResolvedDerivedPart,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(
            "derive: expected an available derived-part evaluator".into(),
        ))
    }
    /// Construct a named imported body from an external STL or STEP file.
    fn evaluate_imported_model(
        &mut self,
        _feature: FeatureId,
        _input: crate::feature::imported_model::ResolvedImportedModel,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(
            "import: expected an available imported-model evaluator".into(),
        ))
    }
    /// Unfold one or more selected solid faces into a planar surface body.
    fn evaluate_unwrap(
        &mut self,
        _feature: FeatureId,
        _body: &Self::Body,
        _input: crate::feature::unwrap::ResolvedUnwrap<'_>,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(
            "unwrap: expected an available unwrap evaluator".into(),
        ))
    }
    /// Apply a solved Emboss (raise, engrave, or level-to-plane) to a target body.
    fn evaluate_emboss(
        &mut self,
        _feature: FeatureId,
        _body: &Self::Body,
        _input: crate::feature::sketch_emboss::ResolvedSketchEmboss<'_>,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(
            "emboss: expected an available emboss evaluator".into(),
        ))
    }
    /// Apply a solved Rib (open profile web/rib) to a target body.
    fn evaluate_rib(
        &mut self,
        _feature: FeatureId,
        _body: &Self::Body,
        _input: crate::feature::sketch_rib::ResolvedSketchRib<'_>,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(
            "rib: expected an available rib evaluator".into(),
        ))
    }
    /// Apply a solved Decal (sketch image on a target face) to a target body.
    fn evaluate_decal(
        &mut self,
        _feature: FeatureId,
        _body: &Self::Body,
        _input: crate::feature::sketch_decal::ResolvedSketchDecal<'_>,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(
            "decal: expected an available decal evaluator".into(),
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
    /// Combine a solved Sweep with a live target. Example: `backend.evaluate_sweep_boolean(feature, body, input, operation)?`.
    fn evaluate_sweep_boolean(
        &mut self,
        _feature: FeatureId,
        _body: &Self::Body,
        _input: crate::feature::sketch_sweep::ResolvedSketchSweep<'_>,
        _operation: PrismBoolean,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(
            "sweep: expected an available named Boolean evaluator".into(),
        ))
    }
    /// Construct all solid outputs in authored order, including the primary result.
    /// Example: `backend.evaluate_prism_outputs(feature, input)?`.
    fn evaluate_prism_outputs(
        &mut self,
        _feature: FeatureId,
        _input: ResolvedSketchPrism<'_>,
    ) -> Result<Vec<Self::Body>, PartFeatureFailure<Self::Failure>> {
        Err(PartFeatureFailure::Unsupported(
            "extrude: expected an available multiple-output evaluator".into(),
        ))
    }
    /// Construct an isolated result from checked inputs; example: `backend.evaluate_prism(feature.id(), input)?`.
    fn evaluate_prism(
        &mut self,
        feature: FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<Self::Body, Self::Failure>;
}
