// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_api::types::{
    INFER_HORIZONTAL, INFER_PARALLEL, INFER_PERPENDICULAR, INFER_VERTICAL,
};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{
        DocumentEditHistory, SketchLineCreation, SketchLineCreationReport, SketchReservationMarks,
    },
    identity::RecipeIdAllocator,
    sketch::{
        LineRelation, PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId,
        SketchInferencePolicy, SketchPlane,
    },
    sketch_settings::default_sketch_settings,
    units::UnitsOfMeasure,
};
fn owner(geometry: Sketch2D) -> DocumentEditHistory {
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .sketches
        .add(
            DocumentSketchId(1),
            "Inference",
            SketchPlane::xy(),
            geometry,
        )
        .unwrap();
    DocumentEditHistory::new(snapshot, RecipeIdAllocator::default()).unwrap()
}
fn input(endpoints: [[f64; 2]; 2]) -> SketchLineCreation {
    SketchLineCreation {
        endpoints,
        construction: false,
        centerline: false,
    }
}
fn create(owner: &mut DocumentEditHistory, input: SketchLineCreation) -> SketchLineCreationReport {
    let token = owner.begin("Create inferred line").unwrap();
    let result = owner
        .edit(&token, |snapshot, ids| {
            let policy = SketchInferencePolicy::for_units(snapshot.parameters.units());
            snapshot.create_inferred_sketch_line(
                DocumentSketchId(1),
                &input,
                policy,
                Default::default(),
                ids,
            )
        })
        .unwrap();
    owner.commit(&token, false).unwrap();
    result
}
fn geometry(owner: &DocumentEditHistory) -> &Sketch2D {
    owner
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
}
fn reference() -> Sketch2D {
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), [0.0, 0.0]).unwrap();
    geometry.add_point(SketchEntityId(2), [3.0, 2.0]).unwrap();
    geometry
        .add_line(SketchEntityId(3), [SketchEntityId(1), SketchEntityId(2)])
        .unwrap();
    geometry
}
#[test]
fn all_direction_families_append_one_real_owned_relation() {
    let cases = [
        ([[10.0, 10.0], [14.0, 10.01]], INFER_HORIZONTAL, None),
        ([[10.0, 10.0], [10.01, 14.0]], INFER_VERTICAL, None),
        (
            [[10.0, 10.0], [13.0, 12.0]],
            INFER_PARALLEL,
            Some(SketchEntityId(3)),
        ),
        (
            [[10.0, 10.0], [8.0, 13.0]],
            INFER_PERPENDICULAR,
            Some(SketchEntityId(3)),
        ),
    ];
    for (endpoints, kind, target) in cases {
        assert_created_direction(endpoints, kind, target);
    }
}
fn assert_direction_relation(
    geometry: &Sketch2D,
    line: SketchEntityId,
    kind: oblikovati_api::types::ConstraintInferenceKind,
) {
    let expected = match kind {
        INFER_HORIZONTAL => PlanarConstraint::LineHorizontal { line },
        INFER_VERTICAL => PlanarConstraint::LineVertical { line },
        INFER_PARALLEL => PlanarConstraint::Lines {
            lines: [line, SketchEntityId(3)],
            relation: LineRelation::Parallel,
        },
        INFER_PERPENDICULAR => PlanarConstraint::Lines {
            lines: [line, SketchEntityId(3)],
            relation: LineRelation::Perpendicular,
        },
        _ => panic!("unexpected inference {kind:?}"),
    };
    assert_eq!(geometry.constraints()[0].relation, expected);
}
#[test]
fn snaps_keep_fresh_endpoint_ids_and_precede_the_direction_constraint() {
    let mut history = owner(reference());
    let report = create(&mut history, input([[0.0001, 0.0001], [3.0001, 2.0001]]));
    assert_eq!(
        report.primitive.points,
        [SketchEntityId(4), SketchEntityId(5)]
    );
    assert_eq!(report.primitive.entity, SketchEntityId(6));
    assert_eq!(
        report.point_constraints,
        [SketchConstraintId(1), SketchConstraintId(2)]
    );
    assert_eq!(report.direction.unwrap().geometric_index, 2);
    assert_snapped_endpoints(geometry(&history), &report);
}
#[test]
fn disabled_constraint_application_keeps_proposals_without_phantom_records() {
    let mut history = owner(reference());
    disable_constraint_application(&mut history);
    let endpoints = [[0.0001, 0.0001], [3.0001, 2.0001]];
    let report = create(&mut history, input(endpoints));
    assert_eq!(report.inference.points.len(), 2);
    assert!(report.inference.direction.is_some());
    assert!(report.direction.is_none());
    assert!(report.point_constraints.is_empty());
    assert!(geometry(&history).constraints().is_empty());
    assert_eq!(
        geometry(&history)
            .point(report.primitive.points[0])
            .unwrap()
            .position,
        endpoints[0]
    );
}
#[test]
fn zero_length_new_endpoints_do_not_snap_to_each_other_or_create_a_direction() {
    let mut history = owner(Sketch2D::default());
    let report = create(&mut history, input([[2.0, 3.0]; 2]));
    assert!(report.inference.points.is_empty());
    assert!(report.direction.is_none());
    assert!(geometry(&history).constraints().is_empty());
    assert_ne!(report.primitive.points[0], report.primitive.points[1]);
}
#[test]
fn construction_and_centerline_flags_remain_independent_after_creation() {
    for construction in [false, true] {
        for centerline in [false, true] {
            let mut history = owner(Sketch2D::default());
            let mut request = input([[0.0, 0.0], [5.0, 1.0]]);
            request.construction = construction;
            request.centerline = centerline;
            let line = create(&mut history, request).primitive.entity;
            assert_eq!(geometry(&history).is_centerline(line).unwrap(), centerline);
            assert_eq!(
                geometry(&history).is_construction(line).unwrap(),
                construction || centerline
            );
        }
    }
}
#[path = "inferred_line_creation/failures.rs"]
mod failures;

#[path = "inferred_line_creation/current_sources.rs"]
mod current_sources;

#[path = "inferred_line_creation/generated.rs"]
mod generated;

#[path = "inferred_line_creation/go_cases.rs"]
mod go_cases;

fn assert_created_direction(
    endpoints: [[f64; 2]; 2],
    kind: oblikovati_api::types::ConstraintInferenceKind,
    target: Option<SketchEntityId>,
) {
    let mut history = owner(reference());
    let report = create(&mut history, input(endpoints));
    let applied = report.direction.unwrap();
    assert_eq!(
        (applied.inference.kind, applied.inference.target),
        (kind, target)
    );
    assert_eq!(applied.constraint, SketchConstraintId(1));
    assert_eq!(applied.geometric_index, 0);
    assert_direction_relation(geometry(&history), report.primitive.entity, kind);
}

fn assert_snapped_endpoints(actual: &Sketch2D, report: &SketchLineCreationReport) {
    for (index, target) in [SketchEntityId(1), SketchEntityId(2)]
        .into_iter()
        .enumerate()
    {
        assert_eq!(
            actual
                .point(report.primitive.points[index])
                .unwrap()
                .position,
            actual.point(target).unwrap().position
        );
        assert_eq!(
            actual.constraints()[index].relation,
            PlanarConstraint::Coincident {
                points: [report.primitive.points[index], target]
            }
        );
    }
}

fn disable_constraint_application(history: &mut DocumentEditHistory) {
    history
        .refresh(|snapshot| -> Result<(), String> {
            let mut settings = default_sketch_settings();
            settings.auto_apply_constraints = false;
            snapshot.sketch_settings = Some(settings);
            Ok(())
        })
        .unwrap();
}
