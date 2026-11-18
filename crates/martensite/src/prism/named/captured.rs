// SPDX-License-Identifier: GPL-2.0-only
//! Capture and restore exact names only through an accepted session transition.
use super::*;
use oblikovati_model::feature::part::{
    CapturedPartTransfer, CompletedPartTransfer, PartFeatureEvaluation, PartReplayCheckpoints,
};
use oblikovati_model::history::{EvaluationScope, EvaluationTransition};

/// Opaque body capture; restoring it requires a matching committed candidate.
pub struct CapturedNamedSolid {
    document: DocumentGuid,
    source: EvaluationScope,
    body: CapturedBody,
}
enum CapturedBody {
    Prism(CapturedPrism),
    Derived(blend::CapturedBlendTopology),
}
struct CapturedPrism {
    snapshot: PrismProbeSnapshot,
    entries: Vec<(Vec<u8>, Option<PrismFaceRole>)>,
    subentities: Option<Vec<(Vec<u8>, subentities::BoundSubentity)>>,
    certification: Option<CertifiedPrismEvidence>,
}
impl NamedSolid {
    /// Capture exact entities before the source generation retires.
    /// Example: `body.capture_detached(&history, document)?` in a candidate capture.
    pub fn capture_detached(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<CapturedNamedSolid, String> {
        let body = match self {
            Self::Prism(body) => {
                CapturedBody::Prism(CapturedPrism::capture(body, history, document)?)
            }
            Self::Derived(body) => CapturedBody::Derived(body.capture_detached(history, document)?),
        };
        Ok(CapturedNamedSolid {
            document,
            source: history.evaluation_scope(),
            body,
        })
    }
}
impl CapturedPrism {
    fn capture(
        body: &NamedPrismSnapshot,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<Self, String> {
        Ok(Self {
            snapshot: body.snapshot.clone(),
            entries: body.bindings.capture_entries(history, document)?,
            subentities: body
                .subentity_bindings
                .as_ref()
                .map(|bindings| bindings.capture_entries(history, document))
                .transpose()?,
            certification: body.certification.clone(),
        })
    }
    fn restore(
        self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<NamedPrismSnapshot, String> {
        let expected = self.entries.clone();
        let subentities = self
            .subentities
            .map(|entries| restore_subentities(entries, history, document))
            .transpose()?;
        let bindings = ExactReferenceBindings::from_entries(
            document,
            history.evaluation_scope(),
            self.entries,
        )?;
        for (key, role) in expected {
            let kind = oblikovati_model::identity::ReferenceEnvelope::decode(&key)
                .map_err(|error| error.to_string())?
                .kind;
            if *bindings.bind(history, document, &key, kind)? != role {
                return Err(format!("prism name {key:?}: expected its captured role"));
            }
        }
        Ok(NamedPrismSnapshot {
            snapshot: self.snapshot,
            bindings,
            subentity_bindings: subentities,
            certification: self.certification,
        })
    }
}
fn restore_subentities(
    entries: Vec<(Vec<u8>, subentities::BoundSubentity)>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
) -> Result<ExactReferenceBindings<subentities::BoundSubentity>, String> {
    let expected = entries.clone();
    let bindings =
        ExactReferenceBindings::from_entries(document, history.evaluation_scope(), entries)?;
    for (key, entity) in expected {
        let kind = oblikovati_model::identity::ReferenceEnvelope::decode(&key)
            .map_err(|error| error.to_string())?
            .kind;
        if !bindings
            .bind(history, document, &key, kind)?
            .shape
            .is_same(&entity.shape)
        {
            return Err(format!(
                "prism subentity {key:?}: expected the captured entity"
            ));
        }
    }
    Ok(bindings)
}
impl CapturedNamedSolid {
    fn restore_for_transition(
        self,
        transition: &EvaluationTransition,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<NamedSolid, String> {
        if !transition.matches(&self.source, history) {
            return Err("captured named solid: expected its accepted source transition".into());
        }
        self.restore(history, document)
    }
    fn restore(
        self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<NamedSolid, String> {
        if self.document != document {
            return Err(format!(
                "captured document {}: expected {document}",
                self.document
            ));
        }
        match self.body {
            CapturedBody::Prism(body) => Ok(NamedSolid::Prism(Box::new(
                body.restore(history, document)?,
            ))),
            CapturedBody::Derived(body) => Ok(NamedSolid::Derived(Box::new(
                body.restore(history, document)?,
            ))),
        }
    }
}
/// Complete exact named bodies after their owning history accepts the candidate.
/// Example: `complete_named_candidate(captured, document_id, &transition, &history, document)?`.
pub fn complete_named_candidate<Failure>(
    candidate: CapturedPartTransfer<PartFeatureEvaluation<CapturedNamedSolid, Failure>>,
    document_id: u64,
    transition: &EvaluationTransition,
    history: &DocumentEditHistory,
    document: DocumentGuid,
) -> Result<CompletedPartTransfer<PartFeatureEvaluation<NamedSolid, Failure>>, String> {
    candidate.complete(document_id, transition, history, |result, history| {
        result.try_map_owned_bodies(|_, body| {
            body.restore_for_transition(transition, history, document)
        })
    })
}
/// Restore named boundaries after their owning history accepts the candidate.
/// Example: `complete_named_checkpoints(captured, document_id, &transition, &history, document)?`.
pub fn complete_named_checkpoints<Failure>(
    candidate: CapturedPartTransfer<PartReplayCheckpoints<CapturedNamedSolid, Failure>>,
    document_id: u64,
    transition: &EvaluationTransition,
    history: &DocumentEditHistory,
    document: DocumentGuid,
) -> Result<CompletedPartTransfer<PartReplayCheckpoints<NamedSolid, Failure>>, String> {
    candidate.complete(document_id, transition, history, |result, history| {
        result.try_map_owned_bodies(|_, body| {
            body.restore_for_transition(transition, history, document)
        })
    })
}
