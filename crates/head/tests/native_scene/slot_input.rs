// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::placement_test_inputs::*;
use oblikovati_model::sketch::{SketchPlacementRecipe, SketchPlane};
use oblikovati_viewport::{ViewportEvent, ViewportKey};
use std::collections::VecDeque;
pub(super) struct PlacementEventWindow {
    pub(super) native: NativeHeadWindow,
    pub(super) scripted: VecDeque<Vec<ViewportEvent>>,
}
impl HeadWindow for PlacementEventWindow {
    fn submit_sketch_scene(&mut self, update: crate::HeadSketchSceneUpdate) -> Result<(), String> {
        self.native.submit_sketch_scene(update)
    }

    fn set_sketch_edit(&mut self, edit: Option<crate::HeadSketchEdit>) -> Result<bool, String> {
        self.native.set_sketch_edit(edit)
    }

    fn set_sketch_preview(
        &mut self,
        preview: Option<&oblikovati_viewport::ViewportSketchPreview>,
    ) -> Result<bool, String> {
        self.native.set_sketch_preview(preview)
    }

    fn process_view_input(&mut self, event: &ViewportEvent) -> Result<bool, String> {
        self.native.process_view_input(event)
    }

    fn sketch_point(
        &self,
        pixel: [i32; 2],
        plane: SketchPlane,
    ) -> Result<Option<[f64; 2]>, String> {
        self.native.sketch_point(pixel, plane)
    }
    fn show_command_notice(&mut self, notice: crate::DesktopCommandNotice) -> Result<(), String> {
        self.native.show_command_notice(notice)
    }
    fn submit_scene(&mut self, update: crate::HeadSceneUpdate) -> Result<(), String> {
        self.native.submit_scene(update)
    }
    fn events(&mut self, limit: usize) -> Result<Vec<ViewportEvent>, String> {
        let mut events = self.native.events(limit)?;
        events.extend(self.scripted.pop_front().unwrap_or_default());
        Ok(events)
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        self.native.present()
    }
    fn wait(&mut self, timeout: Option<std::time::Duration>) -> Result<bool, String> {
        self.native.wait(timeout)
    }
    fn finish(&mut self) -> Result<(), String> {
        self.native.finish()
    }
}
#[test]
fn native_slot_input_loop_preserves_preview_and_commits_click_and_drag_identically() {
    let clicked = run_slot_input(false);
    let dragged = run_slot_input(true);
    assert!(clicked.same_recipe(&dragged));
}
fn run_slot_input(drag: bool) -> oblikovati_model::document_sketch::DocumentSketches {
    let directory = tempfile::tempdir().unwrap();
    let mut native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    let document = open_native_part(&mut owner, &directory.path().join("slot-input.opd"));
    let original = publish_native_scene(&mut owner, &mut native);
    owner
        .start_slot_placement(document, fixture::SKETCH, 0.5)
        .unwrap();
    let mut window = PlacementEventWindow {
        native,
        scripted: VecDeque::new(),
    };
    let preview = drive_slot_preview(&mut owner, &mut window, document, drag);
    let after = drive_slot_commit(&mut owner, &mut window, document, drag, &preview);
    drive_slot_undo(&mut owner, &mut window, &original);
    owner.finish().unwrap();
    window.finish().unwrap();
    after
}
fn drive_slot_preview(
    owner: &mut NativeDesktopOwner,
    window: &mut PlacementEventWindow,
    document: u64,
    drag: bool,
) -> SketchPlacementRecipe {
    let before = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .clone();
    let mut first = vec![button([650, 430], true)];
    if !drag {
        first.push(button([650, 430], false));
    }
    window.scripted = VecDeque::from([first, vec![motion([820, 500])]]);
    crate::frames::run_frames(window, HeadOptions { frames: Some(2) }, owner).unwrap();
    let preview = owner.placement_preview().unwrap().clone();
    verify_outline_pixels(owner, window);
    assert!(
        before.same_recipe(
            &owner
                .documents()
                .workspace()
                .session(document)
                .unwrap()
                .snapshot()
                .sketches
        )
    );
    assert!(solid_pixels(&capture_native_scene(&mut window.native)) > 0);
    assert!(
        window
            .native
            .view
            .native_title()
            .unwrap()
            .contains("second centre")
    );
    preview
}
fn drive_slot_commit(
    owner: &mut NativeDesktopOwner,
    window: &mut PlacementEventWindow,
    document: u64,
    drag: bool,
    preview: &SketchPlacementRecipe,
) -> oblikovati_model::document_sketch::DocumentSketches {
    window.scripted = VecDeque::from([
        vec![button([820, 500], !drag)],
        vec![button([820, 500], false)],
    ]);
    crate::frames::run_frames(window, HeadOptions { frames: Some(2) }, owner).unwrap();
    let sketches = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .clone();
    let geometry = sketches.get(fixture::SKETCH).unwrap().geometry();
    let points = &geometry.points()[geometry.points().len() - preview.points.len()..];
    for (actual, expected) in points.iter().zip(&preview.points) {
        for (a, b) in actual.position.into_iter().zip(expected) {
            assert!((a - b).abs() < 1e-8);
        }
    }
    assert!(owner.placement_active());
    assert!(owner.placement_preview().is_none());
    assert!(!window.native.set_sketch_preview(None).unwrap());
    assert_eq!(solid_pixels(&capture_native_scene(&mut window.native)), 0);
    sketches
}
fn drive_slot_undo(
    owner: &mut NativeDesktopOwner,
    window: &mut PlacementEventWindow,
    original: &ViewportCapture,
) {
    window.scripted = VecDeque::from([vec![key(ViewportKey::Undo, true)]]);
    crate::frames::run_frames(window, HeadOptions { frames: Some(2) }, owner).unwrap();
    assert!(!owner.placement_active());
    window.native.view.set_status_text("").unwrap();
    assert_eq!(capture_native_scene(&mut window.native).rgba, original.rgba);
}

