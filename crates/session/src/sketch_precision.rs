// SPDX-License-Identifier: GPL-2.0-only
//! Session-borrowed conditional evidence; no feature admission.
mod prism;
use crate::{DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{ConditionalOriginProjection, DocumentSnapshot},
    sketch::{SketchEntityId, SketchSolveOutcome},
};
pub use prism::PrismPrecisionObligation;

/// A fresh solve retained only while its owning session is immutably borrowed.
/// Derived numeric bounds can be copied and do not carry this lifetime.
///
/// ```compile_fail
/// use oblikovati_session::{DocumentSession, TransactionObserver};
/// use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchEntityId};
/// fn cannot_mutate<O: TransactionObserver>(session: &mut DocumentSession<O>) {
///     let evidence = session.conditional_canonical_sketch(DocumentSketchId(1), SketchEntityId(1)).unwrap();
///     session.undo().unwrap();
///     let _ = evidence.projection();
/// }
/// ```
///
/// ```compile_fail
/// use oblikovati_session::{DocumentSession, SessionConditionalSketch, TransactionObserver};
/// use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchEntityId};
/// fn cannot_outlive<'a, O: TransactionObserver>(mut session: DocumentSession<O>) -> SessionConditionalSketch<'a, O> {
///     session.conditional_canonical_sketch(DocumentSketchId(1), SketchEntityId(1)).unwrap()
/// }
/// ```
pub struct SessionConditionalSketch<'a, O> {
    session: &'a DocumentSession<O>,
    snapshot: DocumentSnapshot,
    sketch: DocumentSketchId,
    anchor: SketchEntityId,
    outcome: SketchSolveOutcome,
}
impl<O: TransactionObserver> DocumentSession<O> {
    /// Solve and bind conditional evidence to the current session borrow.
    /// Example: `session.conditional_canonical_sketch(sketch, anchor)?`.
    /// A read-only fresh solve must preserve the current authored sketch recipe.
    pub fn conditional_canonical_sketch(
        &self,
        sketch: DocumentSketchId,
        anchor: SketchEntityId,
    ) -> Result<SessionConditionalSketch<'_, O>, String> {
        self.snapshot()
            .require_converged_resolved_sketch(sketch)
            .map_err(|e| e.to_string())?;
        let batch = self
            .snapshot()
            .solve_one_document_sketch(sketch, self.sketch_solve_options(), || false)
            .map_err(|e| e.to_string())?;
        if !self.snapshot().sketches.same_recipe(&batch.candidate) {
            return Err(format!(
                "sketch {sketch:?}: fresh solve changed authored geometry; explicit recompute required"
            ));
        }
        let mut snapshot = self.snapshot().clone();
        snapshot.sketches = batch.candidate;
        let mut reports = batch.report.reports.into_iter();
        let report = reports
            .next()
            .ok_or_else(|| format!("sketch {sketch:?}: expected one fresh solve report"))?;
        if batch.report.cancelled || report.sketch != sketch || reports.next().is_some() {
            return Err(format!(
                "sketch {sketch:?}: expected one uncancelled matching solve report"
            ));
        }
        let outcome = report.result.map_err(|e| e.to_string())?;
        snapshot.canonical_origin_projection(sketch, anchor, &outcome)?;
        Ok(SessionConditionalSketch {
            session: self,
            snapshot,
            sketch,
            anchor,
            outcome,
        })
    }
}
impl<O: TransactionObserver> SessionConditionalSketch<'_, O> {
    /// Borrow the current session's conditional projection; example: `evidence.projection()?`.
    pub fn projection(&self) -> Result<ConditionalOriginProjection<'_>, String> {
        self.snapshot
            .canonical_origin_projection(self.sketch, self.anchor, &self.outcome)
    }
    /// Read the worker generation at this session borrow; example: `evidence.generation()`.
    pub fn generation(&self) -> u64 {
        self.session.generation()
    }
}
