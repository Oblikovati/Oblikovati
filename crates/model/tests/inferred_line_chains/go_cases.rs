// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::*;
use oblikovati_model::sketch::{PlanarConstraint, SketchDirectionInference};
use oblikovati_model::sketch_settings::default_sketch_settings;
#[derive(serde::Deserialize)]
#[serde(rename_all = "PascalCase")]
struct GoChain {
    input: Vec<[f64; 2]>,
    closed: bool,
    infer: bool,
    apply: bool,
    priority: i32,
    points: Vec<[f64; 2]>,
    members: Vec<usize>,
    segments: Vec<GoSegment>,
}
#[derive(serde::Deserialize)]
#[serde(rename_all = "PascalCase")]
struct GoSegment {
    target_positions: [[f64; 2]; 2],
    endpoints: [usize; 2],
    positions: [[f64; 2]; 2],
    snaps: Vec<[usize; 2]>,
    kind: String,
    target: i32,
    constraints: usize,
}
#[test]
fn eighty_go_cases_preserve_chain_semantics_and_reject_zero_vector_directions() {
    let fixtures: Vec<GoChain> = serde_json::from_str(include_str!("go-chains.json")).unwrap();
    assert_eq!(fixtures.len(), 80);
    let exceptions: Vec<_> = fixtures
        .iter()
        .map(|fixture| {
            fixture
                .segments
                .iter()
                .filter(|segment| degenerate_go_direction(segment))
                .count()
        })
        .collect();
    assert_eq!(exceptions.iter().filter(|count| **count > 0).count(), 18);
    assert_eq!(exceptions.iter().sum::<usize>(), 33);
    for fixture in fixtures {
        let mut settings = default_sketch_settings();
        settings.infer_constraints = fixture.infer;
        settings.auto_apply_constraints = fixture.apply;
        settings.constraint_priority = ConstraintInferencePriority(fixture.priority);
        let mut history = owner(Some(settings));
        let report = create(&mut history, &input(&fixture.input, fixture.closed));
        verify_fixture(geometry(&history), &report, &fixture);
    }
}
fn verify_fixture(sketch: &Sketch2D, report: &SketchLineChainCreationReport, fixture: &GoChain) {
    assert_eq!(
        sketch
            .points()
            .iter()
            .map(|p| p.position)
            .collect::<Vec<_>>(),
        fixture.points
    );
    let members = sketch.entity_ids();
    assert_eq!(members.len(), fixture.members.len());
    assert_eq!(
        members
            .iter()
            .map(|id| id.0 as usize - 1)
            .collect::<Vec<_>>(),
        fixture.members
    );
    assert_eq!(report.lines.len(), fixture.segments.len());
    let mut constraints = 0;
    let mut degenerate_go_relations = 0;
    for (line, expected) in report.lines.iter().zip(&fixture.segments) {
        verify_segment(sketch, line, expected, fixture.apply);
        constraints += line.point_constraints.len() + usize::from(line.direction.is_some());
        if fixture.apply && degenerate_go_direction(expected) {
            degenerate_go_relations += usize::from(line.direction.is_none());
        }
        assert_eq!(constraints, expected.constraints - degenerate_go_relations);
    }
    assert_eq!(sketch.constraints().len(), constraints);
}
fn verify_segment(
    sketch: &Sketch2D,
    line: &oblikovati_model::history::SketchLineCreationReport,
    expected: &GoSegment,
    apply: bool,
) {
    let point_ids: Vec<_> = sketch.points().iter().map(|point| point.id).collect();
    assert_eq!(
        line.primitive.points,
        expected.endpoints.map(|i| point_ids[i])
    );
    assert_eq!(line.inference.candidate_endpoints, expected.positions);
    let snaps: Vec<_> = line
        .inference
        .points
        .iter()
        .map(|snap| [line.primitive.points[snap.endpoint], snap.target])
        .collect();
    assert_eq!(
        snaps,
        expected
            .snaps
            .iter()
            .map(|indices| indices.map(|i| point_ids[i]))
            .collect::<Vec<_>>()
    );
    verify_direction(sketch, line.inference.direction, expected);
    assert_eq!(
        line.direction.is_some(),
        apply && line.inference.direction.is_some()
    );
    verify_snap_relations(sketch, line, &snaps, apply);
}
fn verify_snap_relations(
    sketch: &Sketch2D,
    line: &oblikovati_model::history::SketchLineCreationReport,
    snaps: &[[SketchEntityId; 2]],
    apply: bool,
) {
    assert_eq!(
        line.point_constraints.len(),
        if apply { snaps.len() } else { 0 }
    );
    for (id, points) in line.point_constraints.iter().zip(snaps) {
        let constraint = sketch
            .constraints()
            .iter()
            .find(|relation| relation.id == *id)
            .unwrap();
        assert_ne!(points[0], points[1]);
        assert_eq!(
            constraint.relation,
            PlanarConstraint::Coincident { points: *points }
        );
    }
}
fn verify_direction(
    sketch: &Sketch2D,
    actual: Option<SketchDirectionInference>,
    expected: &GoSegment,
) {
    // The existing Rust selector refuses a direction for a zero vector; Go invents one.
    if degenerate_go_direction(expected) {
        verify_degenerate_direction(sketch, actual, expected);
        return;
    }
    let Some(actual) = actual else {
        assert!(
            expected.kind.is_empty(),
            "positions {:?}, Go kind {}",
            expected.positions,
            expected.kind
        );
        return;
    };
    let kind = match actual.kind {
        INFER_HORIZONTAL => "horizontal",
        INFER_VERTICAL => "vertical",
        INFER_PARALLEL => "parallel",
        INFER_PERPENDICULAR => "perpendicular",
        other => panic!("unexpected direction {other:?}"),
    };
    assert_eq!(kind, expected.kind);
    assert_eq!(
        actual.target,
        (expected.target >= 0).then(|| sketch.entity_ids()[expected.target as usize])
    );
}

fn degenerate_go_direction(expected: &GoSegment) -> bool {
    !expected.kind.is_empty()
        && (expected.positions[0] == expected.positions[1]
            || (expected.target >= 0
                && expected.target_positions[0] == expected.target_positions[1]))
}

fn verify_degenerate_direction(
    sketch: &Sketch2D,
    actual: Option<SketchDirectionInference>,
    expected: &GoSegment,
) {
    if expected.positions[0] == expected.positions[1] {
        assert!(actual.is_none());
        return;
    }
    if let Some(actual) = actual {
        let target = actual
            .target
            .expect("remaining chain cases have only relative directions");
        let line = sketch
            .lines()
            .iter()
            .find(|line| line.id == target)
            .unwrap();
        assert_ne!(
            sketch.point(line.endpoints[0]).unwrap().position,
            sketch.point(line.endpoints[1]).unwrap().position
        );
        assert_ne!(target, sketch.entity_ids()[expected.target as usize]);
    }
}
