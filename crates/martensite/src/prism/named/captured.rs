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

/// Capture a fresh full replay for an exact candidate before its source history changes (#4122).
/// Example: `capture_full_named_candidate(id, history, &candidate, guid, policy)?`.
pub fn capture_full_named_candidate(
    document_id: u64,
    history: &DocumentEditHistory,
    candidate: &oblikovati_model::history::DocumentSnapshot,
    document: DocumentGuid,
    policy: Option<PrismResolutionPolicy>,
) -> Result<
    CapturedPartTransfer<PartReplayCheckpoints<CapturedNamedSolid, EvaluationFailure>>,
    String,
> {
    let source = history.evaluation_scope();
    with_full_named_replay(candidate, document, policy, |replay, checkpoints| {
        let captured = checkpoints.try_map_bodies(|_, body| {
            let mut detached = body.capture_detached(replay, document)?;
            detached.source = source.clone();
            Ok::<_, String>(detached)
        })?;
        CapturedPartTransfer::capture_checkpoints(
            document_id,
            &source,
            candidate,
            captured,
            history,
        )
    })
}

type PreparedPrismParts = (
    oblikovati_model::history::PreparedValue<CapturedPrism>,
    (
        oblikovati_model::identity::PreparedExactReferenceBindings<Option<PrismFaceRole>>,
        Option<
            oblikovati_model::identity::PreparedExactReferenceBindings<subentities::BoundSubentity>,
        >,
    ),
);
type PreparedBlendParts = (
    oblikovati_model::history::PreparedValue<blend::CapturedBlendTopology>,
    oblikovati_model::identity::PreparedExactReferenceBindings<Shape>,
);
type PreparedSolidParts =
    oblikovati_model::history::PreparedChoice<PreparedPrismParts, PreparedBlendParts>;

impl CapturedPrism {
    fn prepare(
        mut self,
        candidate: &oblikovati_model::history::PreparedPrivateEdit<'_>,
        dependent: Option<&oblikovati_model::history::PreparedDependentRefresh<'_>>,
        document: DocumentGuid,
    ) -> Result<PreparedPrismParts, String> {
        let entries = std::mem::take(&mut self.entries);
        let subentities = self.subentities.take();
        let bindings = prepare_index(candidate, dependent, document, entries)?;
        let subentities = subentities
            .map(|entries| prepare_index(candidate, dependent, document, entries))
            .transpose()?;
        Ok((
            oblikovati_model::history::PreparedValue::new(candidate, self),
            (bindings, subentities),
        ))
    }
    fn assemble(
        self,
        bindings: ExactReferenceBindings<Option<PrismFaceRole>>,
        subentity_bindings: Option<ExactReferenceBindings<subentities::BoundSubentity>>,
    ) -> NamedPrismSnapshot {
        NamedPrismSnapshot {
            snapshot: self.snapshot,
            bindings,
            subentity_bindings,
            certification: self.certification,
        }
    }
}

pub(super) fn prepare_index<T>(
    candidate: &oblikovati_model::history::PreparedPrivateEdit<'_>,
    dependent: Option<&oblikovati_model::history::PreparedDependentRefresh<'_>>,
    document: DocumentGuid,
    entries: impl IntoIterator<Item = (Vec<u8>, T)>,
) -> Result<oblikovati_model::identity::PreparedExactReferenceBindings<T>, String> {
    match dependent {
        Some(refresh) => oblikovati_model::identity::PreparedExactReferenceBindings::for_dependent(
            refresh, document, entries,
        ),
        None => oblikovati_model::identity::PreparedExactReferenceBindings::new(
            candidate, document, entries,
        ),
    }
}

fn prepare_solid(
    body: &NamedSolid,
    replay: &DocumentEditHistory,
    candidate: &oblikovati_model::history::PreparedPrivateEdit<'_>,
    dependent: Option<&oblikovati_model::history::PreparedDependentRefresh<'_>>,
    document: DocumentGuid,
) -> Result<PreparedSolidParts, String> {
    let capture = body.capture_detached(replay, document)?;
    match capture.body {
        CapturedBody::Prism(prism) => prism
            .prepare(candidate, dependent, document)
            .map(oblikovati_model::history::PreparedChoice::First),
        CapturedBody::Derived(blend) => blend
            .prepare(candidate, dependent, document)
            .map(oblikovati_model::history::PreparedChoice::Second),
    }
}

fn assemble_solid(
    parts: <PreparedSolidParts as oblikovati_model::history::PreparedPublication>::Output,
) -> NamedSolid {
    use oblikovati_model::history::PreparedChoice;
    match parts {
        PreparedChoice::First((prism, (bindings, subentities))) => {
            NamedSolid::Prism(Box::new(prism.assemble(bindings, subentities)))
        }
        PreparedChoice::Second((blend, bindings)) => {
            NamedSolid::Derived(Box::new(blend.assemble(bindings)))
        }
    }
}

