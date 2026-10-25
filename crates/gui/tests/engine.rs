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
    assert!(snapshot.textures.len() >= 4);
    assert!(snapshot.batches.iter().any(|batch| batch.texture_id >= 100));
    assert!(actions.is_empty());
    let vertices = snapshot.batches[0].vertices.clone();
    let saved = snapshot.clone();
    exercise_gui_inputs(&mut engine);
    drop(engine);
    let next_owner = GuiFrameEngine::new().unwrap();
    drop(next_owner);
    verify_semantic_ribbon_target();
    assert_eq!(snapshot, saved);
    assert_eq!(snapshot.batches[0].vertices, vertices);
    assert!(
        !compile_gui_frame(&snapshot, &atlas)
            .unwrap()
            .draws()
            .is_empty()
    );
}

fn verify_semantic_ribbon_target() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut input = semantic_ribbon_request();
    let target = engine
        .frame(&input)
        .unwrap()
        .targets
        .into_iter()
        .find(|target| target.command_id == "GetStarted.NewPart")
        .unwrap();
    input.events = vec![GuiInputEvent::PointerMoved(target.center)];
    engine.frame(&input).unwrap();
    input.events = vec![left_button_event(true)];
    let mut actions = engine.frame(&input).unwrap().actions;
    input.events = vec![left_button_event(false)];
    actions.extend(engine.frame(&input).unwrap().actions);
    assert_eq!(actions, [GuiShellAction::NewPart]);
}

fn semantic_ribbon_request() -> GuiFrameRequest {
    let control = GuiRibbonControl {
        command_id: "GetStarted.NewPart".into(),
        display_name: "New Part".into(),
        icon: "new-part".into(),
        button_style: 2,
        enabled: true,
        active: false,
    };
    GuiFrameRequest {
        extent: [640, 360],
        events: Vec::new(),
        commands: GuiCommandState {
            ribbon_key: "ZeroDoc".into(),
            ribbon_tabs: vec![GuiRibbonTab {
                name: "Get Started".into(),
                panels: vec![GuiRibbonPanel {
                    name: "Launch".into(),
                    controls: vec![control],
                }],
            }],
            ..Default::default()
        },
        status: None,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
    }
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
