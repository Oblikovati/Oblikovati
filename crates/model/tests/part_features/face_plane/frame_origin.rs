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
