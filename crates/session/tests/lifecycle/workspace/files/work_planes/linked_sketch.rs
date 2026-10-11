// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchPlacement},
    sketch::{PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId},
};
fn fixed_point(point: SketchEntityId, constraint: SketchConstraintId) -> Sketch2D {
    let mut geometry = Sketch2D::default();
    geometry.add_point(point, [2.0, 3.0]).unwrap();
    geometry
        .add_constraint(
            constraint,
            PlanarConstraint::Fix {
                point,
                position: [2.0, 3.0],
            },
        )
        .unwrap();
    geometry
}
fn create_hosted(host: &mut Host, id: u64, plane: FeatureId) -> DocumentSketchId {
    host.commands(id)
        .unwrap()
        .run("Hosted sketch", |snapshot, ids| {
            let sketch = ids.allocate_sketch().map_err(|error| error.to_string())?;
            let point = ids
                .allocate_sketch_entity()
                .map_err(|error| error.to_string())?;
            let constraint = ids.allocate_sketch_constraint(sketch).unwrap();
            snapshot
                .sketches
                .add_on_work_plane(sketch, "Hosted", plane, fixed_point(point, constraint))
                .map_err(|error| error.to_string())?;
            Ok::<_, String>(sketch)
        })
        .unwrap()
}
fn assert_projection(host: &Host, id: u64, sketch: DocumentSketchId, height: f64) {
    let view = snapshot(host, id).require_resolved_sketch(sketch).unwrap();
    assert_eq!(
        view.sketch.placement(),
        SketchPlacement::WorkPlane(FeatureId(1))
    );
    assert_eq!(view.plane.to_model([2.0, 3.0]).unwrap(), [2.0, 3.0, height]);
    assert_eq!(view.verification.structure.degrees_of_freedom, 0);
}
fn change_height(host: &mut Host, id: u64, sketch: DocumentSketchId) {
    host.commands(id)
        .unwrap()
        .run("Move plane", |snapshot, _| {
            snapshot.parameters.set_expression(1, "6 cm")
        })
        .unwrap();
    assert_projection(host, id, sketch, 6.0);
    host.commands(id).unwrap().undo().unwrap();
    assert_projection(host, id, sketch, 1.0);
    assert!(!host.workspace().needs_save(id).unwrap());
    host.commands(id).unwrap().redo().unwrap();
    assert_projection(host, id, sketch, 6.0);
}
#[test]
fn hosted_sketch_follows_parameter_edits_undo_save_reopen_and_copy() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let plane = create_plane(&mut host, id);
    let sketch = create_hosted(&mut host, id, plane);
    assert_projection(&host, id, sketch, 1.0);
    host.save(id, "alpha", &mut source).unwrap();
    let id = reopen(&mut host, id, &mut source);
    assert_projection(&host, id, sketch, 1.0);
    change_height(&mut host, id, sketch);
    copy_hosted(&mut host, id, sketch, &mut source);
    host.save(id, "alpha", &mut source).unwrap();
    let id = reopen(&mut host, id, &mut source);
    assert_projection(&host, id, sketch, 6.0);
}
fn copy_hosted(host: &mut Host, id: u64, sketch: DocumentSketchId, source: &mut SequenceGuids) {
    host.save_copy(id, &path("hosted-copy.obk"), None, "alpha", source)
        .unwrap();
    let copy = open_file(host, "hosted-copy.obk", source, OpenActivation::Background);
    assert_projection(host, copy, sketch, 6.0);
    assert!(host.workspace().needs_save(id).unwrap());
    assert!(!host.workspace().needs_save(copy).unwrap());
}
fn suppress(host: &mut Host, id: u64, plane: FeatureId, suppressed: bool) {
    host.commands(id)
        .unwrap()
        .run("Suppress plane", |snapshot, _| {
            snapshot.work_geometry.set_suppressed(plane, suppressed)
        })
        .unwrap();
}
#[test]
fn suppressed_placement_stays_unavailable_after_reopen_and_recovers_on_unsuppress() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let plane = create_plane(&mut host, id);
    let sketch = create_hosted(&mut host, id, plane);
    suppress(&mut host, id, plane, true);
    assert!(snapshot(&host, id).require_resolved_sketch(sketch).is_err());
    host.save(id, "alpha", &mut source).unwrap();
    let id = reopen(&mut host, id, &mut source);
    assert!(snapshot(&host, id).require_resolved_sketch(sketch).is_err());
    suppress(&mut host, id, plane, false);
    assert_projection(&host, id, sketch, 1.0);
}
#[test]
fn changing_only_the_reference_is_an_undo_step_and_file_change() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let first = create_plane(&mut host, id);
    let second = create_plane(&mut host, id);
    let sketch = create_hosted(&mut host, id, first);
    host.save(id, "alpha", &mut source).unwrap();
    host.commands(id)
        .unwrap()
        .run("Reattach", |snapshot, _| {
            snapshot.sketches.attach_work_plane(sketch, second)
        })
        .unwrap();
    assert!(host.workspace().needs_save(id).unwrap());
    host.commands(id).unwrap().undo().unwrap();
    assert!(!host.workspace().needs_save(id).unwrap());
    assert_projection(&host, id, sketch, 1.0);
}
