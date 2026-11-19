// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn named_frame_origin_is_resolved_from_the_current_body_lineage() {
    use oblikovati_model::document_sketch::{
        DocumentSketchId, SketchFrameEditDefinition, SketchFrameOrigin, SketchFrameTopologyOrigin,
    };
    let mut owner = part_owner();
    append_prism(&mut owner);
    let mut snapshot = owner.snapshot().clone();
    let id = DocumentSketchId(99);
    snapshot
        .sketches
        .add(id, "Named origin", SketchPlane::xy(), Default::default())
        .unwrap();
    let reference = SketchFrameTopologyOrigin::new(face_key(2)).unwrap();
    snapshot
        .sketches
        .set_frame_edit(
            id,
            Some(SketchFrameEditDefinition {
                origin: Some(SketchFrameOrigin::Topology(reference)),
                alignment: None,
            }),
        )
        .unwrap();
    assert!(snapshot.requires_named_part_replay());
    let evaluated = snapshot
        .evaluate_part_features(&mut FaceEvaluator::default())
        .unwrap();
    let datums = evaluated.datum_evaluation();
    assert_eq!(
        snapshot
            .sketches
            .get(id)
            .unwrap()
            .plane_with_axes(Some(&datums), &datums)
            .unwrap()
            .recipe()
            .origin,
        [3., 4., 0.]
    );
    let failed = snapshot
        .evaluate_part_features(&mut FaceEvaluator {
            fail_face: true,
            ..Default::default()
        })
        .unwrap();
    let missing = failed.datum_evaluation();
    assert!(
        snapshot
            .sketches
            .get(id)
            .unwrap()
            .plane_with_axes(Some(&missing), &missing)
            .is_err()
    );
}

#[test]
fn origin_from_the_sketchs_own_downstream_body_is_rejected_as_a_cycle() {
    use oblikovati_model::document_sketch::{
        SketchFrameEditDefinition, SketchFrameOrigin, SketchFrameTopologyOrigin,
    };
    let mut owner = part_owner();
    append_prism(&mut owner);
    let mut candidate = owner.snapshot().clone();
    let reference = SketchFrameTopologyOrigin::new(face_key(2)).unwrap();
    candidate
        .sketches
        .set_frame_edit(
            SKETCH,
            Some(SketchFrameEditDefinition {
                origin: Some(SketchFrameOrigin::Topology(reference)),
                alignment: None,
            }),
        )
        .unwrap();
    assert!(candidate.refresh_prism_reads().is_err());
    assert!(
        owner
            .snapshot()
            .sketches
            .get(SKETCH)
            .unwrap()
            .frame_edit()
            .is_none()
    );
}

#[test]
fn named_frame_rejects_a_retained_replay_after_source_parameters_change() {
    use oblikovati_model::document_sketch::{
        DocumentSketchId, SketchFrameEditDefinition, SketchFrameOrigin, SketchFrameTopologyOrigin,
    };
    let mut owner = part_owner();
    append_prism(&mut owner);
    let mut snapshot = owner.snapshot().clone();
    let sketch = DocumentSketchId(99);
    snapshot
        .sketches
        .add(
            sketch,
            "Named origin",
            SketchPlane::xy(),
            Default::default(),
        )
        .unwrap();
    snapshot
        .sketches
        .set_frame_edit(
            sketch,
            Some(SketchFrameEditDefinition {
                origin: Some(SketchFrameOrigin::Topology(
                    SketchFrameTopologyOrigin::new(face_key(2)).unwrap(),
                )),
                alignment: None,
            }),
        )
        .unwrap();
    let part = snapshot
        .evaluate_part_features(&mut FaceEvaluator::default())
        .unwrap();
    let old = part.datum_evaluation();
    assert!(
        snapshot
            .resolve_sketch_plane_with_datums(sketch, &old)
            .is_ok()
    );
    snapshot.parameters.set_expression(3, "3 cm").unwrap();
    assert!(
        snapshot
            .resolve_sketch_plane_with_datums(sketch, &old)
            .is_err()
    );
}

#[test]
fn named_frame_rejects_old_replay_after_source_geometry_or_topology_changes() {
    use oblikovati_model::document_sketch::{
        DocumentSketchId, SketchFrameEditDefinition, SketchFrameOrigin, SketchFrameTopologyOrigin,
    };
    use oblikovati_model::sketch::SketchEntityId;
    let mut owner = part_owner();
    append_prism(&mut owner);
    let mut source = owner.snapshot().clone();
    let sketch = DocumentSketchId(99);
    source
        .sketches
        .add(
            sketch,
            "Named origin",
            SketchPlane::xy(),
            Default::default(),
        )
        .unwrap();
    source
        .sketches
        .set_frame_edit(
            sketch,
            Some(SketchFrameEditDefinition {
                origin: Some(SketchFrameOrigin::Topology(
                    SketchFrameTopologyOrigin::new(face_key(2)).unwrap(),
                )),
                alignment: None,
            }),
        )
        .unwrap();
    let datums = source
        .evaluate_part_features(&mut FaceEvaluator::default())
        .unwrap()
        .datum_evaluation();
    let mut geometry_changed = source.clone();
    let mut geometry = geometry_changed
        .sketches
        .get(SKETCH)
        .unwrap()
        .geometry()
        .clone();
    geometry.add_point(SketchEntityId(901), [1., 1.]).unwrap();
    geometry_changed
        .sketches
        .replace_geometry(SKETCH, geometry)
        .unwrap();
    assert!(
        geometry_changed
            .resolve_sketch_plane_with_datums(sketch, &datums)
            .is_err()
    );
    let mut topology_changed = source.clone();
    topology_changed
        .work_geometry
        .set_suppressed(FeatureId(2), true)
        .unwrap();
    assert!(
        topology_changed
            .resolve_sketch_plane_with_datums(sketch, &datums)
            .is_err()
    );
}

#[test]
fn current_part_replay_can_resolve_a_new_sketch_frame() {
    use oblikovati_model::document_sketch::{
        DocumentSketchId, SketchFrameEditDefinition, SketchFrameOrigin, SketchFrameTopologyOrigin,
    };
    let mut owner = part_owner();
    append_prism(&mut owner);
    let source = owner.snapshot();
    let mut candidate = source.clone();
    let sketch = DocumentSketchId(99);
    candidate
        .sketches
        .add(sketch, "New", SketchPlane::xy(), Default::default())
        .unwrap();
    candidate
        .sketches
        .set_frame_edit(
            sketch,
            Some(SketchFrameEditDefinition {
                origin: Some(SketchFrameOrigin::Topology(
                    SketchFrameTopologyOrigin::new(face_key(2)).unwrap(),
                )),
                alignment: None,
            }),
        )
        .unwrap();
    let part = candidate
        .evaluate_part_features(&mut FaceEvaluator::default())
        .unwrap();
    assert_eq!(
        candidate
            .resolve_sketch_plane_with_datums(sketch, &part.datum_evaluation())
            .unwrap()
            .recipe()
            .origin,
        [3., 4., 0.]
    );
}
