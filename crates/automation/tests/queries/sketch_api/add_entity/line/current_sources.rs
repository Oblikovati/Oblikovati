// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{history::RecipeReservations, sketch::DistanceMeasure};
#[test]
fn current_dimensions_move_snap_targets_and_are_excluded_from_reply_indices() {
    let (mut files, document, mut source) = fresh();
    let seed = add_line(&mut files, &mut source, [[2.0, 3.0], [5.0, 5.0]]);
    files
        .commands(document)
        .unwrap()
        .run("Drive reference", |snapshot, ids| {
            drive_reference(snapshot, ids, seed.entity_id)
        })
        .unwrap();
    set_width(&mut files, &mut source, "4 cm");
    let reply = add_line(&mut files, &mut source, [[6.0001, 3.0001], [10.0, 3.0001]]);
    let snap = &reply.inferred_points.as_ref().unwrap()[0];
    let point = reply.point_i_ds.as_ref().unwrap()[0];
    let target = seed.point_i_ds.as_ref().unwrap()[1];
    assert_point_record(snap, point, target);
    assert_direction(&reply, "horizontal", 3, None);
    assert_model_index(&files, document, 5, direction(&reply).constraint_index);
}
fn drive_reference(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    line: u64,
) -> Result<(), String> {
    let sketch = DocumentSketchId(1);
    let mut geometry = snapshot.sketches.get(sketch).unwrap().geometry().clone();
    let endpoints = geometry.line(SketchEntityId(line)).unwrap().endpoints;
    append_driving_relations(&mut geometry, ids, sketch, SketchEntityId(line), endpoints);
    snapshot
        .sketches
        .replace_geometry(sketch, geometry)
        .map_err(|cause| cause.to_string())
}
fn append_driving_relations(
    geometry: &mut Sketch2D,
    ids: &mut RecipeReservations<'_>,
    sketch: DocumentSketchId,
    line: SketchEntityId,
    points: [SketchEntityId; 2],
) {
    let fixed = PlanarConstraint::Fix {
        point: points[0],
        position: [2.0, 3.0],
    };
    let horizontal = PlanarConstraint::LineHorizontal { line };
    let distance = PlanarConstraint::Distance {
        points,
        parameter: 1,
        measure: DistanceMeasure::Horizontal,
    };
    for relation in [fixed, horizontal, distance] {
        geometry
            .add_constraint(ids.allocate_sketch_constraint(sketch).unwrap(), relation)
            .unwrap();
    }
}
fn assert_model_index(files: &DiscoveryWorkspace, document: u64, constraint: u64, index: i64) {
    let geometry = owned_geometry(files, document).sketches()[0].geometry();
    let actual = geometry
        .geometric_constraint_index(SketchConstraintId(constraint))
        .unwrap();
    assert_eq!(i64::try_from(actual).unwrap(), index);
}
