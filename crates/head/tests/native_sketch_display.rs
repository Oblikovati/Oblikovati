// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::HeadWork;
use oblikovati_api::{client::Client, wire::AddSketchEntityArgs};
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

#[test]
fn typed_spline_and_offset_creation_reaches_native_window_pixels() {
    let directory = tempfile::tempdir().unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    super::scene_tests::open_native_part(&mut owner, &directory.path().join("spline.opd"));
    let baseline = super::scene_tests::publish_native_scene(&mut owner, &mut window);
    let fit = add_fit_spline(&mut owner);
    add_offset_spline(&mut owner, fit);
    let changed = super::scene_tests::publish_native_scene(&mut owner, &mut window);
    assert!(window.sketch_display_declines().unwrap().is_empty());
    assert_ne!(changed.rgba, baseline.rgba);
    owner.finish().unwrap();
    window.finish().unwrap();
}

fn add_fit_spline(owner: &mut NativeDesktopOwner) -> u64 {
    let points = [[0., 0.], [0.2, 1.], [2., 1.5], [3., -0.5], [1., -2.]];
    Client::new(owner.document_host())
        .sketch()
        .add_entity(&AddSketchEntityArgs {
            kind: "spline".into(),
            points: Some(points.iter().map(|point| Some(point.to_vec())).collect()),
            construction: true,
            ..Default::default()
        })
        .unwrap()
        .entity_id
}

fn add_offset_spline(owner: &mut NativeDesktopOwner, parent: u64) {
    Client::new(owner.document_host())
        .sketch()
        .add_entity(&AddSketchEntityArgs {
            kind: "offsetSpline".into(),
            entity_refs: Some(vec![parent]),
            radius: "-5 mm".into(),
            construction: true,
            ..Default::default()
        })
        .unwrap();
}
