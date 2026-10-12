// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        AuthoredFeature, FeatureId, FeatureReadFootprint, FeatureReadKey,
        work_plane::{WorkPlaneDefinition, WorkPlaneDistance, WorkPlaneReference},
    },
    history::{DocumentSketchDeleteError, DocumentSnapshot},
    sketch::{Sketch2D, SketchEntityId, SketchPlane},
};
use std::error::Error;

fn add_datum(
    snapshot: &mut DocumentSnapshot,
    id: u64,
    definition: WorkPlaneDefinition,
    construction: bool,
) {
    let feature = AuthoredFeature::work_plane(FeatureId(id), &format!("Plane{id}"), definition);
    snapshot.work_geometry.append(feature).unwrap();
    snapshot
        .work_geometry
        .set_work_plane_construction(FeatureId(id), construction)
        .unwrap();
}
fn offset_of(id: u64) -> WorkPlaneDefinition {
    WorkPlaneDefinition::Offset {
        base: WorkPlaneReference::Feature(FeatureId(id)),
        distance: WorkPlaneDistance::Constant(2.0),
    }
}
fn add_hosted(snapshot: &mut DocumentSnapshot, sketch: u64, plane: u64) {
    let mut geometry = Sketch2D::default();
    geometry
        .add_point(SketchEntityId(sketch), [4.0, 5.0])
        .unwrap();
    snapshot
        .sketches
        .add_on_work_plane(
            DocumentSketchId(sketch),
            &format!("Sketch{sketch}"),
            FeatureId(plane),
            geometry,
        )
        .unwrap();
}
fn chain(flags: [bool; 3]) -> DocumentSnapshot {
    let mut snapshot = document_snapshot::empty_snapshot();
    add_datum(
        &mut snapshot,
        1,
        WorkPlaneDefinition::Fixed(SketchPlane::xy()),
        flags[0],
    );
    add_datum(&mut snapshot, 2, offset_of(1), flags[1]);
    add_datum(
        &mut snapshot,
        3,
        WorkPlaneDefinition::Fixed(SketchPlane::yz()),
        flags[2],
    );
    add_hosted(&mut snapshot, 1, 2);
    snapshot
}
fn declare_sketch_reader(snapshot: &mut DocumentSnapshot, feature: u64, unknown: bool) {
    let id = FeatureId(feature);
    let definition = snapshot.work_geometry.get(id).unwrap().definition().clone();
    let mut keys = definition.reads().keys().clone();
    keys.insert(FeatureReadKey::Sketch(1));
    snapshot
        .work_geometry
        .replace(id, definition, FeatureReadFootprint::new(keys, unknown))
        .unwrap();
}
fn assert_authored_unchanged(before: &DocumentSnapshot, after: &DocumentSnapshot) {
    assert!(
        before
            .work_geometry
            .same_work_plane_recipe(&after.work_geometry)
    );
    assert!(before.sketches.same_evaluated_state(&after.sketches));
}

#[test]
fn deletion_prunes_the_complete_construction_chain_but_keeps_initially_unused_planes() {
    let mut snapshot = chain([true, true, true]);
    let unused = snapshot.resolve_sketch_plane(DocumentSketchId(1)).unwrap();
    let report = snapshot.delete_sketch(DocumentSketchId(1)).unwrap();
    assert_eq!(report.sketch, DocumentSketchId(1));
    assert_eq!(report.construction_datums, [FeatureId(2), FeatureId(1)]);
    assert_eq!(snapshot.work_geometry.features()[0].id(), FeatureId(3));
    assert!(snapshot.sketches.sketches().is_empty());
    assert_eq!(unused.recipe().origin, [0.0, 0.0, 2.0]);
}

#[test]
fn ordinary_child_planes_protect_construction_ancestors() {
    let mut snapshot = chain([true, false, true]);
    let before = snapshot.work_geometry.clone();
    let report = snapshot.delete_sketch(DocumentSketchId(1)).unwrap();
    assert!(report.construction_datums.is_empty());
    assert!(snapshot.work_geometry.same_work_plane_recipe(&before));
}

