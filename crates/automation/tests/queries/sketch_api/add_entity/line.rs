// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::{
    AppliedConstraintInference, AppliedPointInference, InferenceOptionsView,
};
use oblikovati_model::sketch::{LineRelation, PlanarConstraint};
fn line_input(endpoints: [[f64; 2]; 2]) -> AddSketchEntityArgs {
    literal_input("line", &endpoints)
}
fn set_inference(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    infer: bool,
    constrain: bool,
    priority: &str,
) {
    Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .set_inference_options(&InferenceOptionsView {
            infer_enabled: infer,
            constrain_enabled: constrain,
            priority: priority.into(),
        })
        .unwrap();
}
fn direction(reply: &AddSketchEntityResult) -> &AppliedConstraintInference {
    let records = reply.inferred_constraints.as_ref().unwrap();
    assert_eq!(records.len(), 1);
    &records[0]
}
fn assert_direction(reply: &AddSketchEntityResult, kind: &str, index: i64, target: Option<u64>) {
    let actual = direction(reply);
    let mut entities = vec![reply.entity_id];
    entities.extend(target);
    assert_eq!(actual.kind, kind);
    assert_eq!(actual.constraint_index, index);
    assert_eq!(actual.entities.as_ref(), Some(&entities));
}
#[test]
fn canonical_horizontal_reply_matches_the_go_wire_fixture() {
    let (mut files, document, mut source) = fresh();
    let reply = add_line(&mut files, &mut source, [[0.0, 8.0], [4.0, 8.02]]);
    assert_eq!(reply.kind, "line");
    assert_eq!(reply.point_i_ds.as_deref(), Some([2, 3].as_slice()));
    assert_eq!(reply.entity_id, 4);
    assert_direction(&reply, "horizontal", 0, None);
    let placed = owned_geometry(&files, document).sketches()[0].geometry();
    assert_eq!(
        placed.constraints()[0].relation,
        PlanarConstraint::LineHorizontal {
            line: SketchEntityId(4)
        }
    );
    assert!(reply.inferred_points.is_none());
}
#[test]
fn vertical_reply_uses_the_canonical_kind_and_owned_line_operand() {
    let (mut files, document, mut source) = fresh();
    let reply = add_line(&mut files, &mut source, [[8.0, 0.0], [8.02, 4.0]]);
    assert_direction(&reply, "vertical", 0, None);
    let placed = owned_geometry(&files, document).sketches()[0].geometry();
    assert_eq!(
        placed.constraints()[0].relation,
        PlanarConstraint::LineVertical {
            line: SketchEntityId(reply.entity_id),
        }
    );
}
#[test]
fn both_endpoint_snap_records_use_created_point_ids_and_existing_target_ids() {
    let (mut files, _, mut source) = fresh();
    add_call(
        &mut files,
        &mut source,
        &literal_input("point", &[[4.0, 0.0]]),
    )
    .unwrap();
    let reply = add_line(
        &mut files,
        &mut source,
        [[0.0001, 0.0001], [4.0001, 0.0001]],
    );
    assert_eq!(reply.point_i_ds.as_deref(), Some([3, 4].as_slice()));
    assert_direction(&reply, "horizontal", 2, None);
    let snaps = reply.inferred_points.as_ref().unwrap();
    assert_point_record(&snaps[0], 3, 1);
    assert_point_record(&snaps[1], 4, 2);
}
fn assert_point_record(record: &AppliedPointInference, point: u64, target: u64) {
    assert_eq!(record.kind, "onPoint");
    assert_eq!(record.point_id, point);
    assert_eq!(record.entities.as_deref(), Some([target].as_slice()));
}
#[test]
fn parallel_and_perpendicular_replies_name_the_new_line_and_reference_line() {
    for (end, kind, relation) in [
        ([13.0, 12.0], "parallel", LineRelation::Parallel),
        ([8.0, 13.0], "perpendicular", LineRelation::Perpendicular),
    ] {
        assert_line_relation_wire(end, kind, relation);
    }
}
fn assert_line_relation_wire(end: [f64; 2], kind: &str, relation: LineRelation) {
    let (mut files, document, mut source) = fresh();
    let first = add_line(&mut files, &mut source, [[0.0, 1.0], [3.0, 3.0]]);
    let reply = add_line(&mut files, &mut source, [[10.0, 10.0], end]);
    assert_direction(&reply, kind, 0, Some(first.entity_id));
    let geometry = owned_geometry(&files, document).sketches()[0].geometry();
    assert_eq!(
        geometry.constraints()[0].relation,
        PlanarConstraint::Lines {
            lines: [
                SketchEntityId(reply.entity_id),
                SketchEntityId(first.entity_id)
            ],
            relation,
        }
    );
}
#[test]
fn audit_mode_reports_point_proposals_without_a_phantom_applied_direction() {
    let (mut files, document, mut source) = fresh();
    set_inference(&mut files, &mut source, true, false, "horizontalVertical");
    let reply = add_line(&mut files, &mut source, [[0.0001, 0.0001], [4.0, 0.01]]);
    assert!(reply.inferred_constraints.is_none());
    assert_point_record(&reply.inferred_points.as_ref().unwrap()[0], 2, 1);
    assert_eq!(
        row(&files, reply.entity_id).points,
        expected_points(&[[0.0001, 0.0001], [4.0, 0.01]])
    );
    assert_no_authored_constraints(&files, document);
}
#[test]
fn disabled_inference_keeps_raw_coordinates_and_omits_both_reply_arrays() {
    let (mut files, _, mut source) = fresh();
    set_inference(&mut files, &mut source, false, false, "none");
    let reply = add_line(&mut files, &mut source, [[0.0001, 0.0001], [4.0, 0.01]]);
    assert!(reply.inferred_constraints.is_none());
    assert!(reply.inferred_points.is_none());
    assert_eq!(
        row(&files, reply.entity_id).points,
        expected_points(&[[0.0001, 0.0001], [4.0, 0.01]])
    );
}
#[test]
fn zero_length_line_keeps_distinct_points_without_phantom_inference() {
    let (mut files, _, mut source) = fresh();
    let reply = add_line(&mut files, &mut source, [[2.0, 3.0]; 2]);
    let points = reply.point_i_ds.unwrap();
    assert_eq!(points.len(), 2);
    assert_ne!(points[0], points[1]);
    assert!(reply.inferred_constraints.is_none());
    assert!(reply.inferred_points.is_none());
}
#[test]
fn line_variant_is_ignored_and_both_roles_survive_the_canonical_constructor() {
    for variant in ["", "twoPoints", "legacy-ignored-variant"] {
        let (mut files, document, mut source) = fresh();
        let mut input = line_input([[0.0, 1.0], [4.0, 2.0]]);
        input.variant = variant.into();
        input.construction = true;
        input.centerline = true;
        let reply = add_call(&mut files, &mut source, &input).unwrap();
        assert!(row(&files, reply.entity_id).construction);
        assert!(
            owned_geometry(&files, document).sketches()[0]
                .geometry()
                .is_centerline(SketchEntityId(reply.entity_id))
                .unwrap()
        );
    }
}
#[path = "line/failures.rs"]
mod failures;
#[path = "line/history.rs"]
mod history;
#[path = "line/units.rs"]
mod units;

#[path = "line/current_sources.rs"]
mod current_sources;

fn add_line(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    endpoints: [[f64; 2]; 2],
) -> AddSketchEntityResult {
    add_call(files, source, &line_input(endpoints)).unwrap()
}
fn assert_no_authored_constraints(files: &DiscoveryWorkspace, document: u64) {
    let sketch = &owned_geometry(files, document).sketches()[0];
    assert!(sketch.geometry().constraints().is_empty());
}
