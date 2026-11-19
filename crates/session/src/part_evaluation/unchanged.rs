// SPDX-License-Identifier: GPL-2.0-only
//! Capture current evaluated inputs before a metadata-only transition.
use super::*;
impl<Body, Failure: Clone> PartCandidatePrefix<Body, Failure> {
    /// Capture an unchanged recipe for an accepted metadata edit.
    /// Example: `prefix.capture_unchanged(history, candidate, |_, body| copy(body))?`.
    pub fn capture_unchanged<Mapped>(
        &self,
        history: &oblikovati_model::history::DocumentEditHistory,
        candidate: &oblikovati_model::history::DocumentSnapshot,
        capture: impl FnMut(oblikovati_model::feature::FeatureId, &Body) -> Result<Mapped, String>,
    ) -> Result<CapturedPartCandidate<Mapped, Failure>, String> {
        if !self.published.is_active() || !self.published.scope().is_current(history) {
            return Err("part capture: expected the current owning history".into());
        }
        let evaluation = self.published.evaluation.try_map_bodies(capture)?;
        if !self.published.is_active() || !self.published.scope().is_current(history) {
            return Err("part capture: expected unchanged ownership after capture".into());
        }
        Ok(CapturedPartCandidate {
            transfer: oblikovati_model::feature::part::CapturedPartTransfer::capture_result(
                self.published.document,
                self.published.scope(),
                candidate,
                evaluation,
                history,
            )?,
        })
    }
}
