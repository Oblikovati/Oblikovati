// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::HeadWork;
use oblikovati_gui::{GuiFrameEngine, GuiFrameRequest};
use oblikovati_viewport::{GuiFrame, GuiFrameCompiler};
use scene_tests::{capture_native_scene, open_native_part, publish_native_scene, solid_pixels};
#[test]
fn native_owned_imgui_packet_composes_with_the_document_after_context_destruction() {
    let directory = tempfile::tempdir().unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    open_native_part(&mut owner, &directory.path().join("gui-part.opd"));
    let baseline = publish_native_scene(&mut owner, &mut window);
    assert!(solid_pixels(&baseline) > 0);
    let (compiler, packet) = owned_shell_packet(window.drawable_extent().unwrap());
    window.submit_gui_frame(&compiler, &packet).unwrap();
    drop(compiler);
    drop(packet);
    assert!(window.view.capture().is_err());
    let composed = capture_native_scene(&mut window);
    verify_panel_and_document(&baseline, &composed);
    window.clear_gui_frame().unwrap();
    assert!(window.view.capture().is_err());
    assert_eq!(capture_native_scene(&mut window).rgba, baseline.rgba);
    verify_gui_retirement(&mut window);
    owner.finish().unwrap();
}
fn owned_shell_packet(extent: [u32; 2]) -> (GuiFrameCompiler, GuiFrame) {
    let mut engine = GuiFrameEngine::new().unwrap();
    let compiler = GuiFrameCompiler::new(engine.atlas()).unwrap();
    let request = GuiFrameRequest {
        extent,
        events: vec![],
        commands: Default::default(),
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
    };
    // Dear ImGui lays out the new window before it emits its visible contents.
    engine.frame(&request).unwrap();
    let packet = engine.frame(&request).unwrap().draws;
    assert!(!packet.batches.is_empty());
    (compiler, packet)
}
fn verify_panel_and_document(
    baseline: &oblikovati_viewport::ViewportCapture,
    composed: &oblikovati_viewport::ViewportCapture,
) {
    assert_eq!(composed.scene_generation, baseline.scene_generation);
    assert_eq!(composed.camera_generation, baseline.camera_generation);
    assert!(composed.overlay_generation > baseline.overlay_generation);
    assert_eq!(solid_pixels(composed), solid_pixels(baseline));
    let width = baseline.extent[0] as usize;
    let changed: Vec<_> = baseline
        .rgba
        .as_chunks::<4>()
        .0
        .iter()
        .zip(composed.rgba.as_chunks::<4>().0)
        .enumerate()
        .filter(|(_, (before, after))| before != after)
        .map(|(index, _)| [index % width, index / width])
        .collect();
    assert!(changed.len() > 1000, "expected a visible Dear ImGui panel");
    assert!(changed.iter().all(|[x, y]| *x < 320 && *y < 130));
}
fn verify_gui_retirement(window: &mut NativeHeadWindow) {
    let (compiler, packet) = owned_shell_packet(window.drawable_extent().unwrap());
    window.submit_gui_frame(&compiler, &packet).unwrap();
    capture_native_scene(window);
    window.finish().unwrap();
    assert!(window.submit_gui_frame(&compiler, &packet).is_err());
    assert!(window.clear_gui_frame().is_err());
    assert!(window.drawable_extent().is_err());
}
