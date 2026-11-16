// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::SketchPlacement, history::DocumentSketchCreation, sketch::SketchPlane,
};
use oblikovati_viewport::ViewportNavigation;
#[test]
fn native_sketch_edit_faces_plane_and_restores_view_after_finish_and_resize() {
    let directory = tempfile::tempdir().unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    let document = open_native_part(&mut owner, &directory.path().join("edit.opd"));
    publish_native_scene(&mut owner, &mut window);
    let original = window.view.camera_state().unwrap();
    let plane =
        SketchPlane::new([2., 0., 0.], [1., 0., 0.], [0., 0., 1.], Default::default()).unwrap();
    let sketch = owner
        .create_sketch_for_edit(
            document,
            &DocumentSketchCreation {
                name: "Side sketch".into(),
                placement: SketchPlacement::Fixed(plane),
                orientation: None,
                project_origin: true,
            },
        )
        .unwrap();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    let facing = window.view.camera_state().unwrap();
    assert_eq!(facing.eye, [2., -20., 0.]);
    assert_eq!(facing.target, [2., 0., 0.]);
    assert_eq!(facing.up, [0., 0., 1.]);
    assert_eq!(facing.scale, original.scale);
    verify_native_edit_navigation(&mut window, owner.sketch_edit().unwrap());
    owner.start_slot_placement(document, sketch, 0.5).unwrap();
    owner.finish_sketch_edit().unwrap();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    assert_eq!(window.view.camera_state().unwrap(), original);
    assert!(!owner.placement_active());
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn verify_native_edit_navigation(window: &mut NativeHeadWindow, edit: crate::HeadSketchEdit) {
    let point = window
        .sketch_point([700, 400], edit.plane)
        .unwrap()
        .unwrap();
    let pixel = window.sketch_pixel(edit.plane, point).unwrap();
    assert!((pixel[0] - 700).abs() <= 1 && (pixel[1] - 400).abs() <= 1);
    window
        .view
        .navigate(ViewportNavigation::Pan([20, 10]))
        .unwrap();
    let navigated = window.view.camera_state().unwrap();
    assert!(!window.set_sketch_edit(Some(edit)).unwrap());
    assert_eq!(window.view.camera_state().unwrap(), navigated);
    window.view.resize([900, 600]).unwrap();
    window.view.poll_events(64).unwrap();
    window.gui_frame(None, &Default::default()).unwrap();
    capture_native_scene(window);
}
#[test]
fn native_sketch_camera_update_and_failed_entry_preserve_return_view() {
    let mut window = NativeHeadWindow::open().unwrap();
    let original = window.view.camera_state().unwrap();
    let mut edit = crate::HeadSketchEdit {
        document: 1,
        sketch: fixture::SKETCH,
        plane: SketchPlane::xy(),
        look_at_plane: true,
    };
    assert!(window.set_sketch_edit(Some(edit)).unwrap());
    edit.plane =
        SketchPlane::new([1., 2., 3.], [0., 1., 0.], [0., 0., 1.], Default::default()).unwrap();
    let current = window.view.camera_state().unwrap();
    assert!(!window.set_sketch_edit(Some(edit)).unwrap());
    assert_eq!(window.view.camera_state().unwrap(), current);
    let invalid = crate::HeadSketchEdit {
        sketch: oblikovati_model::document_sketch::DocumentSketchId(300),
        plane: SketchPlane::new(
            [f64::MAX; 3],
            [1., 0., 0.],
            [0., 1., 0.],
            Default::default(),
        )
        .unwrap(),
        ..edit
    };
    assert!(window.set_sketch_edit(Some(invalid)).is_err());
    assert_eq!(window.view.camera_state().unwrap(), current);
    edit.sketch = oblikovati_model::document_sketch::DocumentSketchId(200);
    window.set_sketch_edit(Some(edit)).unwrap();
    assert!(window.set_sketch_edit(None).unwrap());
    assert_eq!(window.view.camera_state().unwrap(), original);
    assert!(!window.set_sketch_edit(None).unwrap());
    window.finish().unwrap();
}

#[test]
fn native_unframeable_sketch_declines_edit_and_keeps_created_document() {
    let directory = tempfile::tempdir().unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    let document = open_native_part(&mut owner, &directory.path().join("far-edit.opd"));
    publish_native_scene(&mut owner, &mut window);
    let original = window.view.camera_state().unwrap();
    let plane = SketchPlane::new(
        [f64::MAX; 3],
        [1., 0., 0.],
        [0., 1., 0.],
        Default::default(),
    )
    .unwrap();
    let sketch = owner
        .create_sketch_for_edit(
            document,
            &DocumentSketchCreation {
                name: "Far sketch".into(),
                placement: SketchPlacement::Fixed(plane),
                orientation: None,
                project_origin: false,
            },
        )
        .unwrap();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    assert!(owner.active_sketch_edit().is_none());
    assert_eq!(window.view.camera_state().unwrap(), original);
    assert!(matches!(
        window.command_notice,
        Some(crate::DesktopCommandNotice::SketchEditFailed { .. })
    ));
    assert!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .sketches
            .get(sketch)
            .is_ok()
    );
    owner.finish().unwrap();
    window.finish().unwrap();
}

