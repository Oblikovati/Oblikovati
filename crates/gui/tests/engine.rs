// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::*;
use oblikovati_viewport::compile_gui_frame;
fn request() -> GuiFrameRequest {
    GuiFrameRequest {
        extent: [640, 360],
        pointer: [0., 0.],
        left_pressed: false,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
    }
}
#[test]
fn real_imgui_frames_are_owned_and_context_survives_invalid_input() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let atlas = engine.atlas().clone();
    let input = request();
    engine.frame(&input).unwrap();
    let (snapshot, actions) = engine.frame(&input).unwrap();
    assert!(!snapshot.batches.is_empty());
    assert!(actions.is_empty());
    let vertices = snapshot.batches[0].vertices.clone();
    let saved = snapshot.clone();
    exercise_gui_inputs(&mut engine);
    drop(engine);
    assert_eq!(snapshot, saved);
    assert_eq!(snapshot.batches[0].vertices, vertices);
    assert!(
        !compile_gui_frame(&snapshot, &atlas)
            .unwrap()
            .draws()
            .is_empty()
    );
}
fn verify_gui_click(
    engine: &mut GuiFrameEngine,
    input: &mut GuiFrameRequest,
    x: f32,
    expected: GuiShellAction,
) {
    input.pointer = [x, 45.];
    engine.frame(input).unwrap();
    input.left_pressed = true;
    engine.frame(input).unwrap();
    input.left_pressed = false;
    let (_, actions) = engine.frame(input).unwrap();
    assert_eq!(actions, [expected]);
}
fn verify_invalid_requests(engine: &mut GuiFrameEngine) {
    let mut input = request();
    for invalid in [0., -1., f32::NAN] {
        input.delta_seconds = invalid;
        assert!(engine.frame(&input).is_err());
    }
    input = request();
    input.framebuffer_scale = [0., 1.];
    assert!(engine.frame(&input).is_err());
    input = request();
    input.extent[0] = 0;
    assert!(engine.frame(&input).is_err());
    input = request();
    input.pointer[0] = f32::INFINITY;
    assert!(engine.frame(&input).is_err());
    engine.frame(&request()).unwrap();
}

fn verify_scaled_frame(engine: &mut GuiFrameEngine) {
    let (baseline, _) = engine.frame(&request()).unwrap();
    let mut input = request();
    input.framebuffer_scale = [2.; 2];
    let (snapshot, _) = engine.frame(&input).unwrap();
    assert_eq!(snapshot.extent, input.extent);
    assert!(!snapshot.batches.is_empty());
    let original = vertex_positions(&baseline);
    let scaled = vertex_positions(&snapshot);
    assert_eq!(
        scaled,
        original
            .into_iter()
            .map(|point| point.map(|value| value * 2.))
            .collect::<Vec<_>>()
    );
}
fn verify_exclusive_context(engine: &mut GuiFrameEngine) {
    let occupied = imgui::Context::create();
    assert!(
        engine
            .frame(&request())
            .unwrap_err()
            .contains("exclusive owner activation")
    );
    assert!(GuiFrameEngine::new().is_err());
    drop(occupied);
    engine.frame(&request()).unwrap();
}

fn vertex_positions(frame: &GuiFrame) -> Vec<[f32; 2]> {
    frame
        .batches
        .iter()
        .flat_map(|batch| batch.vertices.iter().map(|vertex| vertex.position))
        .collect()
}

fn exercise_gui_inputs(engine: &mut GuiFrameEngine) {
    let mut input = request();
    for (x, action) in [
        (30., GuiShellAction::Save),
        (75., GuiShellAction::Undo),
        (120., GuiShellAction::Redo),
        (165., GuiShellAction::Home),
    ] {
        verify_gui_click(engine, &mut input, x, action);
    }
    verify_scaled_frame(engine);
    verify_exclusive_context(engine);
    verify_invalid_requests(engine);
    verify_scaled_input_overflow(engine);
}

fn verify_scaled_input_overflow(engine: &mut GuiFrameEngine) {
    let mut input = request();
    input.framebuffer_scale[0] = f32::from_bits(1);
    assert!(engine.frame(&input).is_err());
    input = request();
    input.pointer[0] = f32::MAX;
    input.framebuffer_scale[0] = 0.5;
    assert!(engine.frame(&input).is_err());
    engine.frame(&request()).unwrap();
}
