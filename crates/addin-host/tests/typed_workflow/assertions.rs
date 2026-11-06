// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) type WorkflowFiles =
    FileWorkspace<WorkflowObserver, WorkflowObserver, NativeDocumentFiles>;
pub(super) fn assert_saved_workflow(files: &WorkflowFiles, path: &Path, report: &WorkflowReport) {
    assert!(path.is_file());
    let owner = files.workspace().session(report.reopened).unwrap();
    assert!(!files.workspace().needs_save(report.reopened).unwrap());
    let placed = &owner.snapshot().sketches.sketches()[0];
    owner
        .snapshot()
        .require_resolved_sketch(placed.id())
        .unwrap();
    let circle = placed.geometry().circles()[0];
    assert_eq!(circle.radius.to_bits(), report.radius_bits);
    assert_ne!(report.created, report.reopened);
    assert!(
        owner
            .snapshot()
            .parameters
            .find(&report.parameter)
            .is_some()
    );
}
