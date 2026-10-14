// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::GuiFrameCompiler;
use gui_fixture::{atlas, frame};
#[test]
fn gui_submission_retains_model_and_refuses_stale_or_failed_captures() {
    let (runtime, recorder) = runtime();
    let mut view = runtime.open_offscreen([64, 48]).unwrap();
    view.submit(&scene()).unwrap();
    let baseline = view.redraw().unwrap();
    let packet = frame([64, 48], [255; 4], [0., 0., 64., 48.]);
    let compiler = GuiFrameCompiler::new(&atlas()).unwrap();
    view.submit_gui_frame(&compiler, &packet).unwrap();
    assert!(view.capture().is_err());
    let drawn = view.redraw().unwrap();
    assert_eq!(drawn.scene_generation, baseline.scene_generation);
    assert_eq!(drawn.camera_generation, baseline.camera_generation);
    assert_eq!(drawn.overlay_generation, baseline.overlay_generation + 1);
    assert_gui_layers(&recorder);
    verify_gui_failure(&mut view, &recorder);
    view.clear_gui_frame().unwrap();
    assert!(view.capture().is_err());
    view.redraw().unwrap();
    assert_eq!(
        recorder
            .state
            .lock()
            .unwrap()
            .frames
            .last()
            .unwrap()
            .draws()
            .len(),
        1
    );
}
fn assert_gui_layers(recorder: &RecordingDevice) {
    let state = recorder.state.lock().unwrap();
    let frame = state.frames.last().unwrap();
    assert_eq!(frame.draws().len(), 2);
    assert_eq!(
        frame.layers().last().unwrap().depth_policy(),
        (false, false, false)
    );
    assert!(frame.draws().last().unwrap().textured_style().is_some());
}
fn verify_gui_failure(view: &mut Viewport, recorder: &RecordingDevice) {
    recorder.state.lock().unwrap().fail = true;
    assert!(view.redraw().unwrap_err().contains("recorded device fault"));
    assert!(view.capture().is_err());
    recorder.state.lock().unwrap().fail = false;
    view.redraw().unwrap();
}
#[test]
fn malformed_gui_extent_and_exhausted_generation_leave_the_old_packet_intact() {
    let (runtime, _) = runtime();
    let mut view = runtime.open_offscreen([64, 48]).unwrap();
    let mut packet = frame([64, 48], [255; 4], [0., 0., 64., 48.]);
    let compiler = GuiFrameCompiler::new(&atlas()).unwrap();
    view.submit_gui_frame(&compiler, &packet).unwrap();
    let before = view.redraw().unwrap();
    packet.batches[0].indices[0] = 99;
    assert!(view.submit_gui_frame(&compiler, &packet).is_err());
    assert_eq!(view.capture().unwrap(), before);
    packet.batches[0].indices[0] = 0;
    packet.extent = [32, 48];
    assert!(view.submit_gui_frame(&compiler, &packet).is_err());
    assert_eq!(view.capture().unwrap(), before);
    packet.extent = [64, 48];
    view.overlay = u64::MAX;
    assert!(view.submit_gui_frame(&compiler, &packet).is_err());
    assert!(view.clear_gui_frame().is_err());
    view.overlay = before.overlay_generation;
    assert_eq!(view.capture().unwrap(), before);
    view.close();
    assert!(view.submit_gui_frame(&compiler, &packet).is_err());
    assert!(view.clear_gui_frame().is_err());
}
#[test]
fn resize_requires_a_packet_for_the_current_target_without_rescaling_old_input() {
    let (runtime, recorder) = runtime();
    let mut view = runtime.open_offscreen([64, 48]).unwrap();
    let compiler = GuiFrameCompiler::new(&atlas()).unwrap();
    let packet = frame([64, 48], [255; 4], [0., 0., 64., 48.]);
    view.submit_gui_frame(&compiler, &packet).unwrap();
    view.redraw().unwrap();
    view.resize([80, 60]).unwrap();
    assert!(view.capture().is_err());
    assert!(view.redraw().unwrap_err().contains("GUI"));
    assert!(view.submit_gui_frame(&compiler, &packet).is_err());
    assert_eq!(recorder.state.lock().unwrap().frames.len(), 1);
    let fresh = frame([80, 60], [255; 4], [0., 0., 80., 60.]);
    view.submit_gui_frame(&compiler, &fresh).unwrap();
    assert_eq!(view.redraw().unwrap().extent, [80, 60]);
    assert_eq!(recorder.state.lock().unwrap().frames.len(), 2);
}

#[test]
fn drawable_extent_observes_external_resize_and_invalidates_the_completed_frame() {
    let (runtime, _) = runtime();
    let mut view = runtime.open_offscreen([64, 48]).unwrap();
    let before = view.redraw().unwrap();
    assert_eq!(view.drawable_extent().unwrap(), [64, 48]);
    assert_eq!(view.capture().unwrap(), before);
    let Some(ViewTarget::Offscreen(window)) = view.target.as_ref() else {
        panic!("expected an offscreen target");
    };
    window.set_size(80, 60);
    assert_eq!(view.drawable_extent().unwrap(), [80, 60]);
    assert!(view.capture().is_err());
    let after = view.redraw().unwrap();
    assert!(after.surface_generation > before.surface_generation);
    assert_eq!(after.scene_generation, before.scene_generation);
    view.close();
    assert!(view.drawable_extent().is_err());
}
