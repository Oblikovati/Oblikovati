// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use martensite_tkbrep::topods::Shape;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::FeatureId,
    history::{DocumentEditHistory, EvaluationScope},
    identity::{
        DocumentGuid, ExactReferenceBindings, SketchPrismReferenceRole, sketch_prism_reference_name,
    },
};

/// Retained exact geometry and current semantic names; native handles remain private.
#[derive(Clone)]
pub struct NamedPrismSnapshot {
    snapshot: PrismProbeSnapshot,
    bindings: ExactReferenceBindings<Option<PrismFaceRole>>,
    subentity_bindings: Option<ExactReferenceBindings<subentities::BoundSubentity>>,
    certification: Option<CertifiedPrismEvidence>,
}
impl NamedPrismSnapshot {
    fn capture_current(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<Self, String> {
        Ok(Self {
            snapshot: self.snapshot.clone(),
            bindings: self.bindings.clone_current(history, document)?,
            subentity_bindings: self
                .subentity_bindings
                .as_ref()
                .map(|bindings| bindings.clone_current(history, document))
                .transpose()?,
            certification: self.certification.clone(),
        })
    }
    pub(crate) fn into_probe_snapshot(self) -> PrismProbeSnapshot {
        self.snapshot
    }
}
/// Measured tolerance and authored point identities retained with a certified replay.
#[derive(Clone, Debug, PartialEq)]
pub struct CertifiedPrismEvidence {
    pub resolution: PrismResolution,
    pub achieved_tolerance: PrismAchievedTolerance,
    pub subentities: PrismSubentityHistory,
    pub vertex_sources: std::collections::BTreeMap<u64, Vec<u64>>,
}
/// Evaluate an authored prism and certify its body/face names against construction history.
/// Example: `retain_named_sketch_prism(&history, document_guid, FeatureId(1))?`.
pub fn retain_named_sketch_prism(
    history: &DocumentEditHistory,
    document: DocumentGuid,
    feature: FeatureId,
) -> Result<NamedPrismSnapshot, String> {
    let job = history.evaluation_snapshot();
    let evaluation = job
        .snapshot()
        .evaluate_part_features(&mut CertifiedNamedSolidEvaluator::with_optional_policy(
            history, document, None,
        ))
        .map_err(|cause| cause.to_string())?;
    require_diagnostic_blend_certainty(job.snapshot(), &evaluation, feature)?;
    current_prism_result(evaluation, feature)
}

fn require_diagnostic_blend_certainty(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    evaluation: &oblikovati_model::feature::part::PartFeatureEvaluation<
        NamedSolid,
        EvaluationFailure,
    >,
    source: FeatureId,
) -> Result<(), String> {
    use oblikovati_model::feature::FeatureEvaluationStatus;
    for record in evaluation.health() {
        if matches!(
            record.status,
            FeatureEvaluationStatus::Healthy | FeatureEvaluationStatus::Suppressed
        ) {
            continue;
        }
        if diagnostic_blend_source(snapshot, record.feature).map_err(|error| error.to_string())?
            == Some(source)
        {
            return Err(format!(
                "prism {source:?}, blend {:?}: expected a certified policy to establish the current body",
                record.feature
            ));
        }
    }
    Ok(())
}

fn diagnostic_blend_source(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    feature: FeatureId,
) -> Result<Option<FeatureId>, oblikovati_model::feature::FeatureSequenceError> {
    use oblikovati_model::feature::part::PartFeatureDefinition;
    let authored = snapshot.work_geometry.get(feature)?;
    Ok(match authored.definition() {
        PartFeatureDefinition::Fillet(blend) => Some(blend.body_source()),
        PartFeatureDefinition::Chamfer(blend) => Some(blend.body_source()),
        _ => None,
    })
}

fn current_prism_result(
    evaluation: oblikovati_model::feature::part::PartFeatureEvaluation<
        NamedSolid,
        EvaluationFailure,
    >,
    feature: FeatureId,
) -> Result<NamedPrismSnapshot, String> {
    if evaluation.body(feature).is_none() && evaluation.body_from_source(feature).is_ok() {
        return Err(format!(
            "prism {feature:?}: expected a current prism body; a later feature replaced it"
        ));
    }
    let decline = crate::stored_prism::stored_body_decline(&evaluation, feature);
    match evaluation.into_body(feature) {
        Some(NamedSolid::Prism(prism)) => Ok(*prism),
        Some(_) => Err(format!(
            "feature {feature:?}: expected a current prism body"
        )),
        None => Err(decline),
    }
}
mod blend;
mod bounds;
mod edge_pick;
mod evaluate;
mod face_mesh;
mod face_pick;
pub use face_mesh::NamedFaceMesh;
mod captured;
mod face_topology;
mod solid;
pub use captured::{CapturedNamedSolid, complete_named_candidate, complete_named_checkpoints};
pub use edge_pick::NamedEdgeHit;
mod subentities;
pub use blend::{BlendCandidate, BlendHistoryRecord, BlendNamedTopology, RigidMotion};
pub use evaluate::retain_certified_named_sketch_prism;
use evaluate::{CertifiedNamedPrismEvaluator, NamedPrismEvaluator};
pub use face_topology::{PrismFaceTopology, PrismSurfaceKind};
pub use solid::{CertifiedNamedSolidEvaluator, NamedSolid, named_solid_part_properties};
fn attach_names(
    snapshot: PrismProbeSnapshot,
    document: DocumentGuid,
    feature: FeatureId,
    sketch: u64,
    scope: EvaluationScope,
) -> Result<NamedPrismSnapshot, String> {
    let mut entries = vec![(prism_key(document, feature, sketch, None)?, None)];
    for (role, _) in &snapshot.faces {
        entries.push((
            prism_key(document, feature, sketch, Some(*role))?,
            Some(*role),
        ));
    }
    let bindings = ExactReferenceBindings::from_entries(document, scope, entries)?;
    Ok(NamedPrismSnapshot {
        snapshot,
        bindings,
        subentity_bindings: None,
        certification: None,
    })
}
fn prism_key(
    document: DocumentGuid,
    feature: FeatureId,
    sketch: u64,
    role: Option<PrismFaceRole>,
) -> Result<Vec<u8>, String> {
    let role = match role {
        None => SketchPrismReferenceRole::Body,
        Some(PrismFaceRole::Start) => SketchPrismReferenceRole::StartCap,
        Some(PrismFaceRole::End) => SketchPrismReferenceRole::EndCap,
        Some(PrismFaceRole::ProfileEdge(edge)) => SketchPrismReferenceRole::Side(edge),
    };
    sketch_prism_reference_name(document, feature, DocumentSketchId(sketch), role)
}
impl NamedPrismSnapshot {
    /// Read the kernel evidence if this body used certified replay; example: `named.certification()`.
    pub fn certification(&self) -> Option<&CertifiedPrismEvidence> {
        self.certification.as_ref()
    }
    /// The nearest face a forward ray meets; example: `body.face_on_line(window_line)?`.
    pub fn face_on_line(
        &self,
        line: [[f64; 3]; 2],
    ) -> Result<Option<(PrismFaceRole, f64)>, String> {
        self.snapshot.face_on_line(line)
    }
    /// Resolve a picked construction role to one durable face name.
    /// Example: `body.face_key_for_role(history, document, role)?` after an exact face pick.
    pub fn face_key_for_role(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        role: PrismFaceRole,
    ) -> Result<Vec<u8>, String> {
        let mut found = None;
        for key in self.bindings.encoded_keys(history, document)? {
            let envelope = oblikovati_model::identity::ReferenceEnvelope::decode(key)
                .map_err(|error| error.to_string())?;
            if envelope.kind != 1 || *self.bindings.bind(history, document, key, 1)? != Some(role) {
                continue;
            }
            if found.replace(key.to_vec()).is_some() {
                return Err(format!(
                    "face role {role:?}: expected exactly one semantic name"
                ));
            }
        }
        found.ok_or_else(|| format!("face role {role:?}: expected exactly one semantic name"))
    }
    /// Borrow the sole current body name; example: `named.body_reference_key(history, guid)?` for API enumeration.
    pub fn body_reference_key<'a>(
        &'a self,
        history: &'a DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<&'a [u8], String> {
        let mut body_keys = self
            .bindings
            .encoded_keys(history, document)?
            .filter(|key| {
                oblikovati_model::identity::ReferenceEnvelope::decode(key)
                    .is_ok_and(|envelope| envelope.kind == 4)
            });
        let key = body_keys
            .next()
            .ok_or_else(|| "prism body names 0: expected exactly one".to_string())?;
        if body_keys.next().is_some() {
            return Err("prism body names greater than 1: expected exactly one".into());
        }
        self.bindings.bind(history, document, key, 4)?;
        Ok(key)
    }
    /// Integrate a verified current body without a mesh; example: `named.physical_properties(history, guid, key)?`.
    pub fn physical_properties(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<crate::AnalyticBodyProperties, String> {
        self.bindings.bind(history, document, key, 4)?;
        self.snapshot.physical_properties()
    }
    /// Read current body properties and face reports; example: `named.body_report(&history, doc, &body_key)?`.
    pub fn body_report<'a>(
        &'a self,
        history: &'a DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<&'a PrismProbeReport, String> {
        self.bindings.bind(history, document, key, 4)?;
        Ok(self.snapshot.report())
    }
    /// Integrate the exact bound face, not a display triangle; example: `named.face_report(&history, doc, &face_key)?`.
    pub fn face_report(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<PrismFaceProbe, String> {
        let role = self
            .bindings
            .bind(history, document, key, 1)?
            .ok_or_else(|| "face reference: expected a face construction role".to_string())?;
        report::face_report(role, unique_face(&self.snapshot, role)?)
            .map_err(|cause| cause.to_string())
    }
    /// Enumerate canonical names in stable byte order; example: `named.reference_keys(&history, doc)?`.
    pub fn reference_keys(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<Vec<Vec<u8>>, String> {
        Ok(self
            .bindings
            .encoded_keys(history, document)?
            .map(<[u8]>::to_vec)
            .collect())
    }
    fn verify_written_names(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        sketch: u64,
    ) -> Result<(), String> {
        let key = prism_key(document, feature, sketch, None)?;
        self.body_report(history, document, &key)?;
        for (role, expected) in &self.snapshot.faces {
            let key = prism_key(document, feature, sketch, Some(*role))?;
            let bound = self.bindings.bind(history, document, &key, 1)?;
            if *bound != Some(*role) || !unique_face(&self.snapshot, *role)?.is_same(expected) {
                return Err(format!(
                    "prism face {role:?}: expected exact write-time reference round trip"
                ));
            }
        }
        Ok(())
    }
}
fn unique_face(snapshot: &PrismProbeSnapshot, role: PrismFaceRole) -> Result<&Shape, String> {
    let mut matches = snapshot
        .faces
        .iter()
        .filter(|(candidate, _)| *candidate == role);
    let first = matches.next();
    if first.is_none() || matches.next().is_some() {
        return Err(format!(
            "prism face {role:?}: expected exactly one retained construction role"
        ));
    }
    Ok(&first.expect("one verified role").1)
}

impl super::PrismPresentationSource for NamedPrismSnapshot {
    fn presentation(&self, options: PrismMeshOptions) -> Result<PrismPresentation, String> {
        self.snapshot.presentation(options)
    }
}

/// Integrate current named bodies as one unit-density system in caller order.
/// Example: `named_part_properties(history, guid, evaluation.ordered_bodies())?`.
pub fn named_part_properties<'a>(
    history: &DocumentEditHistory,
    document: DocumentGuid,
    bodies: impl IntoIterator<Item = &'a NamedPrismSnapshot>,
) -> Result<Option<crate::AnalyticBodyProperties>, String> {
    let mut shapes = Vec::new();
    for body in bodies {
        body.body_reference_key(history, document)?;
        shapes.push(&body.snapshot.shape);
    }
    crate::analytic_body::integrate_system(shapes)
}

#[path = "named/base_view.rs"]
mod base_view;
#[path = "named/discretize.rs"]
mod discretize;
#[path = "named/projection.rs"]
mod projection;
pub use base_view::{DrawingBaseProjection, drawing_base_view_bodies, drawing_base_view_solids};
pub use projection::{
    DrawingProjectedCurve, DrawingProjection, DrawingProjectionCurveKind, DrawingProjectionFrame,
    drawing_projection_bodies, drawing_projection_solids,
};
