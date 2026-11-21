// SPDX-License-Identifier: GPL-2.0-only
//! Retain exact named face hits until selection priority is applied (#4045).
use super::*;

impl NamedSolid {
    /// Return all named forward-ray face hits in deterministic depth order.
    /// Example: `body.face_hits_on_line(&history, document, [point, direction])?`.
    pub fn face_hits_on_line(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        line: [[f64; 3]; 2],
    ) -> Result<Vec<(Vec<u8>, f64)>, String> {
        let keys = self.face_reference_keys(history, document)?;
        let faces = keys
            .iter()
            .map(|key| {
                self.native_face(history, document, key)
                    .map(|shape| (key.clone(), shape))
            })
            .collect::<Result<Vec<_>, _>>()?;
        face_hits(faces, line)
    }

    /// Resolve an unambiguous nearest face; example: `body.face_key_on_line(&history, document, ray)?`.
    pub fn face_key_on_line(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        line: [[f64; 3]; 2],
    ) -> Result<Option<(Vec<u8>, f64)>, String> {
        nearest_face(self.face_hits_on_line(history, document, line)?, line)
    }
}

impl BlendNamedTopology {
    /// Resolve an unambiguous nearest face; example: `body.face_key_on_line(&history, document, ray)?`.
    pub fn face_key_on_line(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        line: [[f64; 3]; 2],
    ) -> Result<Option<(Vec<u8>, f64)>, String> {
        let keys = self.keys_of_kinds(history, document, &[1])?;
        let faces = keys
            .iter()
            .map(|key| {
                self.native_face(history, document, key)
                    .map(|shape| (key.clone(), shape))
            })
            .collect::<Result<Vec<_>, _>>()?;
        nearest_face(face_hits(faces, line)?, line)
    }
}

fn face_hits(
    faces: Vec<(Vec<u8>, &Shape)>,
    [point, direction]: [[f64; 3]; 2],
) -> Result<Vec<(Vec<u8>, f64)>, String> {
    let line = super::snapshot::sight_line(point, direction)?;
    let mut hits = Vec::new();
    for (key, face) in faces {
        let Some(distance) = super::snapshot::nearest_face_distance(face, &line)
            .map_err(|cause| cause.to_string())?
        else {
            continue;
        };
        if !distance.is_finite() || distance < 0. {
            return Err(format!(
                "face ray distance {distance}: expected finite nonnegative distance"
            ));
        }
        hits.push((key, distance));
    }
    hits.sort_by(|left, right| left.1.total_cmp(&right.1).then(left.0.cmp(&right.0)));
    Ok(hits)
}

fn nearest_face(
    mut hits: Vec<(Vec<u8>, f64)>,
    line: [[f64; 3]; 2],
) -> Result<Option<(Vec<u8>, f64)>, String> {
    if hits.len() > 1 && hits[0].1 == hits[1].1 {
        return Err(format!(
            "face ray {line:?}, distance {}: expected one nearest face",
            hits[0].1
        ));
    }
    Ok(if hits.is_empty() {
        None
    } else {
        Some(hits.remove(0))
    })
}
