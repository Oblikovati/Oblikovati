// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn empty_update() -> crate::HeadSketchSceneUpdate {
    crate::HeadSketchSceneUpdate {
        source: None,
        sketches: vec![],
        plane_declines: vec![],
    }
}
#[test]
fn unsupported_window_accepts_clear_but_refuses_committed_sketches() {
    let mut window = FakeHeadWindow::default();
    window.submit_sketch_scene(empty_update()).unwrap();
    let cause = window.submit_sketch_scene(nonempty_update()).unwrap_err();
    assert!(
        cause.contains("Some((1, 2))") && cause.contains("committed sketch rendering"),
        "{cause}"
    );
    assert!(window.calls.is_empty());
}
fn nonempty_update() -> crate::HeadSketchSceneUpdate {
    let packet = oblikovati_viewport::ViewportSketchDisplay {
        sketch: oblikovati_model::document_sketch::DocumentSketchId(3),
        plane: oblikovati_model::sketch::SketchPlane::xy(),
        geometry: Default::default(),
    };
    crate::HeadSketchSceneUpdate {
        source: Some((1, 2)),
        sketches: vec![packet],
        plane_declines: vec![],
    }
}
#[test]
fn unsupported_window_cannot_silently_drop_a_plane_failure() {
    let mut update = empty_update();
    update.plane_declines.push((
        oblikovati_model::document_sketch::DocumentSketchId(7),
        "missing plane".into(),
    ));
    let mut window = FakeHeadWindow::default();
    assert!(
        window
            .submit_sketch_scene(update)
            .unwrap_err()
            .contains("committed sketch rendering")
    );
}
