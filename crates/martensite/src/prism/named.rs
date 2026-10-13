// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    feature::FeatureId,
    history::{DocumentEditHistory, EvaluationScope},
    identity::{DocumentGuid, ExactReferenceBindings, SemanticReferenceName},
};
use rustycad_tkbrep::topods::Shape;

/// Retained exact geometry and current semantic names; native handles remain private.
pub struct NamedPrismSnapshot {
    snapshot: PrismProbeSnapshot,
    bindings: ExactReferenceBindings<Option<PrismFaceRole>>,
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
        .evaluate_part_features(&mut NamedPrismEvaluator::new(history, document))
        .map_err(|cause| cause.to_string())?;
    let decline = crate::stored_prism::stored_body_decline(&evaluation, feature);
    evaluation.into_body(feature).ok_or(decline)
}
mod evaluate;
pub use evaluate::NamedPrismEvaluator;
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
    Ok(NamedPrismSnapshot { snapshot, bindings })
}
fn prism_key(
    document: DocumentGuid,
    feature: FeatureId,
    sketch: u64,
    role: Option<PrismFaceRole>,
) -> Result<Vec<u8>, String> {
    let (kind, tag, sources) = match role {
        None => (4, "body", vec![sketch]),
        Some(PrismFaceRole::Start) => (1, "start", vec![sketch]),
        Some(PrismFaceRole::End) => (1, "end", vec![sketch]),
        Some(PrismFaceRole::ProfileEdge(edge)) => (1, "side", vec![sketch, edge]),
    };
    SemanticReferenceName::new(document, feature.0, kind, "sketch-prism", tag, &sources)?.encode()
}
impl NamedPrismSnapshot {
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
