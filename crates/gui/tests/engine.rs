// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::*;
#[cfg(target_os = "linux")]
use oblikovati_viewport::compile_gui_frame;
#[path = "support/engine_control_points.rs"]
mod engine_control_points;
use engine_control_points::{quick_access_point, ribbon_button_point};

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
            sketch_block_names: Vec::new(),
            sketch_block_created: None,
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
            repeat_command: None,
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
            ..Default::default()
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
    let [left, top, right, bottom] = output.document_area.unwrap();
    assert!(left > 0 && top > 0 && right <= input.extent[0] && bottom <= input.extent[1]);
    assert!(left < right && top < bottom);
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
    label: &str,
    expected: GuiShellAction,
) {
    let point = quick_access_point(engine, input, label);
    let output = click_control(engine, input, point);
    assert_eq!(output.actions, [expected]);
    assert!(output.capture.pointer);
}
fn click_control(
    engine: &mut GuiFrameEngine,
    input: &mut GuiFrameRequest,
    point: [f32; 2],
) -> GuiFrameOutput {
    input.events = vec![GuiInputEvent::PointerMoved(point)];
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
    let mut input = request();
    // At narrow widths ImGui truncates tab text based on rasterized font metrics.
    // Use enough room to compare the same visible glyphs at both scales.
    input.extent[0] = 1440;
    for _ in 0..SCALE_SETTLING_FRAMES {
        engine.frame(&input).unwrap();
    }
    let baseline = engine.frame(&input).unwrap().draws;
    // Preserve logical viewport size so responsive panels have the same layout.
    input.extent = input.extent.map(|value| value * 2);
    input.framebuffer_scale = [2.; 2];
    input.events = vec![GuiInputEvent::PointerMoved([1260., 700.])];
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
fn verify_narrow_backing_click(engine: &mut GuiFrameEngine) {
    for scale in [1., 2.] {
        let mut input = request();
        input.commands = GuiCommandState::default();
        input.extent = [(640. * scale) as u32, (360. * scale) as u32];
        input.framebuffer_scale = [scale; 2];
        input.events = vec![GuiInputEvent::PointerMoved([40. * scale, 68. * scale])];
        for _ in 0..SCALE_SETTLING_FRAMES {
            engine.frame(&input).unwrap();
        }
        input.events = vec![left_button_event(true)];
        engine.frame(&input).unwrap();
        input.events = vec![left_button_event(false)];
        assert_eq!(
            engine.frame(&input).unwrap().actions,
            [GuiShellAction::NewPart]
        );
    }
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
    verify_ribbon_dispatch(engine, &mut input);
    verify_quick_access_dispatch(engine, &mut input);
    verify_keyboard_save_activation(engine);
    verify_scaled_frame(engine);
    verify_narrow_backing_click(engine);
    verify_exclusive_context(engine);
    verify_invalid_requests(engine);
    verify_scaled_input_overflow(engine);
    verify_invalid_event_batch_is_atomic(engine);
    verify_queued_clicks_survive_frame_boundaries(engine);
    verify_owner_command_enabling(engine);
}

fn verify_ribbon_dispatch(engine: &mut GuiFrameEngine, input: &mut GuiFrameRequest) {
    use GuiShellAction::{ChooseSketchPlane, FinishSketch, NewPart, PlaceSketchTool};
    // #3849: Keep the commands inline. The dedicated ribbon suite checks collapsed groups.
    input.extent[0] = 1920;
    let part = input.commands.clone();
    let mut sketch = part.clone();
    sketch.authoring = GuiAuthoringContext::Sketch;
    for (state, label, action) in [
        (part, "Start2DSketch", ChooseSketchPlane),
        (sketch.clone(), "FinishSketch", FinishSketch),
        (
            sketch.clone(),
            "Rectangle",
            PlaceSketchTool(GuiSketchTool::Rectangle),
        ),
        (GuiCommandState::default(), "NewPart", NewPart),
    ] {
        input.commands = state;
        let point = ribbon_button_point(engine, input, label);
        assert_eq!(click_control(engine, input, point).actions, [action]);
    }
    input.commands = sketch;
}

fn verify_quick_access_dispatch(engine: &mut GuiFrameEngine, input: &mut GuiFrameRequest) {
    for (label, action) in [
        ("Save", GuiShellAction::Save),
        ("Undo", GuiShellAction::Undo),
        ("Redo", GuiShellAction::Redo),
        ("Home", GuiShellAction::ShowHome),
    ] {
        verify_gui_click(engine, input, label, action);
    }
}
fn verify_keyboard_save_activation(engine: &mut GuiFrameEngine) {
    let mut input = request();
    let home = quick_access_point(engine, &mut input, "Home");
    click_control(engine, &mut input, home);
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
            assert!(
                output.capture.keyboard,
                "a queued GUI navigation must own its arrow"
            );
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
    let save = quick_access_point(engine, &mut input, "Save");
    input.events = vec![
        GuiInputEvent::PointerMoved(save),
        left_button_event(true),
        GuiInputEvent::Text("bad\0text".into()),
    ];
    assert!(engine.frame(&input).is_err());
    input.events = vec![GuiInputEvent::PointerMoved(save)];
    engine.frame(&input).unwrap();
    input.events = vec![left_button_event(false)];
    assert!(engine.frame(&input).unwrap().actions.is_empty());
    let output = engine.frame(&request()).unwrap();
    assert!(!output.capture.pointer);
}

fn verify_queued_clicks_survive_frame_boundaries(engine: &mut GuiFrameEngine) {
    let mut input = request();
    let save = quick_access_point(engine, &mut input, "Save");
    input.events = vec![GuiInputEvent::PointerMoved(save)];
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
        for label in ["Undo", "Redo"] {
            let point = quick_access_point(engine, &mut input, label);
            assert!(click_control(engine, &mut input, point).actions.is_empty());
        }
        verify_gui_click(engine, &mut input, "Save", GuiShellAction::Save);
        verify_gui_click(engine, &mut input, "Home", GuiShellAction::ShowHome);
    }
}

