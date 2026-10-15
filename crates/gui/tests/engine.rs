// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::*;
use oblikovati_viewport::compile_gui_frame;
fn request() -> GuiFrameRequest {
    GuiFrameRequest {
        extent: [640, 360],
        status: None,
        events: vec![GuiInputEvent::PointerMoved([0., 0.])],
        commands: GuiCommandState {
            undo_label: Some("Edit".into()),
            redo_label: Some("Edit".into()),
            transaction_open: false,
            ..Default::default()
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
    assert_eq!(snapshot.textures.len(), 4);
    assert!(snapshot.batches.iter().any(|batch| batch.texture_id >= 100));
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
fn verify_invalid_requests(engine: &mut GuiFrameEngine) {
    let mut status_request = request();
    status_request.status = Some("Saved\0part".into());
    status_request.events = vec![left_button_event(true)];
    let error = engine.frame(&status_request).unwrap_err();
    assert!(error.contains("status") && error.contains("NUL"));
    status_request.status = Some("Saved part".into());
    status_request.events.clear();
    let status_frame = engine.frame(&status_request).unwrap();
    assert!(status_frame.actions.is_empty());
    assert_ne!(status_frame.draws, engine.frame(&request()).unwrap().draws);
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
    engine.frame(&request()).unwrap();
    let mut input = request();
    input.framebuffer_scale = [2.; 2];
    let snapshot = engine.frame(&input).unwrap().draws;
    assert_eq!(snapshot.extent, input.extent);
    assert!(!snapshot.batches.is_empty());
    assert!(
        vertex_positions(&snapshot)
            .iter()
            .flatten()
            .all(|value| value.is_finite())
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
    select_ribbon_tab(engine, &mut input, [25., 38.]);
    for (x, action) in [
        (100., GuiShellAction::Save),
        (145., GuiShellAction::Undo),
        (190., GuiShellAction::Redo),
    ] {
        verify_gui_click_at(engine, &mut input, [x, 60.], action);
    }
    verify_cube_actions(engine);
    verify_scaled_frame(engine);
    verify_exclusive_context(engine);
    verify_invalid_requests(engine);
    verify_scaled_input_overflow(engine);
    verify_invalid_event_batch_is_atomic(engine);
    verify_queued_clicks_survive_frame_boundaries(engine);
    verify_owner_command_enabling(engine);
}
fn verify_cube_actions(engine: &mut GuiFrameEngine) {
    let mut input = request();
    verify_gui_click_at(engine, &mut input, [40., 60.], GuiShellAction::NewPart);
    select_ribbon_tab(engine, &mut input, [90., 38.]);
    verify_gui_click_at(engine, &mut input, [40., 60.], GuiShellAction::SketchXY);
    verify_gui_click_at(engine, &mut input, [115., 60.], GuiShellAction::Rectangle);
    select_ribbon_tab(engine, &mut input, [135., 38.]);
    verify_gui_click_at(engine, &mut input, [40., 60.], GuiShellAction::Extrude);
    select_ribbon_tab(engine, &mut input, [200., 38.]);
    verify_gui_click_at(engine, &mut input, [40., 60.], GuiShellAction::Home);
}

fn verify_gui_click_at(
    engine: &mut GuiFrameEngine,
    input: &mut GuiFrameRequest,
    point: [f32; 2],
    action: GuiShellAction,
) {
    let output = click_control_at(engine, input, point);
    assert_eq!(output.actions, [action], "ribbon point {point:?}");
}

fn select_ribbon_tab(engine: &mut GuiFrameEngine, input: &mut GuiFrameRequest, point: [f32; 2]) {
    assert!(click_control_at(engine, input, point).actions.is_empty());
}

fn click_control_at(
    engine: &mut GuiFrameEngine,
    input: &mut GuiFrameRequest,
    point: [f32; 2],
) -> GuiFrameOutput {
    input.events = vec![GuiInputEvent::PointerMoved(point)];
    engine.frame(input).unwrap();
    input.events = vec![left_button_event(true)];
    let mut down = engine.frame(input).unwrap();
    input.events = vec![left_button_event(false)];
    let mut output = engine.frame(input).unwrap();
    down.actions.append(&mut output.actions);
    output.actions = down.actions;
    output
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
    input.events = vec![GuiInputEvent::PointerMoved([500., 300.])];
    engine.frame(&input).unwrap();
    input.events = vec![left_button_event(false)];
    assert!(engine.frame(&input).unwrap().actions.is_empty());
    let mut idle = request();
    idle.events = vec![GuiInputEvent::PointerMoved([500., 300.])];
    let output = engine.frame(&idle).unwrap();
    assert!(!output.capture.pointer);
}

fn verify_queued_clicks_survive_frame_boundaries(engine: &mut GuiFrameEngine) {
    let mut input = request();
    select_ribbon_tab(engine, &mut input, [25., 38.]);
    input.events = vec![GuiInputEvent::PointerMoved([100., 60.])];
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
        select_ribbon_tab(engine, &mut input, [25., 38.]);
        for x in [145., 190.] {
            assert!(
                click_control_at(engine, &mut input, [x, 60.])
                    .actions
                    .is_empty()
            );
        }
        assert_eq!(
            click_control_at(engine, &mut input, [100., 60.]).actions,
            [GuiShellAction::Save]
        );
        select_ribbon_tab(engine, &mut input, [200., 38.]);
        assert_eq!(
            click_control_at(engine, &mut input, [40., 60.]).actions,
            [GuiShellAction::Home]
        );
    }
}