#[test]
fn ordinary_ancestors_survive_construction_child_deletion() {
    let mut snapshot = chain([false, true, false]);
    let report = snapshot.delete_sketch(DocumentSketchId(1)).unwrap();
    assert_eq!(report.construction_datums, [FeatureId(2)]);
    assert_eq!(
        snapshot
            .work_geometry
            .features()
            .iter()
            .map(|plane| plane.id())
            .collect::<Vec<_>>(),
        [FeatureId(1), FeatureId(3)]
    );
}

#[test]
fn the_last_of_multiple_sketch_consumers_unlocks_cleanup_without_changing_other_sketches() {
    let mut snapshot = chain([true, true, true]);
    add_hosted(&mut snapshot, 2, 2);
    let other = snapshot.sketches.get(DocumentSketchId(2)).unwrap().clone();
    let frame = snapshot.resolve_sketch_plane(DocumentSketchId(2)).unwrap();
    let first = snapshot.delete_sketch(DocumentSketchId(1)).unwrap();
    assert!(first.construction_datums.is_empty());
    assert_eq!(&other, snapshot.sketches.get(DocumentSketchId(2)).unwrap());
    assert!(frame.same_recipe(snapshot.resolve_sketch_plane(DocumentSketchId(2)).unwrap()));
    assert_eq!(
        snapshot
            .delete_sketch(DocumentSketchId(2))
            .unwrap()
            .construction_datums,
        [FeatureId(2), FeatureId(1)]
    );
}

#[test]
fn suppressed_and_cutoff_feature_consumers_still_protect_authored_references() {
    let mut snapshot = chain([true, true, false]);
    let definition = offset_of(1);
    snapshot
        .work_geometry
        .replace(FeatureId(3), definition.clone(), definition.reads())
        .unwrap();
    snapshot
        .work_geometry
        .set_suppressed(FeatureId(3), true)
        .unwrap();
    snapshot.work_geometry.set_end_of_part(Some(0)).unwrap();
    let report = snapshot.delete_sketch(DocumentSketchId(1)).unwrap();
    assert_eq!(report.construction_datums, [FeatureId(2)]);
    assert_eq!(snapshot.work_geometry.end_of_part(), Some(0));
    assert!(snapshot.work_geometry.get(FeatureId(1)).is_ok());
    assert!(
        snapshot
            .work_geometry
            .get(FeatureId(3))
            .unwrap()
            .suppressed()
    );
}

#[test]
fn consumed_sketch_declines_name_sorted_stable_readers_before_any_mutation() {
    let mut snapshot = chain([true, true, true]);
    for id in [3, 2] {
        declare_sketch_reader(&mut snapshot, id, false);
    }
    let before = snapshot.clone();
    let error = snapshot.delete_sketch(DocumentSketchId(1)).unwrap_err();
    assert_eq!(
        error,
        DocumentSketchDeleteError::Consumed {
            sketch: DocumentSketchId(1),
            readers: vec![
                FeatureReadKey::Feature(FeatureId(2)),
                FeatureReadKey::Feature(FeatureId(3))
            ]
        }
    );
    assert_decline(&error, "consumed");
    assert_authored_unchanged(&before, &snapshot);
}

#[test]
fn unknown_attribution_is_refused_and_missing_sketches_do_not_prune_planes() {
    let mut snapshot = chain([true, true, true]);
    declare_sketch_reader(&mut snapshot, 2, true);
    let before = snapshot.clone();
    let error = snapshot.delete_sketch(DocumentSketchId(1)).unwrap_err();
    assert!(matches!(
        error,
        DocumentSketchDeleteError::Unattributed { .. }
    ));
    assert!(error.to_string().contains("unattributed"));
    assert!(error.source().is_none());
    let missing = snapshot.delete_sketch(DocumentSketchId(999)).unwrap_err();
    assert!(missing.to_string().contains("999"));
    assert!(missing.source().is_some());
    assert_authored_unchanged(&before, &snapshot);
}

fn assert_decline(error: &DocumentSketchDeleteError, expected: &str) {
    assert!(error.to_string().contains(expected));
    assert!(error.source().is_none());
}
