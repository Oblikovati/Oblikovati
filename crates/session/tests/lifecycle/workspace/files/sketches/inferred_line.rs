// SPDX-License-Identifier: GPL-2.0-only
use super::recompute::saved_driven_sketch;
use super::*;
use oblikovati_model::history::{SketchLineCreation, SketchLineCreationReport};
fn request() -> SketchLineCreation {
    SketchLineCreation {
        endpoints: [[0.0001, 0.0001], [4.0001, 0.0001]],
        construction: false,
        centerline: true,
    }
}
#[test]
fn file_commands_preserve_line_relations_and_reservations_through_save_and_copy() {
    let (mut host, mut source, document, sketch, report) = saved_inferred_line();
    assert_line_owner(&host, document, sketch, &report);
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Background);
    assert_line_owner(&host, reopened, sketch, &report);
    let copy = open_file(
        &mut host,
        "inferred-copy.obk",
        &mut source,
        OpenActivation::Background,
    );
    assert_line_owner(&host, copy, sketch, &report);
}
fn assert_line_owner(
    host: &Host,
    document: u64,
    sketch: DocumentSketchId,
    report: &SketchLineCreationReport,
) {
    let snapshot = host.workspace().session(document).unwrap().snapshot();
    let view = snapshot.require_resolved_sketch(sketch).unwrap();
    let geometry = view.sketch.geometry();
    assert!(geometry.is_centerline(report.primitive.entity).unwrap());
    assert_eq!(report.point_constraints.len(), 2);
    let applied = report.direction.unwrap();
    assert_eq!(
        geometry
            .geometric_constraint_index(applied.constraint)
            .unwrap(),
        applied.geometric_index
    );
    assert_eq!(view.verification.structure.degrees_of_freedom, 0);
}
#[test]
fn file_command_reply_rejection_keeps_reservations_dirty_without_publishing_a_line() {
    let (mut host, _, document, sketch) = saved_driven_sketch();
    let before = sketch_recipe(&host, document);
    let result = host
        .commands(document)
        .unwrap()
        .create_inferred_sketch_line_projected(sketch, &request(), |result| {
            assert_eq!(result.unwrap().point_constraints.len(), 2);
            Err::<(), _>("reply refused")
        });
    assert!(result.unwrap_err().to_string().contains("reply refused"));
    assert_eq!(sketch_recipe(&host, document), before);
    assert!(host.workspace().needs_save(document).unwrap());
}

fn saved_inferred_line() -> (
    Host,
    SequenceGuids,
    u64,
    DocumentSketchId,
    SketchLineCreationReport,
) {
    let (mut host, mut source, document, sketch) = saved_driven_sketch();
    let report = host
        .commands(document)
        .unwrap()
        .create_inferred_sketch_line(sketch, &request())
        .unwrap();
    assert_line_owner(&host, document, sketch, &report);
    host.save(document, "alpha", &mut source).unwrap();
    host.save_copy(
        document,
        &path("inferred-copy.obk"),
        None,
        "alpha",
        &mut source,
    )
    .unwrap();
    (host, source, document, sketch, report)
}

fn sketch_recipe(
    host: &Host,
    document: u64,
) -> oblikovati_model::document_sketch::DocumentSketches {
    host.workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .clone()
}
