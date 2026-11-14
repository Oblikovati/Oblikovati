// SPDX-License-Identifier: GPL-2.0-only
//! Stage ordered legacy prism and cap-plane definitions without mutating the source.
use super::{
    LegacyCapBindingVerifier, LegacyExtrusionFaceRole, LegacyPartCandidate, PackageCodecError,
};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{AuthoredFeature, FeatureId},
    history::DocumentSnapshot,
    identity::DocumentGuid,
};
use std::collections::{BTreeMap, BTreeSet};

/// Explicit target identity selected by the document's permanent allocator.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct LegacyStagedFeatureIdentity {
    pub id: FeatureId,
    pub name: String,
}

/// Solve the staged input and check a named cap in its current prefix.
pub trait LegacyCapStageVerifier {
    /// Solve current sketches without replacing authored input; example: `verifier.solve(&mut snapshot)?`.
    fn solve(&self, snapshot: &mut DocumentSnapshot) -> Result<(), String>;
    /// Qualify the current sketch's spatial and angular solve evidence; default is a named decline.
    /// Implementations own this trust boundary and must not infer bounds from raw residuals.
    /// Example: `verifier.qualify_sketch(snapshot, sketch)?` before creating a staged feature.
    fn qualify_sketch(
        &self,
        _snapshot: &DocumentSnapshot,
        sketch: DocumentSketchId,
    ) -> Result<(), String> {
        Err(format!(
            "sketch {sketch:?}: qualified spatial and angular solve evidence is unavailable (#3928)"
        ))
    }
    /// Verify an exact cap against the current prefix; example: `verifier.verify(snapshot, doc, id, key, role)?`.
    fn verify(
        &self,
        snapshot: &DocumentSnapshot,
        document: DocumentGuid,
        feature: FeatureId,
        key: &[u8],
        role: LegacyExtrusionFaceRole,
    ) -> Result<(), String>;
}

struct PrefixVerifier<'a> {
    owner: &'a dyn LegacyCapStageVerifier,
    snapshot: &'a DocumentSnapshot,
    document: DocumentGuid,
    feature: FeatureId,
}
impl LegacyCapBindingVerifier for PrefixVerifier<'_> {
    fn verify(&self, key: &[u8], role: LegacyExtrusionFaceRole) -> Result<(), String> {
        self.owner
            .verify(self.snapshot, self.document, self.feature, key, role)
    }
}

impl LegacyPartCandidate {
    /// Build an isolated ordered program with caller-reserved IDs and verified cap bindings.
    /// The verifier must qualify every source sketch; no production qualifier is supplied here.
    /// Example: `candidate.stage_cap_program(&base, document, &identities, verifier)?`.
    pub fn stage_cap_program(
        &self,
        base: &DocumentSnapshot,
        document: DocumentGuid,
        identities: &BTreeMap<u64, LegacyStagedFeatureIdentity>,
        verifier: &dyn LegacyCapStageVerifier,
    ) -> Result<DocumentSnapshot, PackageCodecError> {
        validate_base(self, base)?;
        let order = validate_identities(self, identities)?;
        let mut staged = base.clone();
        verifier.solve(&mut staged).map_err(failure)?;
        validate_solved_sources(base, &staged)?;
        for sequence in order {
            self.append_staged(
                &mut staged,
                document,
                sequence,
                &identities[&sequence],
                identities,
                verifier,
            )?;
        }
        Ok(staged)
    }

    fn append_staged(
        &self,
        snapshot: &mut DocumentSnapshot,
        document: DocumentGuid,
        sequence: u64,
        identity: &LegacyStagedFeatureIdentity,
        identities: &BTreeMap<u64, LegacyStagedFeatureIdentity>,
        verifier: &dyn LegacyCapStageVerifier,
    ) -> Result<(), PackageCodecError> {
        if let Some(extrusion) = self
            .extrusions
            .iter()
            .find(|row| row.creation_sequence == sequence)
        {
            if identity.name != extrusion.name || extrusion.suppressed {
                return Err(failure(format!(
                    "extrusion {sequence}: expected original name and an unsuppressed source"
                )));
            }
            let sketch = extrusion.definition.sketch();
            snapshot.require_resolved_sketch(sketch).map_err(failure)?;
            verifier.qualify_sketch(snapshot, sketch).map_err(failure)?;
            let feature = AuthoredFeature::sketch_prism(
                identity.id,
                &identity.name,
                extrusion.definition.clone(),
                snapshot,
            )
            .map_err(failure)?;
            return snapshot
                .work_geometry
                .append(feature)
                .map(|_| ())
                .map_err(failure);
        }
        self.append_plane(snapshot, document, sequence, identity, identities, verifier)
    }

    fn append_plane(
        &self,
        snapshot: &mut DocumentSnapshot,
        document: DocumentGuid,
        sequence: u64,
        identity: &LegacyStagedFeatureIdentity,
        identities: &BTreeMap<u64, LegacyStagedFeatureIdentity>,
        verifier: &dyn LegacyCapStageVerifier,
    ) -> Result<(), PackageCodecError> {
        let plane = self
            .face_offsets
            .iter()
            .find(|row| row.creation_sequence == sequence)
            .ok_or_else(|| {
                failure(format!(
                    "sequence {sequence}: expected an extrusion or face-offset plane"
                ))
            })?;
        let producer = self.extrusion_face_producer(&plane.face_key)?;
        let target_id = identities
            .get(&producer.creation_sequence)
            .ok_or_else(|| {
                failure(format!(
                    "producer {}: expected an assigned identity",
                    producer.creation_sequence
                ))
            })?
            .id;
        let target = snapshot.work_geometry.get(target_id).map_err(failure)?;
        let adapter = PrefixVerifier {
            owner: verifier,
            snapshot,
            document,
            feature: target_id,
        };
        let definition = self.migrate_cap_offset(sequence, document, target, &adapter)?;
        let feature = AuthoredFeature::datum(identity.id, &identity.name, definition);
        snapshot
            .work_geometry
            .append(feature)
            .map(|_| ())
            .map_err(failure)
    }
}

