// SPDX-License-Identifier: GPL-2.0-only
//! Feature-specific conditional diagnostics; this is not an admission gate.
use super::SessionConditionalSketch;
use crate::TransactionObserver;
use oblikovati_model::{feature::FeatureId, history::ConditionalProjectedSketchRoot};

/// Proof obligations that remain open after conditional line evidence was checked.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum PrismPrecisionObligation {
    /// The interval backend has not been qualified on the running platform.
    ArithmeticPlatform,
    /// The extrusion displacement source has no propagated physical bound.
    ExtrusionSource,
    /// The complete profile's coincidence identity mapping is not proven here.
    ProfileIdentityMapping,
    /// The adapter's bit-identical coincidence admission is not proven here.
    ExactCoincidenceAdmission,
    /// A model tolerance budget for the relevant operand origins is unavailable.
    OperandToleranceBudget,
    /// All profile, preview, commit, replay and direct adapter paths need one gate.
    AdmissionCoverage,
}
impl<O: TransactionObserver> SessionConditionalSketch<'_, O> {
    /// Inspect a current authored prism using conditional selected-line evidence.
    /// Example: `evidence.prism_precision_obligations(feature, &bounds)?`.
    /// This diagnostic never returns a certificate and does not authorize a feature.
    pub fn prism_precision_obligations(
        &self,
        feature: FeatureId,
        bounds: &[[f64; 2]],
    ) -> Result<Vec<PrismPrecisionObligation>, String> {
        let projection = self.projection()?;
        let root = projection.conditional_root_auto(bounds, &[])?;
        self.prism_precision_obligations_with_root(feature, &root)
    }
    /// Check the same non-admitting obligations with this session's retained root.
    /// Example: `evidence.prism_precision_obligations_with_root(feature, &root)?`.
    pub fn prism_precision_obligations_with_root(
        &self,
        feature: FeatureId,
        root: &ConditionalProjectedSketchRoot<'_, '_>,
    ) -> Result<Vec<PrismPrecisionObligation>, String> {
        self.require_root(root)?;
        let input = self
            .prism_definition(feature)?
            .resolve(self.session.snapshot())
            .map_err(|e| e.to_string())?;
        let segments = root.canonical_segment_errors()?;
        for edge in std::iter::once(&input.selection.outer)
            .chain(&input.selection.holes)
            .flatten()
        {
            if !segments.iter().any(|segment| segment.line == edge.curve) {
                return Err(format!(
                    "feature {feature:?} curve {:?}: expected a supported line with conditional direction evidence",
                    edge.curve
                ));
            }
        }
        Ok(vec![
            PrismPrecisionObligation::ArithmeticPlatform,
            PrismPrecisionObligation::ExtrusionSource,
            PrismPrecisionObligation::ProfileIdentityMapping,
            PrismPrecisionObligation::ExactCoincidenceAdmission,
            PrismPrecisionObligation::OperandToleranceBudget,
            PrismPrecisionObligation::AdmissionCoverage,
        ])
    }
}