#[test]
fn native_slot_clicks_and_camera_changes_in_one_batch_keep_input_order() {
    let directory = tempfile::tempdir().unwrap();
    let mut native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    let document = open_native_part(&mut owner, &directory.path().join("ordered-input.opd"));
    publish_native_scene(&mut owner, &mut native);
    owner
        .start_slot_placement(document, fixture::SKETCH, 0.5)
        .unwrap();
    let plane = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .resolve_sketch_plane(fixture::SKETCH)
        .unwrap();
    let first = native.sketch_point([650, 430], plane).unwrap().unwrap();
    let before = native.view.camera_state().unwrap();
    let events = vec![
        button([650, 430], true),
        button([650, 430], false),
        ViewportEvent::Input(oblikovati_viewport::ViewportInput::Wheel {
            notches: 2,
            position: [900, 500],
        }),
        button([820, 500], true),
    ];
    let mut window = PlacementEventWindow {
        native,
        scripted: VecDeque::from([events]),
    };
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    assert_ne!(window.native.view.camera_state().unwrap(), before);
    verify_ordered_slot_centers(&owner, &window, document, plane, first);
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn verify_ordered_slot_centers(
    owner: &NativeDesktopOwner,
    window: &PlacementEventWindow,
    document: u64,
    plane: SketchPlane,
    first: [f64; 2],
) {
    let second = window
        .native
        .sketch_point([820, 500], plane)
        .unwrap()
        .unwrap();
    let geometry = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(fixture::SKETCH)
        .unwrap()
        .geometry();
    let points = geometry.points();
    for (actual, expected) in [
        (points[points.len() - 1].position, first),
        (points[points.len() - 2].position, second),
    ] {
        for (a, b) in actual.into_iter().zip(expected) {
            assert!((a - b).abs() < 1e-8, "{actual:?} != {expected:?}");
        }
    }
}

pub(super) fn verify_outline_pixels(owner: &NativeDesktopOwner, window: &mut PlacementEventWindow) {
    let outlined = capture_native_scene(&mut window.native);
    assert!(window.native.set_sketch_preview(None).unwrap());
    let cleared = capture_native_scene(&mut window.native);
    assert_ne!(outlined.rgba, cleared.rgba);
    assert_eq!(outlined.scene_generation, cleared.scene_generation);
    window
        .native
        .set_sketch_preview(owner.sketch_preview().as_ref())
        .unwrap();
    assert_eq!(capture_native_scene(&mut window.native).rgba, outlined.rgba);
}
#[test]
fn native_escape_clears_outline_without_editing_document() {
    let directory = tempfile::tempdir().unwrap();
    let mut native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    let document = open_native_part(&mut owner, &directory.path().join("slot-cancel.opd"));
    publish_native_scene(&mut owner, &mut native);
    let original = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .clone();
    owner
        .start_slot_placement(document, fixture::SKETCH, 0.5)
        .unwrap();
    let mut window = PlacementEventWindow {
        native,
        scripted: VecDeque::new(),
    };
    drive_slot_preview(&mut owner, &mut window, document, false);
    window.scripted = VecDeque::from([vec![key(ViewportKey::Cancel, true)]]);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    assert!(owner.sketch_preview().is_none());
    assert!(!window.native.set_sketch_preview(None).unwrap());
    assert!(
        original.same_recipe(
            &owner
                .documents()
                .workspace()
                .session(document)
                .unwrap()
                .snapshot()
                .sketches
        )
    );
    owner.finish().unwrap();
    window.finish().unwrap();
}

#[test]
fn native_preview_recovery_removes_only_the_preview_failure_notice() {
    let mut window = NativeHeadWindow::open().unwrap();
    let preview = oblikovati_viewport::ViewportSketchPreview {
        solve_options: Default::default(),
        plane: SketchPlane::xy(),
        recipe: SketchPlacementRecipe::straight_slot([[0., 0.], [5., 0.]], 2.).unwrap(),
    };
    window
        .show_command_notice(crate::DesktopCommandNotice::PreviewFailed {
            cause: "sample budget".into(),
        })
        .unwrap();
    window.set_sketch_preview(None).unwrap();
    assert!(window.command_notice.is_some());
    assert!(window.set_sketch_preview(Some(&preview)).unwrap());
    assert!(window.command_notice.is_none());
    let notice = crate::DesktopCommandNotice::PlacementPrompt {
        message: "Choose a centre".into(),
    };
    window.show_command_notice(notice.clone()).unwrap();
    assert!(!window.set_sketch_preview(Some(&preview)).unwrap());
    assert_eq!(window.command_notice, Some(notice));
    window.finish().unwrap();
}
