// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::GuiOriginPlaneKind;

fn part(document: u64) -> GuiCommandState {
    GuiCommandState {
        document: Some(document),
        authoring: GuiAuthoringContext::Part,
        ..Default::default()
    }
}
fn sketch(document: u64, id: u64) -> GuiCommandState {
    GuiCommandState {
        active_sketch: Some(DocumentSketchId(id)),
        authoring: GuiAuthoringContext::Sketch,
        ..part(document)
    }
}
#[test]
fn finishing_the_requested_sketch_resumes_its_extrusion() {
    let mut pending = PendingExtrude::default();
    pending.start(Some(7));
    assert_eq!(pending.observe(&part(7), true, &[]), None);
    assert_eq!(
        pending.observe(
            &part(7),
            false,
            &[GuiShellAction::StartSketchOn(GuiOriginPlaneKind::Xy)]
        ),
        None
    );
    assert_eq!(pending.observe(&sketch(7, 9), false, &[]), None);
    assert_eq!(
        pending.observe(&sketch(7, 9), false, &[GuiShellAction::FinishSketch]),
        None
    );
    assert_eq!(
        pending.observe(&part(7), false, &[]),
        Some(DocumentSketchId(9))
    );
    assert_eq!(pending.observe(&part(7), false, &[]), None);
}
#[test]
fn cancelling_plane_selection_or_changing_document_clears_the_request() {
    let mut pending = PendingExtrude::default();
    pending.start(Some(7));
    assert_eq!(pending.observe(&part(7), false, &[]), None);
    assert_eq!(
        pending.observe(&sketch(7, 9), false, &[GuiShellAction::FinishSketch]),
        None
    );
    assert_eq!(pending.observe(&part(7), false, &[]), None);
    pending.start(Some(7));
    assert_eq!(pending.observe(&part(8), true, &[]), None);
    assert_eq!(
        pending.observe(&sketch(7, 9), false, &[GuiShellAction::FinishSketch]),
        None
    );
    assert_eq!(pending.observe(&part(7), false, &[]), None);
}
#[test]
fn ordinary_sketch_and_unfinished_exit_do_not_resume_extrusion() {
    let mut pending = PendingExtrude::default();
    assert_eq!(
        pending.observe(&sketch(7, 9), false, &[GuiShellAction::FinishSketch]),
        None
    );
    assert_eq!(pending.observe(&part(7), false, &[]), None);
    pending.start(Some(7));
    assert_eq!(pending.observe(&sketch(7, 9), false, &[]), None);
    assert_eq!(pending.observe(&part(7), false, &[]), None);
    assert_eq!(
        pending.observe(&sketch(7, 10), false, &[GuiShellAction::FinishSketch]),
        None
    );
    assert_eq!(pending.observe(&part(7), false, &[]), None);
}

#[test]
fn switching_sketches_cancels_the_original_request() {
    let mut pending = PendingExtrude::default();
    pending.start(Some(7));
    assert_eq!(pending.observe(&sketch(7, 9), false, &[]), None);
    assert_eq!(
        pending.observe(&sketch(7, 10), false, &[GuiShellAction::FinishSketch]),
        None
    );
    assert_eq!(pending.observe(&part(7), false, &[]), None);
}
