// SPDX-License-Identifier: GPL-2.0-only
//! Bind a legacy side only when independently witnessed source lineage exists.
use super::{
    LegacyExtrusionCandidate, LegacyExtrusionFaceRole, LegacyPartCandidate, PackageCodecError,
    cap_binding::{legacy_offset_definition, matching_prism},
    failure,
};
use oblikovati_model::{
    feature::{
        AuthoredFeature,
        part::PartFeatureDefinition,
        work_plane::{WorkPlaneDefinition, WorkPlaneFaceReference},
    },
    identity::{DocumentGuid, SketchPrismReferenceRole, sketch_prism_reference_name},
    sketch::SketchEntityId,
};

/// Independent Go source lineage and current named-body verification.
pub trait LegacySideLineageWitness {
    /// Certify the Go side ordinal's source curve; refuse absent or ambiguous evidence.
    /// Example: `witness.source_curve(&key, 1, source)?`.
    fn source_curve(
        &self,
        key: &[u8],
        ordinal: u64,
        source: &LegacyExtrusionCandidate,
    ) -> Result<SketchEntityId, String>;
    /// Verify the target name resolves to the same source curve in the current body.
    /// Example: `witness.verify_target(&key, curve)?`.
    fn verify_target(&self, key: &[u8], curve: SketchEntityId) -> Result<(), String>;
}

impl LegacyPartCandidate {
    /// Convert one saved side-offset plane after verifying both source and target lineage.
    /// Example: `candidate.migrate_side_offset(sequence, document, feature, witness)?`.
    pub fn migrate_side_offset(
        &self,
        sequence: u64,
        document: DocumentGuid,
        target: &AuthoredFeature<PartFeatureDefinition>,
        witness: &dyn LegacySideLineageWitness,
    ) -> Result<WorkPlaneDefinition, PackageCodecError> {
        let plane = self.validated_face_offset(sequence)?;
        let face = self.migrate_side_face(&plane.face_key, document, target, witness)?;
        Ok(legacy_offset_definition(face, plane.offset))
    }
    /// Migrate a Go side only with independent source lineage and a current verified target.
    /// Example: `candidate.migrate_side_face(&key, document, feature, witness)?`.
    pub fn migrate_side_face(
        &self,
        legacy_key: &[u8],
        document: DocumentGuid,
        target: &AuthoredFeature<PartFeatureDefinition>,
        witness: &dyn LegacySideLineageWitness,
    ) -> Result<WorkPlaneFaceReference, PackageCodecError> {
        let producer = self.extrusion_face_producer(legacy_key)?;
        let LegacyExtrusionFaceRole::Side(ordinal) = producer.role else {
            return Err(failure(format!(
                "face role {:?}: expected a Go extrusion side",
                producer.role
            )));
        };
        let source = self.unique_extrusion_sequence(producer.creation_sequence)?;
        let definition = matching_prism(target, source)?;
        let curve = witness
            .source_curve(legacy_key, ordinal, source)
            .map_err(failure)?;
        require_unique_boundary_curve(definition.selection(), curve)?;
        let key = sketch_prism_reference_name(
            document,
            target.id(),
            definition.sketch(),
            SketchPrismReferenceRole::Side(curve.0),
        )
        .map_err(failure)?;
        witness.verify_target(&key, curve).map_err(failure)?;
        WorkPlaneFaceReference::new(key).map_err(failure)
    }
}

fn require_unique_boundary_curve(
    selection: &oblikovati_model::feature::sketch_prism::SketchPrismSelection,
    curve: SketchEntityId,
) -> Result<(), PackageCodecError> {
    let count = selection
        .outer
        .iter()
        .chain(selection.holes.iter().flatten())
        .filter(|item| item.curve == curve)
        .count();
    if count != 1 {
        return Err(failure(format!(
            "source curve {curve:?} occurs {count} times: expected one selected boundary use"
        )));
    }
    Ok(())
}
