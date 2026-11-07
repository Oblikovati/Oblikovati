// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::*;
#[cfg(target_os = "linux")]
use oblikovati_viewport::compile_gui_frame;
fn request() -> GuiFrameRequest {
    GuiFrameRequest {
        extent: [640, 360],
        status: None,
        events: vec![GuiInputEvent::PointerMoved([630., 350.])],
        commands: GuiCommandState {
            document_units: None,
            ui_scale: None,
            dimension_draft: None,
            dimension_labels: Vec::new(),
            sketch_selection: Default::default(),
            theme: None,
            color_scheme: None,
            documents: Vec::new(),
            document_model_states: Default::default(),
            document_storage: Default::default(),
            drawing_documents: Default::default(),
            drawing_projections: Default::default(),
            prompts: Default::default(),
            projects: Vec::new(),
            template_sizes: Vec::new(),
            browser: None,
            undo_label: Some("Edit".into()),
            redo_label: Some("Edit".into()),
            transaction_open: false,
            extrude_available: false,
            sketch_tool_keyboard: false,
            dimensions_needed: None,
            sketch_hud: None,
            constraint_glyphs: Vec::new(),
            precise_target: None,
            dimension_tool_active: false,
            takes_command_alias: false,
            center_point_format: false,
            sketch_context: None,
            repeat_tool: None,
            undo_labels: Vec::new(),
            line_command: None,
            sketch_options: None,
            sketch_at_ready: false,
            origin_planes: Vec::new(),
            feature_edges: Vec::new(),
            origin_axes: Vec::new(),
            view_axes: None,
            model_face_chosen: false,
            inspected_file: None,
            document_windows: Vec::new(),
            active_window: None,
            recent_documents: Vec::new(),
            measure: None,
            region_properties: None,
            sketch_delete_available: false,
            sketch_format_available: false,
            sketch_formats: Default::default(),
            authoring: GuiAuthoringContext::Part,
            document_type: Some(oblikovati_api::types::DOCUMENT_PART),
            document: Some(1),
            parameters: Vec::new(),
        },
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
    }
}
#[test]
fn real_imgui_frames_are_owned_and_context_survives_invalid_input() {
    let mut engine = GuiFrameEngine::new().unwrap();
    engine.apply_theme(&oblikovati_model::theme::ThemeDefinition::dark().unwrap());
    let atlas = engine.atlas().clone();
    assert!(!atlas.rgba.is_empty());
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
    #[cfg(target_os = "linux")]
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

const SCALE_SETTLING_FRAMES: usize = 3;
fn verify_scaled_frame(engine: &mut GuiFrameEngine) {
    let baseline = engine.frame(&request()).unwrap().draws;
    let mut input = request();
    // Preserve logical viewport size so responsive panels have the same layout.
    input.extent = input.extent.map(|value| value * 2);
    input.framebuffer_scale = [2.; 2];
    // A new raster scale measures the ribbon's panels again, so the layout settles first.
    for _ in 0..SCALE_SETTLING_FRAMES {
        engine.frame(&input).unwrap();
    }
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
    // #3849: these gestures address expanded controls; ribbon_click also covers their narrow flyouts.
    input.extent[0] = 1440;
    for (x, expected) in [
        (110., GuiShellAction::ChooseSketchPlane),
        (440., GuiShellAction::FinishSketch),
    ] {
        input.commands.authoring = if expected == GuiShellAction::ChooseSketchPlane {
            GuiAuthoringContext::Part
        } else {
            GuiAuthoringContext::Sketch
        };
        input.events = vec![GuiInputEvent::PointerMoved([x, 68.])];
        engine.frame(&input).unwrap();
        input.events = vec![left_button_event(true)];
        engine.frame(&input).unwrap();
        input.events = vec![left_button_event(false)];
        assert_eq!(engine.frame(&input).unwrap().actions, [expected]);
    }
    input.events = vec![GuiInputEvent::PointerMoved([40., 107.])];
    engine.frame(&input).unwrap();
    input.events = vec![left_button_event(true)];
    engine.frame(&input).unwrap();
    input.events = vec![left_button_event(false)];
    assert_eq!(
        engine.frame(&input).unwrap().actions,
        [GuiShellAction::PlaceSketchTool(
            oblikovati_gui::GuiSketchTool::Rectangle
        )]
    );
    let active_document = input.commands.clone();
    input.commands = GuiCommandState::default();
    input.events = vec![GuiInputEvent::PointerMoved([40., 68.])];
    engine.frame(&input).unwrap();
    input.events = vec![left_button_event(true)];
    engine.frame(&input).unwrap();
    input.events = vec![left_button_event(false)];
    assert_eq!(
        engine.frame(&input).unwrap().actions,
        [GuiShellAction::NewPart]
    );
    input.commands = active_document;
    for (x, action) in [
        (111., GuiShellAction::Save),
        (156., GuiShellAction::Undo),
        (201., GuiShellAction::Redo),
        (246., GuiShellAction::ShowHome),
    ] {
        verify_gui_click(engine, &mut input, x, action);
    }
    verify_keyboard_save_activation(engine);
    verify_scaled_frame(engine);
    verify_exclusive_context(engine);
    verify_invalid_requests(engine);
    verify_scaled_input_overflow(engine);
    verify_invalid_event_batch_is_atomic(engine);
    verify_queued_clicks_survive_frame_boundaries(engine);
    verify_owner_command_enabling(engine);
}
fn verify_keyboard_save_activation(engine: &mut GuiFrameEngine) {
    let mut input = request();
    click_control(engine, &mut input, 246.);
    input.events = vec![GuiInputEvent::PointerMoved([600., 300.])];
    engine.frame(&input).unwrap();
    let mut actions = Vec::new();
    for key in [
        GuiKey::LeftArrow,
        GuiKey::LeftArrow,
        GuiKey::LeftArrow,
        GuiKey::Space,
    ] {
        actions.extend(copy_keyboard_actions(engine, &mut input, key));
    }
    input.events.clear();
    actions.extend(engine.frame(&input).unwrap().actions);
    assert_eq!(actions, [GuiShellAction::Save]);
}

fn copy_keyboard_actions(
    engine: &mut GuiFrameEngine,
    input: &mut GuiFrameRequest,
    key: GuiKey,
) -> Vec<GuiShellAction> {
    let mut actions = Vec::new();
    for pressed in [true, false] {
        input.events = vec![GuiInputEvent::Key { key, pressed }];
        let output = engine.frame(input).unwrap();
        if key == GuiKey::LeftArrow && pressed {
            assert!(output.activity.pending_navigation);
            assert!(output.activity.needs_frame());
        }
        actions.extend(output.actions);
    }
    actions
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
        GuiInputEvent::PointerMoved([111., 45.]),
        left_button_event(true),
        GuiInputEvent::Text("bad\0text".into()),
    ];
    assert!(engine.frame(&input).is_err());
    input.events = vec![GuiInputEvent::PointerMoved([111., 45.])];
    engine.frame(&input).unwrap();
    input.events = vec![left_button_event(false)];
    assert!(engine.frame(&input).unwrap().actions.is_empty());
    let output = engine.frame(&request()).unwrap();
    assert!(!output.capture.pointer);
}

fn verify_queued_clicks_survive_frame_boundaries(engine: &mut GuiFrameEngine) {
    let mut input = request();
    input.events = vec![GuiInputEvent::PointerMoved([111., 45.])];
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
        for x in [156., 201.] {
            assert!(click_control(engine, &mut input, x).actions.is_empty());
        }
        assert_eq!(
            click_control(engine, &mut input, 111.).actions,
            [GuiShellAction::Save]
        );
        assert_eq!(
            click_control(engine, &mut input, 246.).actions,
            [GuiShellAction::ShowHome]
        );
    }
}
