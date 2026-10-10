// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn recompute_command_groups_parameter_and_geometry_edits_for_undo_and_file_storage() {
    let (mut host, mut source, id, sketch) = saved_driven_sketch();
    edit_and_recompute(&mut host, id, sketch);
    assert!((point_height(&host, id, sketch) - 6.0).abs() < 1e-8);
    host.commands(id).unwrap().undo().unwrap();
    assert!((point_height(&host, id, sketch) - 1.0).abs() < 1e-8);
    assert!(!host.workspace().needs_save(id).unwrap());
    host.commands(id).unwrap().redo().unwrap();
    host.save(id, "alpha", &mut source).unwrap();
    host.close_discarding_changes(id).unwrap();
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert!((point_height(&host, reopened, sketch) - 6.0).abs() < 1e-8);
}

#[test]
fn cancelled_or_unchanged_recompute_does_not_add_history_or_dirty_a_saved_document() {
    let (mut host, _, id, _) = saved_driven_sketch();
    let labels = owned_undo_labels(&host, id);
    for cancelled in [false, true] {
        let report = host
            .commands(id)
            .unwrap()
            .recompute_sketches(SolveOptions::default(), || cancelled)
            .unwrap();
        assert_eq!(report.cancelled, cancelled);
        assert_eq!(host.workspace().session(id).unwrap().undo_labels(), labels);
        assert!(!host.workspace().needs_save(id).unwrap());
    }
}

fn edit_and_recompute(host: &mut Host, id: u64, sketch: DocumentSketchId) {
    host.commands(id)
        .unwrap()
        .begin("Edit driven sketch")
        .unwrap();
    host.commands(id)
        .unwrap()
        .run("Width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "6 cm")
        })
        .unwrap();
    let report = host
        .commands(id)
        .unwrap()
        .recompute_sketches(SolveOptions::default(), || false)
        .unwrap();
    assert_recompute_report(&report, sketch);
    host.commands(id).unwrap().end().unwrap();
}

fn saved_driven_sketch() -> (Host, SequenceGuids, u64, DocumentSketchId) {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let sketch = host
        .commands(id)
        .unwrap()
        .run("Sketch", add_sketch)
        .unwrap();
    host.save(id, "alpha", &mut source).unwrap();
    (host, source, id, sketch)
}
fn owned_undo_labels(host: &Host, id: u64) -> Vec<String> {
    host.workspace()
        .session(id)
        .unwrap()
        .undo_labels()
        .into_iter()
        .map(str::to_owned)
        .collect()
}
fn assert_recompute_report(
    report: &oblikovati_model::document_sketch::DocumentSketchRecompute,
    sketch: DocumentSketchId,
) {
    assert!(!report.cancelled);
    assert_eq!(report.reports[0].sketch, sketch);
    let result = report.reports[0].result.as_ref().unwrap();
    assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
}