#[test]
fn disabled_sketch_framing_preserves_view_through_entry_navigation_and_finish() {
    let mut window = NativeHeadWindow::open().unwrap();
    let original = window.view.camera_state().unwrap().isometric();
    window.view.set_camera(original).unwrap();
    let edit = crate::HeadSketchEdit {
        document: 1,
        sketch: fixture::SKETCH,
        plane: SketchPlane::xy(),
        look_at_plane: false,
    };
    assert!(!window.set_sketch_edit(Some(edit)).unwrap());
    assert_eq!(window.view.camera_state().unwrap(), original);
    window
        .view
        .navigate(ViewportNavigation::Pan([20, 10]))
        .unwrap();
    let navigated = window.view.camera_state().unwrap();
    assert!(!window.set_sketch_edit(Some(edit)).unwrap());
    assert!(!window.set_sketch_edit(None).unwrap());
    assert_eq!(window.view.camera_state().unwrap(), navigated);
    let framed = crate::HeadSketchEdit {
        look_at_plane: true,
        ..edit
    };
    assert!(window.set_sketch_edit(Some(framed)).unwrap());
    assert_ne!(window.view.camera_state().unwrap(), navigated);
    assert!(window.set_sketch_edit(None).unwrap());
    assert_eq!(window.view.camera_state().unwrap(), navigated);
    window.finish().unwrap();
}

struct FixedPartPreference(bool);
impl oblikovati_model::application_options::PartPreferenceAccess for FixedPartPreference {
    fn part_options(&self) -> oblikovati_model::application_options::PartApplicationOptions {
        oblikovati_model::application_options::PartApplicationOptions {
            look_at_sketch_plane: self.0,
            ..Default::default()
        }
    }
    fn set_part_options(
        &mut self,
        options: oblikovati_model::application_options::PartApplicationOptions,
    ) -> Result<(), String> {
        self.0 = options.look_at_sketch_plane;
        Ok(())
    }
}
#[test]
fn accepted_part_setting_reaches_the_created_sketch_camera() {
    let directory = tempfile::tempdir().unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone())
        .unwrap()
        .with_part_preferences(Box::new(FixedPartPreference(false)));
    let document = open_native_part(&mut owner, &directory.path().join("no-face.opd"));
    publish_native_scene(&mut owner, &mut window);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    let original = window.view.camera_state().unwrap().isometric();
    window.view.set_camera(original).unwrap();
    owner
        .create_sketch_for_edit(
            document,
            &DocumentSketchCreation {
                name: "Sketch".into(),
                placement: SketchPlacement::Fixed(SketchPlane::xy()),
                orientation: None,
                project_origin: true,
            },
        )
        .unwrap();
    assert!(!owner.sketch_edit().unwrap().look_at_plane);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    assert_eq!(window.view.camera_state().unwrap(), original);
    owner.finish_sketch_edit().unwrap();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    assert_eq!(window.view.camera_state().unwrap(), original);
    owner.finish().unwrap();
    window.finish().unwrap();
}
