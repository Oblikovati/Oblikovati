// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchAxisReference, SketchOrientationDefinition},
    feature::work_geometry::{
        DatumLength, WorkAxisPoints, WorkGeometryDefinition, WorkPointReference,
    },
    sketch::{Sketch2D, SketchFrameAxis},
};

fn reserve_datum(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    definition: WorkGeometryDefinition,
) -> Result<FeatureId, String> {
    let id = ids.allocate_feature().map_err(|cause| cause.to_string())?;
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(id, "Datum", definition))
        .map_err(|cause| cause.to_string())?;
    Ok(id)
}
fn add_driven_datums(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<(), String> {
    let point = WorkGeometryDefinition::Point([
        DatumLength::Constant(2.0),
        DatumLength::Parameter(1),
        DatumLength::Constant(0.0),
    ]);
    let point_id = reserve_datum(snapshot, ids, point)?;
    let axis = WorkGeometryDefinition::AxisThroughPoints(WorkAxisPoints {
        first: WorkPointReference::Origin,
        second: WorkPointReference::Feature(point_id),
    });
    reserve_datum(snapshot, ids, axis)?;
    Ok(())
}
fn add_axis_sketch(
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
        .set_orientation(id, Some(point_axis_orientation()))
        .map_err(|cause| cause.to_string())?;
    Ok(id)
}
fn point_axis_orientation() -> SketchOrientationDefinition {
    SketchOrientationDefinition {
        reference: SketchAxisReference::Feature(FeatureId(3)),
        sketch_axis: SketchFrameAxis::Y,
        reverse: false,
        origin: None,
    }
}
fn saved_points() -> (Host, SequenceGuids, u64) {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let document = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    create_plane(&mut host, document);
    host.commands(document)
        .unwrap()
        .run("Point and axis", add_driven_datums)
        .unwrap();
    host.commands(document)
        .unwrap()
        .run("Axis sketch", add_axis_sketch)
        .unwrap();
    host.save(document, "alpha", &mut source).unwrap();
    (host, source, document)
}
fn assert_point_placement(host: &Host, document: u64, height: f64) {
    let snapshot = snapshot(host, document);
    let evaluated = snapshot
        .work_geometry
        .evaluate_work_geometry(&snapshot.parameters)
        .unwrap();
    assert_eq!(evaluated.point(FeatureId(2)).unwrap(), [2.0, height, 0.0]);
    let plane = snapshot.resolve_sketch_plane(DocumentSketchId(1)).unwrap();
    assert_eq!(plane.recipe().origin, [0.0, 0.0, height]);
    for (actual, expected) in plane
        .recipe()
        .y_axis
        .into_iter()
        .zip(evaluated.axis(FeatureId(3)).unwrap().direction())
    {
        assert!((actual - expected).abs() < f64::EPSILON * 2.0);
    }
}
fn change_height(host: &mut Host, document: u64) {
    host.commands(document)
        .unwrap()
        .run("Move point", |snapshot, _| {
            snapshot.parameters.set_expression(1, "4 cm")
        })
        .unwrap();
}
#[test]
fn point_parameter_edits_follow_history_storage_and_independent_copies() {
    let (mut host, mut source, document) = saved_points();
    assert_point_placement(&host, document, 1.0);
    change_height(&mut host, document);
    assert_point_placement(&host, document, 4.0);
    host.commands(document).unwrap().undo().unwrap();
    assert_point_placement(&host, document, 1.0);
    assert!(!host.workspace().needs_save(document).unwrap());
    host.commands(document).unwrap().redo().unwrap();
    host.save(document, "alpha", &mut source).unwrap();
    let document = reopen(&mut host, document, &mut source);
    assert_point_placement(&host, document, 4.0);
    verify_point_copy(&mut host, document, &mut source);
}
fn verify_point_copy(host: &mut Host, document: u64, source: &mut SequenceGuids) {
    host.save_copy(document, &path("point-copy.obk"), None, "alpha", source)
        .unwrap();
    let copied = open_file(host, "point-copy.obk", source, OpenActivation::Background);
    assert_point_placement(host, copied, 4.0);
    assert_eq!(create_plane(host, copied), FeatureId(4));
    assert!(!host.workspace().needs_save(document).unwrap());
}
fn flag_point_chain(host: &mut Host, document: u64) {
    host.commands(document)
        .unwrap()
        .run("Construction datums", |snapshot, _| {
            for id in [1, 2, 3] {
                snapshot
                    .work_geometry
                    .set_work_plane_construction(FeatureId(id), true)?;
            }
            Ok::<(), oblikovati_model::feature::FeatureSequenceError>(())
        })
        .unwrap();
}
fn delete_point_chain(host: &mut Host, document: u64) {
    let report = host
        .commands(document)
        .unwrap()
        .run("Delete axis sketch", |snapshot, _| {
            snapshot.delete_sketch(DocumentSketchId(1))
        })
        .unwrap();
    assert_eq!(
        report.construction_datums,
        [FeatureId(3), FeatureId(2), FeatureId(1)]
    );
    assert!(snapshot(host, document).work_geometry.features().is_empty());
}
#[test]
fn point_axis_construction_cleanup_is_one_step_with_permanent_reservations() {
    let (mut host, mut source, document) = saved_points();
    flag_point_chain(&mut host, document);
    host.save(document, "alpha", &mut source).unwrap();
    delete_point_chain(&mut host, document);
    host.commands(document).unwrap().undo().unwrap();
    assert_point_placement(&host, document, 1.0);
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
    assert_eq!(create_plane(&mut host, document), FeatureId(4));
}

fn change_point_x(host: &mut Host, document: u64, value: f64) {
    let definition = WorkGeometryDefinition::Point([
        DatumLength::Constant(value),
        DatumLength::Parameter(1),
        DatumLength::Constant(0.0),
    ]);
    host.commands(document)
        .unwrap()
        .run("Point X", |snapshot, _| {
            snapshot
                .work_geometry
                .replace(FeatureId(2), definition, definition.reads())
        })
        .unwrap();
}
fn assert_signed_point_x(host: &Host, document: u64) {
    let snapshot = snapshot(host, document);
    let evaluated = snapshot
        .work_geometry
        .evaluate_work_geometry(&snapshot.parameters)
        .unwrap();
    assert_eq!(
        evaluated.point(FeatureId(2)).unwrap()[0].to_bits(),
        (-0.0f64).to_bits()
    );
}
#[test]
fn signed_point_coordinate_is_an_edit_and_survives_saved_recipe_restore() {
    let (mut host, mut source, document) = saved_points();
    change_point_x(&mut host, document, 0.0);
    host.save(document, "alpha", &mut source).unwrap();
    change_point_x(&mut host, document, -0.0);
    assert!(host.workspace().needs_save(document).unwrap());
    assert_signed_point_x(&host, document);
    host.commands(document).unwrap().undo().unwrap();
    assert!(!host.workspace().needs_save(document).unwrap());
    host.commands(document).unwrap().redo().unwrap();
    assert!(host.workspace().needs_save(document).unwrap());
    host.save(document, "alpha", &mut source).unwrap();
    let document = reopen(&mut host, document, &mut source);
    assert_signed_point_x(&host, document);
}
