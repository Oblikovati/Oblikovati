// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use gui_fixture::{atlas, frame};
use oblikovati_viewport::{GuiFrame, GuiFrameCompiler, ViewportNavigation};
#[test]
fn native_gui_composes_over_model_and_replaces_clears_resizes_and_retires() {
    let runtime = ViewportRuntime::new(Default::default()).unwrap();
    let mut view = open_fixture(&runtime, "Native GUI composition");
    let baseline = verify_native(&mut view);
    let compiler = GuiFrameCompiler::new(&atlas()).unwrap();
    let packet = frame([320, 240], [0, 0, 255, 128], [148., 108., 172., 132.]);
    view.submit_gui_frame(&compiler, &packet).unwrap();
    assert!(view.capture().is_err());
    let drawn = gui_capture(&mut view);
    verify_gui_pixels(&baseline, &drawn);
    verify_order_and_replacement(&mut view, &compiler, &packet);
    view.clear_gui_frame().unwrap();
    assert!(view.capture().is_err());
    assert_eq!(gui_capture(&mut view).rgba, baseline.rgba);
    verify_gui_resize(&mut view, &compiler, &packet);
    drop(compiler);
    gui_capture(&mut view);
    view.close();
    assert!(view.clear_gui_frame().is_err());
    assert!(
        runtime.validation_errors().is_empty(),
        "{:?}",
        runtime.validation_errors()
    );
}
fn gui_capture(view: &mut Viewport) -> ViewportCapture {
    let capture = complete_surface_frame(view);
    await_presented_pixels(view, &capture);
    capture
}
fn pixel(capture: &ViewportCapture, [x, y]: [usize; 2]) -> [u8; 4] {
    let start = (y * capture.extent[0] as usize + x) * 4;
    capture.rgba[start..start + 4].try_into().unwrap()
}
fn verify_gui_pixels(baseline: &ViewportCapture, drawn: &ViewportCapture) {
    assert_eq!(pixel(baseline, [160, 120]), [255, 0, 0, 255]);
    for (actual, expected) in pixel(drawn, [160, 120]).into_iter().zip([187, 0, 188, 255]) {
        assert!(
            actual.abs_diff(expected) <= 1,
            "{actual}: expected {expected}"
        );
    }
    for point in [[147, 120], [172, 120], [160, 107], [160, 132]] {
        assert_eq!(pixel(drawn, point), pixel(baseline, point));
    }
    assert_eq!(drawn.scene_generation, baseline.scene_generation);
    assert_eq!(drawn.camera_generation, baseline.camera_generation);
    assert_eq!(drawn.surface_generation, baseline.surface_generation);
    assert!(drawn.overlay_generation > baseline.overlay_generation);
}
fn verify_order_and_replacement(
    view: &mut Viewport,
    compiler: &GuiFrameCompiler,
    packet: &GuiFrame,
) {
    let mut ordered = packet.clone();
    ordered
        .batches
        .extend(frame([320, 240], [255; 4], [155., 115., 165., 125.]).batches);
    view.submit_gui_frame(compiler, &ordered).unwrap();
    assert_eq!(pixel(&gui_capture(view), [160, 120]), [255; 4]);
    let replacement = frame([320, 240], [0, 255, 0, 255], [148., 108., 172., 132.]);
    view.submit_gui_frame(compiler, &replacement).unwrap();
    let accepted = gui_capture(view);
    assert_eq!(pixel(&accepted, [160, 120]), [0, 255, 0, 255]);
    let mut invalid = replacement;
    invalid.batches[0].clip_rect[0] = f32::NAN;
    assert!(view.submit_gui_frame(compiler, &invalid).is_err());
    assert_eq!(view.capture().unwrap(), accepted);
    assert_eq!(gui_capture(view).rgba, accepted.rgba);
}
fn verify_gui_resize(view: &mut Viewport, compiler: &GuiFrameCompiler, packet: &GuiFrame) {
    view.submit_gui_frame(compiler, packet).unwrap();
    view.resize([400, 300]).unwrap();
    view.poll_events(64).unwrap();
    assert!(view.capture().is_err());
    assert!(view.redraw().is_err());
    assert!(view.submit_gui_frame(compiler, packet).is_err());
    let fresh = frame(
        view.drawable_extent().unwrap(),
        [0, 255, 0, 255],
        [188., 138., 212., 162.],
    );
    view.submit_gui_frame(compiler, &fresh).unwrap();
    assert_eq!(pixel(&gui_capture(view), [200, 150]), [0, 255, 0, 255]);
    view.set_mapped(false).unwrap();
    assert!(view.present().is_err());
    assert!(view.redraw().is_err());
    assert!(view.capture().is_err());
    view.set_mapped(true).unwrap();
    gui_capture(view);
}
#[test]
fn gui_ownership_survives_compiler_drop_and_does_not_enter_camera_fit() {
    let runtime = ViewportRuntime::new(Default::default()).unwrap();
    let mut view = open_fixture(&runtime, "GUI ownership and Fit");
    let initial = view.camera_state().unwrap();
    view.navigate(ViewportNavigation::Fit { margin: 0.1 })
        .unwrap();
    let expected = view.camera_state().unwrap();
    view.set_camera(initial).unwrap();
    let compiler = GuiFrameCompiler::new(&atlas()).unwrap();
    let mut packet = frame([320, 240], [255; 4], [-50000., -50000., 50000., 50000.]);
    view.submit_gui_frame(&compiler, &packet).unwrap();
    packet.batches.clear();
    drop(compiler);
    view.navigate(ViewportNavigation::Fit { margin: 0.1 })
        .unwrap();
    assert_eq!(view.camera_state().unwrap(), expected);
    assert_eq!(pixel(&gui_capture(&mut view), [160, 120]), [255; 4]);
    view.clear_gui_frame().unwrap();
    verify_native(&mut view);
    view.close();
    assert!(runtime.validation_errors().is_empty());
}