/// Prepare exact named bodies and all feature boundaries before a candidate is published (#4122).
/// Example: `prepare_full_named_publication(&edit, document, policy)?`.
pub fn prepare_full_named_publication(
    candidate: &oblikovati_model::history::PreparedPrivateEdit<'_>,
    document: DocumentGuid,
    policy: Option<PrismResolutionPolicy>,
) -> Result<
    impl oblikovati_model::history::PreparedPublication<
        Output = PartReplayCheckpoints<NamedSolid, EvaluationFailure>,
    > + use<>,
    String,
> {
    prepare_full_named_for_snapshot(candidate, candidate.snapshot(), None, document, policy)
}

/// Prepare a linked part replay under its own unchanged history (#4122).
/// Example: `prepare_dependent_named_publication(&edit, &dependent, document, policy)?`.
pub fn prepare_dependent_named_publication(
    candidate: &oblikovati_model::history::PreparedPrivateEdit<'_>,
    dependent: &oblikovati_model::history::PreparedDependentRefresh<'_>,
    document: DocumentGuid,
    policy: Option<PrismResolutionPolicy>,
) -> Result<
    impl oblikovati_model::history::PreparedPublication<
        Output = PartReplayCheckpoints<NamedSolid, EvaluationFailure>,
    > + use<>,
    String,
> {
    prepare_full_named_for_snapshot(
        candidate,
        dependent.snapshot(),
        Some(dependent),
        document,
        policy,
    )
}

fn prepare_full_named_for_snapshot(
    candidate: &oblikovati_model::history::PreparedPrivateEdit<'_>,
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    dependent: Option<&oblikovati_model::history::PreparedDependentRefresh<'_>>,
    document: DocumentGuid,
    policy: Option<PrismResolutionPolicy>,
) -> Result<
    impl oblikovati_model::history::PreparedPublication<
        Output = PartReplayCheckpoints<NamedSolid, EvaluationFailure>,
    > + use<>,
    String,
> {
    let prepared = with_full_named_replay(snapshot, document, policy, |replay, checkpoints| {
        checkpoints
            .try_map_bodies(|_, body| prepare_solid(body, replay, candidate, dependent, document))
    })?;
    Ok(oblikovati_model::history::PreparedMapped::new(
        prepared,
        |parts: PartReplayCheckpoints<
            <PreparedSolidParts as oblikovati_model::history::PreparedPublication>::Output,
            EvaluationFailure,
        >,
         _| {
            let result = parts.try_map_owned_bodies(|_, body| {
                Ok::<_, std::convert::Infallible>(assemble_solid(body))
            });
            match result {
                Ok(value) => value,
                Err(never) => match never {},
            }
        },
    ))
}

/// Validate a full named replay without changing the source history (#309).
/// Example: `validate_full_named_snapshot(&candidate, document, policy)?`.
pub fn validate_full_named_snapshot(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    document: DocumentGuid,
    policy: Option<PrismResolutionPolicy>,
) -> Result<(), String> {
    with_full_named_replay(snapshot, document, policy, |history, checkpoints| {
        checkpoints
            .try_map_bodies(|_, body| body.capture_detached(history, document))
            .map(|_| ())
    })
}

fn with_full_named_replay<T>(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    document: DocumentGuid,
    policy: Option<PrismResolutionPolicy>,
    capture: impl FnOnce(
        &DocumentEditHistory,
        &PartReplayCheckpoints<NamedSolid, EvaluationFailure>,
    ) -> Result<T, String>,
) -> Result<T, String> {
    use oblikovati_model::feature::FeatureEvaluationStatus;
    let replay = oblikovati_model::history::SnapshotReplay::new(snapshot)
        .map_err(|error| error.to_string())?;
    let mut evaluator =
        CertifiedNamedSolidEvaluator::with_optional_policy(replay.history(), document, policy);
    let checkpoints = replay
        .history()
        .snapshot()
        .evaluate_part_with_checkpoints(&mut evaluator)
        .map_err(|error| error.to_string())?;
    for record in checkpoints.result().health() {
        if !matches!(
            record.status,
            FeatureEvaluationStatus::Healthy | FeatureEvaluationStatus::Suppressed
        ) {
            return Err(format!(
                "model-state candidate feature {:?} health {:?}: expected a healthy or suppressed replay",
                record.feature, record.status
            ));
        }
    }
    capture(replay.history(), &checkpoints)
}
