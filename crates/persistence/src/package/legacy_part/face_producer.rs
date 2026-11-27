// SPDX-License-Identifier: GPL-2.0-only
//! Identify source extrusion roles without assuming Go and Rust face order agree.
use super::{LegacyPartCandidate, PackageCodecError, failure};

/// A role in the Go extrusion's original topological lineage.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum LegacyExtrusionFaceRole {
    StartCap,
    EndCap,
    Side(u64),
}

/// Source identity and role, prior to mapping the name to a Rust face.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub struct LegacyExtrusionFaceProducer {
    pub creation_sequence: u64,
    pub role: LegacyExtrusionFaceRole,
}

impl LegacyPartCandidate {
    pub(super) fn validate_face_producers(&self) -> Result<(), PackageCodecError> {
        for plane in &self.face_offsets {
            let producer = self.extrusion_face_producer(&plane.face_key)?;
            if producer.creation_sequence >= plane.creation_sequence {
                return Err(failure(format!(
                    "face producer sequence {} for plane {}: expected an earlier producer",
                    producer.creation_sequence, plane.creation_sequence
                )));
            }
        }
        Ok(())
    }

    /// Identify one unambiguous source extrusion; this does not resolve a Rust face.
    /// Example: `let producer = candidate.extrusion_face_producer(&key)?;`.
    pub fn extrusion_face_producer(
        &self,
        key: &[u8],
    ) -> Result<LegacyExtrusionFaceProducer, PackageCodecError> {
        let lineage = face_lineage(key)?;
        let mut producers = self.extrusions.iter().filter_map(|feature| {
            let role = extrusion_role(lineage, &feature.name)?;
            Some(LegacyExtrusionFaceProducer {
                creation_sequence: feature.creation_sequence,
                role,
            })
        });
        let first = producers.next().ok_or_else(|| {
            failure(format!(
                "face key {key:?}: expected one supported extrusion face producer"
            ))
        })?;
        if producers.next().is_some() {
            return Err(failure(format!(
                "face key {key:?}: expected an unambiguous producer"
            )));
        }
        Ok(first)
    }
}

fn face_lineage(key: &[u8]) -> Result<&str, PackageCodecError> {
    let Some((&3, lineage)) = key.split_first() else {
        return Err(failure(format!(
            "face key {key:?}: expected Go face kind 3"
        )));
    };
    std::str::from_utf8(lineage)
        .map_err(|cause| failure(format!("face key {key:?}: expected UTF-8 lineage: {cause}")))
}

fn extrusion_role(lineage: &str, name: &str) -> Option<LegacyExtrusionFaceRole> {
    let suffix = lineage.strip_prefix(name)?.strip_prefix(':')?;
    match suffix {
        "start-cap#0" => Some(LegacyExtrusionFaceRole::StartCap),
        "end-cap#0" => Some(LegacyExtrusionFaceRole::EndCap),
        _ => side_role(suffix),
    }
}

fn side_role(suffix: &str) -> Option<LegacyExtrusionFaceRole> {
    let ordinal = suffix.strip_prefix("side#")?;
    let value = ordinal.parse::<u64>().ok()?;
    if value > i64::MAX as u64 || value.to_string() != ordinal {
        return None;
    }
    Some(LegacyExtrusionFaceRole::Side(value))
}
