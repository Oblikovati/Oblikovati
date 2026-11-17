// SPDX-License-Identifier: GPL-2.0-only
//! The named-edge chamfer capability exposed by a geometry backend (#4046).
use crate::feature::FeatureId;

/// A backend that constructs and verifies a replacement chamfer body.
pub trait ChamferEvaluator<Body, Failure> {
    /// Construct a detached result; example: `backend.evaluate_chamfer(id, body, edges, distance)?`.
    fn evaluate_chamfer(
        &mut self,
        feature: FeatureId,
        body: &Body,
        edges: &[Vec<u8>],
        distance: f64,
    ) -> Result<Body, Failure>;
    /// Verify the input names and deletion records; example: `backend.verify_chamfer_lineage(before, after)?`.
    fn verify_chamfer_lineage(&self, before: &Body, after: &Body) -> Result<(), String>;
}
