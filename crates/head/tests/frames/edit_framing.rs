// SPDX-License-Identifier: GPL-2.0-only
//! Edit-framing capability refusal.
use super::*;

#[test]
fn unsupported_window_declines_edit_framing_without_mutation() {
    assert!(
        crate::work_idle::IdleHeadWork
            .decline_sketch_edit("test".into())
            .is_err()
    );
    let mut window = FakeHeadWindow::default();
    assert!(!window.set_sketch_edit(None).unwrap());
    let edit = crate::HeadSketchEdit {
        document: 1,
        sketch: oblikovati_model::document_sketch::DocumentSketchId(2),
        plane: oblikovati_model::sketch::SketchPlane::xy(),
    };
    assert!(
        window
            .set_sketch_edit(Some(edit))
            .unwrap_err()
            .contains("framing")
    );
}
