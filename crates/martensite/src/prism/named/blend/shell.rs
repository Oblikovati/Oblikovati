// SPDX-License-Identifier: GPL-2.0-only
//! Exact inward shell candidates with named opening faces (#4106).
use super::*;
use martensite_tkernel::{message::ProgressRange, ncollection::List};
use martensite_tkoffset::{
    brep_offset::MakeOffsetOptions, brep_offset_api::BRepOffsetAPIMakeThickSolid,
};
use std::collections::BTreeSet;

impl NamedPrismSnapshot {
    /// Build a validated uniform inward shell; example: `body.shell_candidate(&history, document, &faces, 0.1)?`.
    pub fn shell_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        openings: &[Vec<u8>],
        thickness: f64,
    ) -> Result<BlendCandidate, String> {
        let source = self.blend_sources(history, document)?;
        let body = self.body_reference_key(history, document)?.to_vec();
        build_shell_candidate(
            BlendInputTopology {
                body: &self.snapshot.shape,
                source,
                source_bodies: vec![body.clone()],
                body_parent: body.clone(),
                primary_body: body,
                inherited_deleted: Vec::new(),
                resolution: self.certification.as_ref().map(|value| value.resolution),
            },
            history,
            document,
            openings,
            thickness,
        )
    }
}

pub(super) fn build_shell_candidate(
    input: BlendInputTopology<'_>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    openings: &[Vec<u8>],
    thickness: f64,
) -> Result<BlendCandidate, String> {
    validate_shell_inputs(&input.source, openings, thickness)?;
    let resolution = input
        .resolution
        .ok_or("shell input: expected a certified source resolution")?
        .validate()?;
    let mut copier = crate::checked_shape::copy_shape(input.body, "shell input")
        .map_err(|cause| cause.to_string())?;
    let sources = copy_named_sources(&mut copier, input.source)?;
    let body = copier.shape().map_err(|cause| cause.to_string())?.clone();
    let faces = opening_faces(&sources, openings)?;
    let mut maker = BRepOffsetAPIMakeThickSolid::new();
    maker
        .make_thick_solid_by_join(
            &body,
            &faces,
            -thickness,
            resolution.sew,
            MakeOffsetOptions::default(),
            &ProgressRange::default(),
        )
        .map_err(|cause| {
            format!("shell thickness {thickness}: expected a completed shell: {cause}")
        })?;
    let (shape, records) = finish_blend(&mut maker, &sources, "shell", thickness)?;
    let achieved = super::tolerances::check(&body, &shape, Some(resolution), "shell")?;
    Ok(BlendCandidate {
        resolution: Some(resolution),
        achieved_tolerance: Some(achieved),
        shape,
        history: records,
        source_bodies: input.source_bodies,
        body_parent: input.body_parent,
        primary_body: input.primary_body,
        inherited_deleted: input.inherited_deleted,
        document,
        scope: history.evaluation_scope(),
    })
}
fn validate_shell_inputs(
    sources: &[(Vec<u8>, Shape)],
    openings: &[Vec<u8>],
    thickness: f64,
) -> Result<(), String> {
    let unique: BTreeSet<_> = openings.iter().collect();
    if openings.is_empty()
        || unique.len() != openings.len()
        || !thickness.is_finite()
        || thickness <= 0.
    {
        return Err(format!(
            "shell openings {}, unique {}, thickness {thickness}: expected unique nonempty openings and finite positive thickness",
            openings.len(),
            unique.len()
        ));
    }
    opening_faces(sources, openings).map(|_| ())
}
fn opening_faces(
    sources: &[(Vec<u8>, Shape)],
    openings: &[Vec<u8>],
) -> Result<List<Shape>, String> {
    let mut faces = List::default();
    for key in openings {
        let matches: Vec<_> = sources
            .iter()
            .filter(|(name, _)| name == key)
            .map(|(_, shape)| shape)
            .collect();
        let [shape] = matches.as_slice() else {
            return Err(format!(
                "shell opening {key:?}, matches {}: expected exactly one named face in the source body",
                matches.len()
            ));
        };
        martensite_tkbrep::topods::face(shape)
            .map_err(|cause| format!("shell opening {key:?}: expected a face: {cause}"))?;
        faces.append((*shape).clone());
    }
    Ok(faces)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn ambiguous_opening_names_are_refused_before_construction() {
        let key = vec![1, 2, 3];
        let sources = vec![
            (key.clone(), Shape::default()),
            (key.clone(), Shape::default()),
        ];
        let error = opening_faces(&sources, &[key]).err().unwrap();
        assert!(error.contains("matches 2"), "{error}");
    }
}
