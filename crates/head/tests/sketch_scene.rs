// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn invalid_plane_is_named_without_removing_other_sketches() {
    let mut snapshot = crate::scene_fixture::selected_prism_document();
    snapshot
        .sketches
        .add_on_work_plane(
            DocumentSketchId(2),
            "Missing plane",
            oblikovati_model::feature::FeatureId(999),
            Default::default(),
        )
        .unwrap();
    let mut update = HeadSketchSceneUpdate {
        source: Some((1, 0)),
        sketches: vec![],
        plane_declines: vec![],
    };
    collect_sketches(&snapshot, &mut update);
    assert_eq!(update.sketches.len(), 1);
    assert_eq!(update.plane_declines.len(), 1);
    assert_eq!(update.plane_declines[0].0, DocumentSketchId(2));
    assert!(update.plane_declines[0].1.contains("999"));
}

#[test]
fn hidden_sketches_leave_the_scene_before_plane_resolution() {
    let mut snapshot = crate::scene_fixture::selected_prism_document();
    snapshot
        .sketches
        .add_on_work_plane(
            DocumentSketchId(2),
            "Missing plane",
            oblikovati_model::feature::FeatureId(999),
            Default::default(),
        )
        .unwrap();
    for id in [DocumentSketchId(1), DocumentSketchId(2)] {
        snapshot.sketches.set_visible(id, false).unwrap();
    }
    let mut update = HeadSketchSceneUpdate {
        source: Some((1, 0)),
        sketches: vec![],
        plane_declines: vec![],
    };
    collect_sketches(&snapshot, &mut update);
    assert!(update.sketches.is_empty());
    assert!(update.plane_declines.is_empty());
    snapshot
        .sketches
        .set_visible(DocumentSketchId(1), true)
        .unwrap();
    collect_sketches(&snapshot, &mut update);
    assert_eq!(update.sketches.len(), 1);
}
