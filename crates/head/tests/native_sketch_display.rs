// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{Sketch2D, SketchEntityId, SketchPlane},
};
#[test]
fn native_display_failure_names_the_sketch_and_entity_and_clears_with_source() {
    let mut window = NativeHeadWindow::open().unwrap();
    window.submit_sketch_scene(failure_update()).unwrap();
    verify_failure_feedback(&mut window);
    window
        .submit_sketch_scene(crate::HeadSketchSceneUpdate {
            source: None,
            sketches: vec![],
            plane_declines: vec![],
        })
        .unwrap();
    assert_eq!(window.view.native_title().unwrap(), "Oblikovati");
    assert!(window.sketch_display_declines().unwrap().is_empty());
    window.finish().unwrap();
}

fn failure_update() -> crate::HeadSketchSceneUpdate {
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), [0., 0.]).unwrap();
    geometry
        .add_line(SketchEntityId(2), [SketchEntityId(1); 2])
        .unwrap();
    let packet = oblikovati_viewport::ViewportSketchDisplay {
        sketch: DocumentSketchId(4),
        plane: SketchPlane::xy(),
        geometry,
    };
    crate::HeadSketchSceneUpdate {
        source: Some((1, 2)),
        sketches: vec![packet],
        plane_declines: vec![(DocumentSketchId(9), "missing plane 99".into())],
    }
}

fn verify_failure_feedback(window: &mut NativeHeadWindow) {
    assert_eq!(window.sketch_scene_status().unwrap().source, Some((1, 2)));
    assert_eq!(window.sketch_display_declines().unwrap().len(), 1);
    window.present().unwrap();
    let title = window.view.native_title().unwrap();
    assert!(
        title.contains("2 sketch display failures")
            && title.contains("sketch 4, entity 2")
            && title.contains("missing plane 99"),
        "{title}"
    );
}
