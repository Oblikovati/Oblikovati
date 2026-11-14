// SPDX-License-Identifier: GPL-2.0-only
//! Migrate only Go cap keys whose current Rust binding can be verified.
use super::{LegacyExtrusionFaceRole, LegacyPartCandidate, PackageCodecError, failure};
use oblikovati_model::{
    feature::{
        AuthoredFeature,
        part::PartFeatureDefinition,
        work_plane::{
            WorkPlaneDefinition, WorkPlaneDistance, WorkPlaneFaceReference, WorkPlaneReference,
        },
    },
    identity::{DocumentGuid, SketchPrismReferenceRole, sketch_prism_reference_name},
};

/// Check the proposed name against the current named body and its expected role.
pub trait LegacyCapBindingVerifier {
    /// Verify one name and role in the current result; example: `verifier.verify(&key, role)?`.
    fn verify(&self, key: &[u8], role: LegacyExtrusionFaceRole) -> Result<(), String>;
}

impl LegacyPartCandidate {
    /// Migrate a verified Go cap name into the matching authored prism.
    /// Example: `candidate.migrate_cap_face(&key, document, feature, verifier)?`.
    pub fn migrate_cap_face(
        &self,
        legacy_key: &[u8],
        document: DocumentGuid,
        target: &AuthoredFeature<PartFeatureDefinition>,
        verifier: &dyn LegacyCapBindingVerifier,
    ) -> Result<WorkPlaneFaceReference, PackageCodecError> {
        let producer = self.extrusion_face_producer(legacy_key)?;
        let role = cap_role(producer.role)?;
        let source = self.unique_extrusion_sequence(producer.creation_sequence)?;
        let definition = matching_prism(target, source)?;
        let key = sketch_prism_reference_name(document, target.id(), definition.sketch(), role)
            .map_err(failure)?;
        verifier.verify(&key, producer.role).map_err(failure)?;
        WorkPlaneFaceReference::new(key).map_err(failure)
    }
    /// Convert one saved cap-offset plane without changing the source package.
    /// Example: `candidate.migrate_cap_offset(sequence, document, feature, verifier)?`.
    pub fn migrate_cap_offset(
        &self,
        sequence: u64,
        document: DocumentGuid,
        target: &AuthoredFeature<PartFeatureDefinition>,
        verifier: &dyn LegacyCapBindingVerifier,
    ) -> Result<WorkPlaneDefinition, PackageCodecError> {
        let plane = self.unique_cap_offset(sequence)?;
        let producer = self.extrusion_face_producer(&plane.face_key)?;
        if producer.creation_sequence >= sequence || !plane.offset.is_finite() {
            return Err(failure(format!(
                "plane sequence {sequence}, producer {}, offset {}: expected an earlier producer and a finite offset",
                producer.creation_sequence, plane.offset
            )));
        }
        let face = self.migrate_cap_face(&plane.face_key, document, target, verifier)?;
        Ok(WorkPlaneDefinition::Offset {
            base: WorkPlaneReference::GoPrismFace(face),
            distance: WorkPlaneDistance::Constant(plane.offset),
        })
    }
    fn unique_cap_offset(
        &self,
        sequence: u64,
    ) -> Result<&super::LegacyFaceOffsetCandidate, PackageCodecError> {
        let mut matches = self
            .face_offsets
            .iter()
            .filter(|plane| plane.creation_sequence == sequence);
        let plane = matches.next().ok_or_else(|| {
            failure(format!(
                "plane sequence {sequence}: expected one face-offset plane"
            ))
        })?;
        if sequence == 0 || matches.next().is_some() {
            return Err(failure(format!(
                "plane sequence {sequence}: expected a unique nonzero face-offset plane"
            )));
        }
        Ok(plane)
    }
    fn unique_extrusion_sequence(
        &self,
        sequence: u64,
    ) -> Result<&super::LegacyExtrusionCandidate, PackageCodecError> {
        let mut matches = self
            .extrusions
            .iter()
            .filter(|item| item.creation_sequence == sequence);
        let source = matches.next().ok_or_else(|| {
            failure(format!(
                "producer {sequence}: expected one source extrusion"
            ))
        })?;
        if sequence == 0 || matches.next().is_some() {
            return Err(failure(format!(
                "producer {sequence}: expected one nonzero source sequence"
            )));
        }
        Ok(source)
    }
}

fn cap_role(role: LegacyExtrusionFaceRole) -> Result<SketchPrismReferenceRole, PackageCodecError> {
    match role {
        LegacyExtrusionFaceRole::StartCap => Ok(SketchPrismReferenceRole::StartCap),
        LegacyExtrusionFaceRole::EndCap => Ok(SketchPrismReferenceRole::EndCap),
        LegacyExtrusionFaceRole::Side(index) => Err(failure(format!(
            "Go side ordinal {index}: expected verified source-curve lineage before migration"
        ))),
    }
}

fn matching_prism<'a>(
    target: &'a AuthoredFeature<PartFeatureDefinition>,
    source: &super::LegacyExtrusionCandidate,
) -> Result<&'a oblikovati_model::feature::sketch_prism::SketchPrismDefinition, PackageCodecError> {
    let PartFeatureDefinition::SketchPrism(definition) = target.definition() else {
        return Err(failure(format!(
            "target {:?}: expected a sketch prism",
            target.id()
        )));
    };
    if target.name() != source.name || !definition.same_recipe(&source.definition) {
        return Err(failure(format!(
            "target {:?}: expected the source extrusion recipe and name {:?}",
            target.id(),
            source.name
        )));
    }
    Ok(definition)
}
