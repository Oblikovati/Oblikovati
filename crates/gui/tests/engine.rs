// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::*;
use oblikovati_viewport::compile_gui_frame;
fn request() -> GuiFrameRequest {
    GuiFrameRequest {
        extent: [640, 360],
        events: vec![GuiInputEvent::PointerMoved([0., 0.])],
        commands: GuiCommandState {
            undo_label: Some("Edit".into()),
            redo_label: Some("Edit".into()),
            transaction_open: false,
        },
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
    let output = engine.frame(&input).unwrap();
    let snapshot = output.draws;
    let actions = output.actions;
    assert!(!snapshot.batches.is_empty());
    assert!(actions.is_empty());
    let vertices = snapshot.batches[0].vertices.clone();
    let saved = snapshot.clone();
    exercise_gui_inputs(&mut engine);
    drop(engine);
    let next_owner = GuiFrameEngine::new().unwrap();
    drop(next_owner);
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
    let output = click_control(engine, input, x);
    assert_eq!(output.actions, [expected]);
    assert!(output.capture.pointer);
}
fn click_control(
    engine: &mut GuiFrameEngine,
    input: &mut GuiFrameRequest,
    x: f32,
) -> GuiFrameOutput {
    input.events = vec![GuiInputEvent::PointerMoved([x, 45.])];
    engine.frame(input).unwrap();
    input.events = vec![left_button_event(true)];
    engine.frame(input).unwrap();
    input.events = vec![left_button_event(false)];
    engine.frame(input).unwrap()
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
    input.events = vec![GuiInputEvent::PointerMoved([f32::INFINITY, 0.])];
    assert!(engine.frame(&input).is_err());
    engine.frame(&request()).unwrap();
}

fn verify_scaled_frame(engine: &mut GuiFrameEngine) {
    let baseline = engine.frame(&request()).unwrap().draws;
    let mut input = request();
    input.framebuffer_scale = [2.; 2];
    let snapshot = engine.frame(&input).unwrap().draws;
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
    assert!(
        GuiFrameEngine::new()
            .err()
            .unwrap()
            .contains("exclusive owner activation")
    );
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
    verify_invalid_event_batch_is_atomic(engine);
    verify_queued_clicks_survive_frame_boundaries(engine);
    verify_owner_command_enabling(engine);
}

fn verify_scaled_input_overflow(engine: &mut GuiFrameEngine) {
    let mut input = request();
    input.framebuffer_scale[0] = f32::from_bits(1);
    assert!(engine.frame(&input).is_err());
    input = request();
    input.events = vec![GuiInputEvent::PointerMoved([f32::MAX, 0.])];
    input.framebuffer_scale[0] = 0.5;
    assert!(engine.frame(&input).is_err());
    engine.frame(&request()).unwrap();
}

fn left_button_event(pressed: bool) -> GuiInputEvent {
    GuiInputEvent::PointerButton {
        button: GuiPointerButton::Left,
        pressed,
    }
}

fn verify_invalid_event_batch_is_atomic(engine: &mut GuiFrameEngine) {
    let mut input = request();
    input.events = vec![
        GuiInputEvent::PointerMoved([30., 45.]),
        left_button_event(true),
        GuiInputEvent::Text("bad\0text".into()),
    ];
    assert!(engine.frame(&input).is_err());
    input.events = vec![GuiInputEvent::PointerMoved([30., 45.])];
    engine.frame(&input).unwrap();
    input.events = vec![left_button_event(false)];
    assert!(engine.frame(&input).unwrap().actions.is_empty());
    let output = engine.frame(&request()).unwrap();
    assert!(!output.capture.pointer);
}

fn verify_queued_clicks_survive_frame_boundaries(engine: &mut GuiFrameEngine) {
    let mut input = request();
    input.events = vec![GuiInputEvent::PointerMoved([30., 45.])];
    engine.frame(&input).unwrap();
    input.events = [true, false, true, false]
        .into_iter()
        .map(left_button_event)
        .collect();
    let mut output = engine.frame(&input).unwrap();
    assert!(output.activity.queued_input);
    let mut actions = output.actions;
    input.events.clear();
    let mut attempts = 0;
    while output.activity.queued_input {
        assert!(attempts < 8, "expected the finite click batch to drain");
        output = engine.frame(&input).unwrap();
        actions.extend(output.actions);
        attempts += 1;
    }
    assert_eq!(actions, [GuiShellAction::Save, GuiShellAction::Save]);
    assert!(output.activity.interaction);
    input.events = vec![GuiInputEvent::PointerMoved([500., 300.])];
    assert!(!engine.frame(&input).unwrap().activity.needs_frame());
}

fn verify_owner_command_enabling(engine: &mut GuiFrameEngine) {
    let mut input = request();
    for state in [
        GuiCommandState::default(),
        GuiCommandState {
            transaction_open: true,
            ..input.commands.clone()
        },
    ] {
        input.commands = state;
        for x in [75., 120.] {
            assert!(click_control(engine, &mut input, x).actions.is_empty());
        }
        assert_eq!(
            click_control(engine, &mut input, 30.).actions,
            [GuiShellAction::Save]
        );
        assert_eq!(
            click_control(engine, &mut input, 165.).actions,
            [GuiShellAction::Home]
        );
    }
}
