// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchAxisReference, SketchOrientationDefinition},
    feature::{work_axis::WorkAxisLine, work_geometry::WorkGeometryDefinition},
    sketch::{Sketch2D, SketchFrameAxis},
};

fn add_spindle(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<FeatureId, String> {
    let id = ids.allocate_feature().map_err(|cause| cause.to_string())?;
    let axis = WorkAxisLine::new([0.0; 3], [1.0, 0.0, 0.0]).unwrap();
    snapshot
        .work_geometry
        .append(AuthoredFeature::work_axis(id, "Spindle", axis))
        .map_err(|cause| cause.to_string())?;
    Ok(id)
}
fn create_spindle(host: &mut Host, document: u64) -> FeatureId {
    host.commands(document)
        .unwrap()
        .run("Create spindle", add_spindle)
        .unwrap()
}
fn add_oriented_sketch(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<DocumentSketchId, String> {
    let id = ids.allocate_sketch().map_err(|cause| cause.to_string())?;
    snapshot
        .sketches
        .add_on_work_plane(id, "Oriented", FeatureId(1), Sketch2D::default())
        .map_err(|cause| cause.to_string())?;
    snapshot
        .sketches
        .set_orientation(id, Some(spindle_orientation()))
        .map_err(|cause| cause.to_string())?;
    Ok(id)
}
fn spindle_orientation() -> SketchOrientationDefinition {
    SketchOrientationDefinition {
        reference: SketchAxisReference::Feature(FeatureId(2)),
        sketch_axis: SketchFrameAxis::Y,
        reverse: false,
        origin: None,
    }
}
fn saved_mixed_document() -> (Host, SequenceGuids, u64) {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let document = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert_eq!(create_plane(&mut host, document), FeatureId(1));
    assert_eq!(create_spindle(&mut host, document), FeatureId(2));
    host.commands(document)
        .unwrap()
        .run("Orient sketch", add_oriented_sketch)
        .unwrap();
    host.save(document, "alpha", &mut source).unwrap();
    (host, source, document)
}
fn assert_owned_orientation(host: &Host, document: u64, direction: [f64; 3]) {
    let plane = snapshot(host, document)
        .resolve_sketch_plane(DocumentSketchId(1))
        .unwrap();
    assert_eq!(plane.recipe().y_axis, direction);
    assert_eq!(plane.recipe().origin, [0.0, 0.0, 1.0]);
    assert_eq!(snapshot(host, document).work_geometry.features().len(), 2);
}
fn move_spindle(host: &mut Host, document: u64) {
    let axis = WorkAxisLine::new([0.0; 3], [0.0, 1.0, 0.0]).unwrap();
    let definition = WorkGeometryDefinition::Axis(axis);
    host.commands(document)
        .unwrap()
        .run("Move spindle", |snapshot, _| {
            snapshot
                .work_geometry
                .replace(FeatureId(2), definition, definition.reads())
        })
        .unwrap();
}
#[test]
fn current_axis_orientation_survives_history_storage_and_copy() {
    let (mut host, mut source, document) = saved_mixed_document();
    assert_owned_orientation(&host, document, [1.0, 0.0, 0.0]);
    move_spindle(&mut host, document);
    assert_owned_orientation(&host, document, [0.0, 1.0, 0.0]);
    host.commands(document).unwrap().undo().unwrap();
    assert_owned_orientation(&host, document, [1.0, 0.0, 0.0]);
    assert!(!host.workspace().needs_save(document).unwrap());
    host.commands(document).unwrap().redo().unwrap();
    host.save(document, "alpha", &mut source).unwrap();
    let document = reopen(&mut host, document, &mut source);
    assert_owned_orientation(&host, document, [0.0, 1.0, 0.0]);
    verify_mixed_copy(&mut host, document, &mut source);
}
fn verify_mixed_copy(host: &mut Host, document: u64, source: &mut SequenceGuids) {
    host.save_copy(document, &path("mixed-copy.obk"), None, "alpha", source)
        .unwrap();
    let copied = open_file(host, "mixed-copy.obk", source, OpenActivation::Background);
    assert_owned_orientation(host, copied, [0.0, 1.0, 0.0]);
    assert_eq!(create_spindle(host, copied), FeatureId(3));
    assert!(!host.workspace().needs_save(document).unwrap());
}
fn mark_mixed_construction(host: &mut Host, document: u64) {
    host.commands(document)
        .unwrap()
        .run("Construction datums", |snapshot, _| {
            for id in [1, 2] {
                snapshot
                    .work_geometry
                    .set_work_plane_construction(FeatureId(id), true)?;
            }
            Ok::<(), oblikovati_model::feature::FeatureSequenceError>(())
        })
        .unwrap();
}
fn delete_oriented_sketch(host: &mut Host, document: u64) {
    let report = host
        .commands(document)
        .unwrap()
        .run("Delete oriented sketch", |snapshot, _| {
            snapshot.delete_sketch(DocumentSketchId(1))
        })
        .unwrap();
    assert_eq!(report.construction_datums, [FeatureId(2), FeatureId(1)]);
    assert!(snapshot(host, document).work_geometry.features().is_empty());
}
#[test]
fn mixed_construction_cleanup_is_atomic_and_never_reuses_saved_feature_ids() {
    let (mut host, mut source, document) = saved_mixed_document();
    mark_mixed_construction(&mut host, document);
    host.save(document, "alpha", &mut source).unwrap();
    delete_oriented_sketch(&mut host, document);
    host.commands(document).unwrap().undo().unwrap();
    assert_owned_orientation(&host, document, [1.0, 0.0, 0.0]);
    assert!(!host.workspace().needs_save(document).unwrap());
    host.commands(document).unwrap().redo().unwrap();
    host.save(document, "alpha", &mut source).unwrap();
    let document = reopen(&mut host, document, &mut source);
    assert!(
        snapshot(&host, document)
            .work_geometry
            .features()
            .is_empty()
    );
    assert_eq!(create_spindle(&mut host, document), FeatureId(3));
}
