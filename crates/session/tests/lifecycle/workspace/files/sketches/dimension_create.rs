// SPDX-License-Identifier: GPL-2.0-only
use super::recompute::saved_driven_sketch;
use super::*;
use oblikovati_model::{
    history::{SketchDimensionCreation, SketchDimensionCreationReport},
    sketch::{
        DistanceMeasure, MeasuredDimensionOperands, SketchDimensionalConstraintCreation,
        SketchEntityId,
    },
};
fn input() -> SketchDimensionCreation {
    SketchDimensionCreation {
        operands: SketchDimensionalConstraintCreation::Measured(
            MeasuredDimensionOperands::Distance {
                points: [SketchEntityId(1), SketchEntityId(2)],
                measure: DistanceMeasure::Aligned,
            },
        ),
        expression: "width * 4".into(),
        reference: false,
        text_point: Some([-2.0, 3.0]),
    }
}
fn create(
    host: &mut Host,
    document: u64,
    sketch: DocumentSketchId,
) -> SketchDimensionCreationReport {
    host.commands(document)
        .unwrap()
        .create_sketch_dimension(sketch, &input())
        .unwrap()
}
fn assert_saved_creation(
    host: &Host,
    document: u64,
    sketch: DocumentSketchId,
    report: &SketchDimensionCreationReport,
) {
    let snapshot = host.workspace().session(document).unwrap().snapshot();
    let source = snapshot.parameters.get(report.parameter).unwrap();
    assert_eq!(
        (source.name.as_str(), source.source.as_str()),
        (report.parameter_name.as_str(), "width * 4")
    );
    assert_eq!(source.kind, MODEL_PARAM);
    let geometry = snapshot.sketches.get(sketch).unwrap().geometry();
    assert_eq!(
        geometry.dimension_text_point(report.constraint).unwrap(),
        Some([-2.0, 3.0])
    );
    assert!((geometry.measure_dimension(report.constraint).unwrap().value - 4.0).abs() < 1e-8);
}
#[test]
fn workspace_creation_formula_and_annotation_survive_save_reopen_and_copy() {
    let (mut host, mut source, document, sketch) = saved_driven_sketch();
    let report = create(&mut host, document, sketch);
    assert_eq!(
        (
            report.parameter,
            report.constraint.0,
            report.parameter_name.as_str()
        ),
        (2, 5, "d0")
    );
    host.save(document, "alpha", &mut source).unwrap();
    host.close_discarding_changes(document).unwrap();
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert_saved_creation(&host, reopened, sketch, &report);
    assert_copied_creation(&mut host, &mut source, reopened, sketch, &report);
}
fn assert_copied_creation(
    host: &mut Host,
    source: &mut SequenceGuids,
    document: u64,
    sketch: DocumentSketchId,
    report: &SketchDimensionCreationReport,
) {
    host.save_copy(document, &path("dimension-copy.obk"), None, "alpha", source)
        .unwrap();
    let copied = open_file(
        host,
        "dimension-copy.obk",
        source,
        OpenActivation::Background,
    );
    assert_saved_creation(host, copied, sketch, report);
    host.commands(copied)
        .unwrap()
        .delete_sketch_dimension(sketch, report.constraint)
        .unwrap();
    assert_next_identity(&create(host, copied, sketch));
}

#[test]
fn workspace_projected_creation_failure_consumes_reservations_but_publishes_no_source() {
    let (mut host, _, document, sketch) = saved_driven_sketch();
    let result = host
        .commands(document)
        .unwrap()
        .create_sketch_dimension_projected(sketch, &input(), |result| -> Result<(), String> {
            assert_eq!(result.unwrap().parameter_name, "d0");
            Err("reply refused".into())
        });
    assert!(result.is_err());
    let snapshot = host.workspace().session(document).unwrap().snapshot();
    assert!(snapshot.parameters.get(2).is_none());
    assert!(host.workspace().needs_save(document).unwrap());
    assert_next_identity(&create(&mut host, document, sketch));
}

fn assert_next_identity(next: &SketchDimensionCreationReport) {
    assert_eq!(
        (
            next.parameter,
            next.constraint.0,
            next.parameter_name.as_str()
        ),
        (3, 6, "d1")
    );
}