fn validate_base(
    candidate: &LegacyPartCandidate,
    base: &DocumentSnapshot,
) -> Result<(), PackageCodecError> {
    if !base
        .work_geometry
        .same_work_geometry_recipe(&Default::default())
        || !base.sketches.same_recipe(&candidate.sketches.sketches)
        || !base
            .parameters
            .same_recipe(&candidate.sketches.parameters.parameters)
    {
        return Err(failure(
            "base snapshot: expected the source sketches and parameters with an empty feature program",
        ));
    }
    Ok(())
}

fn validate_solved_sources(
    base: &DocumentSnapshot,
    solved: &DocumentSnapshot,
) -> Result<(), PackageCodecError> {
    if !base.same_metadata_recipe(solved)
        || !base.parameters.same_recipe(&solved.parameters)
        || !base
            .work_geometry
            .same_work_geometry_recipe(&solved.work_geometry)
    {
        return Err(failure(
            "solver output: expected unchanged document metadata, parameters and feature program",
        ));
    }
    let before = base.sketches.recipe();
    let mut after = solved.sketches.recipe();
    oblikovati_model::document_sketch::DocumentSketches::from_recipe(&after).map_err(failure)?;
    if before.sketches.len() != after.sketches.len() {
        return Err(failure(
            "solver output: expected the original sketch identities",
        ));
    }
    for (original, current) in before.sketches.iter().zip(&mut after.sketches) {
        let before_ids: Vec<_> = original
            .geometry
            .points
            .iter()
            .map(|point| point.id)
            .collect();
        let after_ids: Vec<_> = current
            .geometry
            .points
            .iter()
            .map(|point| point.id)
            .collect();
        if before_ids != after_ids {
            return Err(failure(format!(
                "sketch {:?}: expected unchanged point identities",
                original.id
            )));
        }
        current.geometry.points = original.geometry.points.clone();
    }
    let restored = oblikovati_model::document_sketch::DocumentSketches::from_recipe(&after)
        .map_err(failure)?;
    if !base.sketches.same_recipe(&restored) {
        return Err(failure(
            "solver output: expected unchanged sketch sources apart from solved point positions",
        ));
    }
    Ok(())
}

fn validate_identities(
    candidate: &LegacyPartCandidate,
    identities: &BTreeMap<u64, LegacyStagedFeatureIdentity>,
) -> Result<Vec<u64>, PackageCodecError> {
    let mut sequences = BTreeSet::new();
    for sequence in candidate
        .extrusions
        .iter()
        .map(|row| row.creation_sequence)
        .chain(
            candidate
                .face_offsets
                .iter()
                .map(|row| row.creation_sequence),
        )
    {
        if sequence == 0 || !sequences.insert(sequence) {
            return Err(failure(format!(
                "source sequence {sequence}: expected one nonzero feature"
            )));
        }
    }
    if identities.keys().copied().collect::<BTreeSet<_>>() != sequences {
        return Err(failure(
            "feature assignments: expected exactly one entry for each source feature",
        ));
    }
    validate_target_ids(identities)?;
    validate_all_source_sequences(candidate, &sequences)?;
    validate_sketch_order(candidate)?;
    Ok(sequences.into_iter().collect())
}

fn validate_target_ids(
    identities: &BTreeMap<u64, LegacyStagedFeatureIdentity>,
) -> Result<(), PackageCodecError> {
    let mut ids = BTreeSet::new();
    for (sequence, identity) in identities {
        if identity.id.0 == 0 || identity.name.is_empty() || !ids.insert(identity.id) {
            return Err(failure(format!(
                "feature assignment {sequence}: expected a unique nonzero ID and a nonempty name"
            )));
        }
    }
    Ok(())
}

fn validate_all_source_sequences(
    candidate: &LegacyPartCandidate,
    features: &BTreeSet<u64>,
) -> Result<(), PackageCodecError> {
    let mut sequences = features.clone();
    let mut sketches = BTreeSet::new();
    for (id, sequence) in &candidate.sketches.creation_sequences {
        if *sequence == 0 || !sequences.insert(*sequence) || !sketches.insert(id.0) {
            return Err(failure(format!(
                "sketch {id:?}, sequence {sequence}: expected unique nonzero source identities"
            )));
        }
    }
    let live: BTreeSet<_> = candidate
        .sketches
        .sketches
        .sketches()
        .iter()
        .map(|sketch| sketch.id().0)
        .collect();
    if live != sketches {
        return Err(failure(
            "sketch sequences: expected one entry for each source sketch",
        ));
    }
    Ok(())
}

fn validate_sketch_order(candidate: &LegacyPartCandidate) -> Result<(), PackageCodecError> {
    for feature in &candidate.extrusions {
        let sketch = feature.definition.sketch();
        let mut rows = candidate
            .sketches
            .creation_sequences
            .iter()
            .filter(|(id, _)| *id == sketch);
        let Some((_, sequence)) = rows.next() else {
            return Err(failure(format!(
                "sketch {sketch:?}: expected one source sequence"
            )));
        };
        if *sequence == 0 || *sequence >= feature.creation_sequence || rows.next().is_some() {
            return Err(failure(format!(
                "sketch {sketch:?} sequence {sequence}: expected one earlier source sequence"
            )));
        }
    }
    Ok(())
}

fn failure(error: impl std::fmt::Display) -> PackageCodecError {
    PackageCodecError(format!(
        "legacy staged program: expected a complete verified source conversion: {error}"
    ))
}
