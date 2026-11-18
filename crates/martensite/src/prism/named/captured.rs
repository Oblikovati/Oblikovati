// SPDX-License-Identifier: GPL-2.0-only
//! Capture and restore exact names only through an accepted session transition.
use super::*;
use oblikovati_model::history::EvaluationTransition;
use oblikovati_session::{
    DocumentSession, TransactionObserver,
    part_evaluation::{CapturedPartCandidate, CompletedPartEvaluation},
};

/// Opaque body capture; restoring it requires a matching committed candidate.
pub struct CapturedNamedSolid {
    document: DocumentGuid,
    body: CapturedBody,
}
enum CapturedBody {
    Prism(CapturedPrism),
    Fillet(blend::CapturedBlendTopology),
    Chamfer(blend::CapturedBlendTopology),
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
            Self::Fillet(body) => CapturedBody::Fillet(body.capture_detached(history, document)?),
            Self::Chamfer(body) => CapturedBody::Chamfer(body.capture_detached(history, document)?),
        };
        Ok(CapturedNamedSolid { document, body })
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
            CapturedBody::Fillet(body) => Ok(NamedSolid::Fillet(Box::new(
                body.restore(history, document)?,
            ))),
            CapturedBody::Chamfer(body) => Ok(NamedSolid::Chamfer(Box::new(
                body.restore(history, document)?,
            ))),
        }
    }
}
/// Complete exact named bodies after the owning session accepts their candidate.
/// Example: `complete_named_candidate(captured, &transition, &session, document)?`.
pub fn complete_named_candidate<O: TransactionObserver, Failure>(
    candidate: CapturedPartCandidate<CapturedNamedSolid, Failure>,
    transition: &EvaluationTransition,
    session: &DocumentSession<O>,
    document: DocumentGuid,
) -> Result<CompletedPartEvaluation<NamedSolid, Failure>, String> {
    candidate.try_complete(transition, session, |_, body, history| {
        body.restore(history, document)
    })
}
