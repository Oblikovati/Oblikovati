// SPDX-License-Identifier: GPL-2.0-only
//! Detached face deletion with open-shell conversion or Martensite defeaturing heal (#3918).
use super::*;
use martensite_tkbo::brep_algo_api::BRepAlgoAPIDefeaturing;
use martensite_tkbrep::{
    brep::BRepBuilder,
    topexp,
    topods::{self, Shell},
    toptools::{IndexedMapOfShape, MapOfShape},
};
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tktopalgo::brep_check::BRepCheckAnalyzer;
use std::collections::BTreeSet;

/// Resolved inputs for detached Martensite Delete Face construction.
pub(in crate::prism::named) struct DeleteFaceRequest<'a> {
    pub faces: &'a [Vec<u8>],
    pub heal_remaining_faces: bool,
    pub select_lump_or_void: bool,
}

impl NamedPrismSnapshot {
    /// Construct a detached Delete Face candidate from named faces (#3918).
    pub(in crate::prism::named) fn delete_face_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        request: &DeleteFaceRequest<'_>,
    ) -> Result<BlendCandidate, String> {
        let source = self.blend_sources(history, document)?;
        let body = self.body_reference_key(history, document)?.to_vec();
        build_delete_face_candidate(
            BlendInputTopology {
                body: &self.snapshot.shape,
                source,
                source_bodies: vec![body.clone()],
                body_parent: body.clone(),
                primary_body: body,
                inherited_deleted: Vec::new(),
                inherited_splits: Default::default(),
                resolution: self.certification.as_ref().map(|value| value.resolution),
            },
            history,
            document,
            request,
        )
    }
}

pub(super) fn build_delete_face_candidate(
    input: BlendInputTopology<'_>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    request: &DeleteFaceRequest<'_>,
) -> Result<BlendCandidate, String> {
    let _ = request.select_lump_or_void;
    validate_delete_face_inputs(&input.source, request.faces)?;
    let mut copier = crate::checked_shape::copy_shape(input.body, "delete face input")
        .map_err(|cause| cause.to_string())?;
    let sources = copy_named_sources(&mut copier, input.source)?;
    let body = copier.shape().map_err(|cause| cause.to_string())?.clone();
    let targets = selected_face_shapes(&sources, request.faces)?;

    let (shape, records) = if request.heal_remaining_faces {
        delete_faces_with_heal(&body, &sources, &targets)?
    } else {
        delete_faces_to_open_shell(&body, &sources, &targets)?
    };

    crate::checked_shape::require_geometry(&shape, "delete face")
        .map_err(|cause| format!("Create DelFace feature failed: {cause}"))?;
    let valid = BRepCheckAnalyzer::new(&shape, true, false, false)
        .and_then(|analyzer| analyzer.is_valid())
        .map_err(|cause| format!("Create DelFace feature failed: BRep check error ({cause})"))?;
    if !valid {
        return Err(
            "Create DelFace feature failed: resulting topology failed BRep validation".into(),
        );
    }
    let achieved = if input.resolution.is_some() {
        Some(super::tolerances::check_budget(
            &body,
            &shape,
            input.resolution,
            "delete face",
        )?)
    } else {
        None
    };

    Ok(BlendCandidate {
        resolution: input.resolution,
        achieved_tolerance: achieved,
        shape,
        history: records,
        source_bodies: input.source_bodies,
        body_parent: input.body_parent,
        primary_body: input.primary_body,
        inherited_deleted: input.inherited_deleted,
        inherited_splits: input.inherited_splits,
        document,
        scope: history.evaluation_scope(),
    })
}

fn delete_faces_to_open_shell(
    body: &Shape,
    sources: &[(Vec<u8>, Shape)],
    targets: &[Shape],
) -> Result<(Shape, Vec<BlendHistoryRecord>), String> {
    let all_faces = collect_shapes(body, ShapeEnum::Face)?;
    let kept: Vec<_> = all_faces
        .into_iter()
        .filter(|face| !targets.iter().any(|target| target.is_same(face)))
        .collect();
    if kept.is_empty() {
        return Err("Create DelFace feature failed: cannot delete every face of the body".into());
    }
    let builder = BRepBuilder::default();
    let mut shell = Shell::default();
    builder.make_shell(&mut shell);
    for face in &kept {
        builder
            .add(&mut shell, face)
            .map_err(|cause| format!("Create DelFace feature failed: shell build ({cause})"))?;
    }
    let shape = shell.into_shape();
    let live = collect_entity_map(&shape)?;
    let records = sources
        .iter()
        .map(|(name, source)| BlendHistoryRecord {
            name: name.clone(),
            source: source.clone(),
            generated: Vec::new(),
            modified: Vec::new(),
            deleted: !live.contains(source),
            transient: false,
        })
        .collect();
    Ok((shape, records))
}

