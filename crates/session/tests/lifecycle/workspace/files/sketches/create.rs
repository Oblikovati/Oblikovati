// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{document_sketch::SketchPlacement, history::DocumentSketchCreation};
fn creation_request() -> DocumentSketchCreation {
    DocumentSketchCreation {
        name: "Created sketch".into(),
        placement: SketchPlacement::Fixed(SketchPlane::yz()),
        orientation: None,
        project_origin: true,
    }
}
fn create_in_workspace(host: &mut Host, document: u64) -> DocumentSketchId {
    host.commands(document)
        .unwrap()
        .create_sketch(&creation_request())
        .unwrap()
}
fn assert_created_workspace_sketch(host: &Host, document: u64, sketch: DocumentSketchId) {
    let snapshot = host.workspace().session(document).unwrap().snapshot();
    let view = snapshot.require_resolved_sketch(sketch).unwrap();
    assert_eq!(view.plane, SketchPlane::yz());
    assert_eq!(view.sketch.geometry().points()[0].position, [0.0; 2]);
    assert_eq!(view.sketch.point_projections().len(), 1);
    assert_eq!(view.verification.structure.degrees_of_freedom, 0);
}
#[test]
fn creation_from_workspace_commands_survives_native_save_and_reopen() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1500);
    let document = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let sketch = create_in_workspace(&mut host, document);
    assert_created_workspace_sketch(&host, document, sketch);
    host.save(document, "alpha", &mut source).unwrap();
    host.close_discarding_changes(document).unwrap();
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert_created_workspace_sketch(&host, reopened, sketch);
    assert!(!host.workspace().needs_save(reopened).unwrap());
    assert!(
        host.workspace()
            .session(reopened)
            .unwrap()
            .undo_labels()
            .is_empty()
    );
}
#[test]
fn workspace_reply_failure_publishes_no_sketch_and_retains_permanent_reservations() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1600);
    let document = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let result =
        host.commands(document)
            .unwrap()
            .create_sketch_projected(&creation_request(), |created| {
                assert_eq!(created.unwrap(), DocumentSketchId(1));
                Err::<(), _>("reply rejected")
            });
    assert!(result.is_err());
    let snapshot = host.workspace().session(document).unwrap().snapshot();
    assert!(snapshot.sketches.sketch_ids().is_empty());
    assert!(host.workspace().needs_save(document).unwrap());
    let sketch = create_in_workspace(&mut host, document);
    assert_eq!(sketch, DocumentSketchId(2));
    assert_created_workspace_sketch(&host, document, sketch);
}