#[test]
fn accessible_frame_focus_moves_between_real_buttons() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = request();
    request.events.clear();
    engine.frame(&request).unwrap();
    for index in 0..3 {
        engine.focus_accessible(GuiAccessibleFocus::Frame(index));
        let mut focused = None;
        for _ in 0..4 {
            focused = engine.frame(&request).unwrap().window_controls.focused;
        }
        assert_eq!(focused, Some(index));
    }
}

#[test]
fn focused_accessible_maximize_responds_to_enter_and_space() {
    for key in [GuiKey::Enter, GuiKey::Space] {
        let mut engine = GuiFrameEngine::new().unwrap();
        let mut request = request();
        request.events.clear();
        engine.frame(&request).unwrap();
        engine.focus_accessible(GuiAccessibleFocus::Frame(1));
        engine.frame(&request).unwrap();
        engine.frame(&request).unwrap();
        request.events = vec![GuiInputEvent::Key { key, pressed: true }];
        assert_eq!(
            engine.frame(&request).unwrap().window_action,
            Some(GuiWindowAction::ToggleMaximize)
        );
        request.events = vec![GuiInputEvent::Key {
            key,
            pressed: false,
        }];
        assert_eq!(engine.frame(&request).unwrap().window_action, None);
    }
}

#[test]
fn queued_window_actions_are_not_consumed_by_tool_closure() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut input = request();
    input.events.clear();
    engine.set_parameters_open(true);
    engine.frame(&input).unwrap();
    engine.focus_accessible(GuiAccessibleFocus::Frame(1));
    engine.frame(&input).unwrap();
    engine.frame(&input).unwrap();
    engine.set_parameters_open(false);
    input.events = [true, false, true, false]
        .into_iter()
        .map(|pressed| GuiInputEvent::Key {
            key: GuiKey::Space,
            pressed,
        })
        .collect();
    let mut output = engine.frame(&input).unwrap();
    assert_eq!(output.window_action, Some(GuiWindowAction::ToggleMaximize));
    assert!(output.activity.queued_input);
    let mut count = usize::from(output.window_action.is_some());
    input.events.clear();
    for _ in 0..8 {
        if !output.activity.queued_input {
            break;
        }
        output = engine.frame(&input).unwrap();
        count += usize::from(output.window_action == Some(GuiWindowAction::ToggleMaximize));
    }
    assert!(!output.activity.queued_input);
    assert_eq!(count, 2);
}