fn delete_faces_with_heal(
    body: &Shape,
    sources: &[(Vec<u8>, Shape)],
    targets: &[Shape],
) -> Result<(Shape, Vec<BlendHistoryRecord>), String> {
    if collect_shapes(body, ShapeEnum::Solid)?.is_empty() {
        return Err(
            "Create DelFace feature failed: Heal Remaining Faces requires a solid body".into(),
        );
    }
    let before_faces = collect_shapes(body, ShapeEnum::Face)?.len();
    let mut algo = BRepAlgoAPIDefeaturing::new();
    algo.set_shape(body);
    for face in targets {
        algo.add_face_to_remove(face);
    }
    algo.set_to_fill_history(true);
    algo.build();
    if !algo.is_done() || algo.has_errors() {
        return Err(
            "Create DelFace feature failed: unable to heal remaining faces around selection".into(),
        );
    }
    let raw_shape = algo.shape().clone();
    if raw_shape.is_null() {
        return Err("Create DelFace feature failed: healed result shape is empty".into());
    }
    let solids = collect_shapes(&raw_shape, ShapeEnum::Solid)?;
    let shape = if solids.len() == 1 {
        solids[0].clone()
    } else {
        raw_shape
    };
    let after_faces = collect_shapes(&shape, ShapeEnum::Face)?.len();
    let live = collect_entity_map(&shape)?;
    for face in targets {
        let deleted = algo.is_deleted(face).unwrap_or(false);
        if !deleted || live.contains(face) {
            return Err(
                "Create DelFace feature failed: remaining faces cannot be extended to heal the deleted face"
                    .into(),
            );
        }
    }
    if after_faces >= before_faces || after_faces == 0 {
        return Err(
            "Create DelFace feature failed: healed face deletion did not reduce the solid cleanly"
                .into(),
        );
    }
    let mut records = Vec::with_capacity(sources.len());
    let mut retained_by_kind = MapOfShape::default();
    for (name, source) in sources {
        let modified = algo
            .modified(source)
            .map_err(|cause| format!("delete face modified history: {cause}"))?;
        let generated = algo
            .generated(source)
            .map_err(|cause| format!("delete face generated history: {cause}"))?;
        let kernel_deleted = algo
            .is_deleted(source)
            .map_err(|cause| format!("delete face deleted history: {cause}"))?;
        if live.contains(source) {
            let _ = retained_by_kind.add(source.clone());
        }
        for image in &modified {
            if live.contains(image) {
                let _ = retained_by_kind.add(image.clone());
            }
        }
        let deleted = kernel_deleted
            || (modified.is_empty() && generated.is_empty() && !live.contains(source));
        records.push(BlendHistoryRecord {
            name: name.clone(),
            source: source.clone(),
            generated,
            modified,
            deleted,
            transient: false,
        });
    }
    // Avoid dual retained+generated ownership when defeaturing both modifies an input edge/vertex
    // and reports it as generated by an extended adjacent face.
    for record in &mut records {
        record
            .generated
            .retain(|child| !retained_by_kind.contains(child));
    }
    Ok((shape, records))
}

fn collect_shapes(shape: &Shape, kind: ShapeEnum) -> Result<Vec<Shape>, String> {
    let mut indexed = IndexedMapOfShape::default();
    topexp::map_shapes_of_type(shape, kind, &mut indexed).map_err(|cause| cause.to_string())?;
    (1..=indexed.extent())
        .map(|index| {
            indexed
                .find_key(index)
                .cloned()
                .map_err(|cause| cause.to_string())
        })
        .collect()
}

fn collect_entity_map(shape: &Shape) -> Result<MapOfShape, String> {
    let mut map = MapOfShape::default();
    for kind in [ShapeEnum::Face, ShapeEnum::Edge, ShapeEnum::Vertex] {
        for entity in collect_shapes(shape, kind)? {
            let _ = map.add(entity);
        }
    }
    Ok(map)
}

fn validate_delete_face_inputs(
    sources: &[(Vec<u8>, Shape)],
    faces: &[Vec<u8>],
) -> Result<(), String> {
    let unique: BTreeSet<_> = faces.iter().collect();
    if faces.is_empty() || unique.len() != faces.len() {
        return Err(format!(
            "delete face faces {}, unique {}: expected unique nonempty faces",
            faces.len(),
            unique.len()
        ));
    }
    for key in faces {
        let _ = named_face_source(sources, key)?;
    }
    Ok(())
}

fn selected_face_shapes(
    sources: &[(Vec<u8>, Shape)],
    faces: &[Vec<u8>],
) -> Result<Vec<Shape>, String> {
    faces
        .iter()
        .map(|key| named_face_source(sources, key).cloned())
        .collect()
}

fn named_face_source<'a>(sources: &'a [(Vec<u8>, Shape)], key: &[u8]) -> Result<&'a Shape, String> {
    let matches: Vec<_> = sources
        .iter()
        .filter(|(name, _)| name == key)
        .map(|(_, shape)| shape)
        .collect();
    let [shape] = matches.as_slice() else {
        return Err(format!(
            "delete face {key:?}, matches {}: expected exactly one named face in the source body",
            matches.len()
        ));
    };
    topods::face(shape)
        .map_err(|cause| format!("delete face {key:?}: expected a face: {cause}"))?;
    Ok(shape)
}
