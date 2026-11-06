// SPDX-License-Identifier: GPL-2.0-only
use GuiShellAction::{FinishSketch, Home, NewPart, StartSketchXY};
use oblikovati_api::types::DOCUMENT_PART;
use oblikovati_gui::*;

fn ribbon_request() -> GuiFrameRequest {
    GuiFrameRequest {
        extent: [1000, 700],
        events: Vec::new(),
        commands: Default::default(),
        status: None,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
    }
}
fn settle_ribbon(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
) -> Vec<GuiShellAction> {
    request.events.clear();
    let mut actions = Vec::new();
    for _ in 0..32 {
        let output = engine.frame(request).unwrap();
        actions.extend(output.actions);
        let activity = output.activity;
        if !activity.queued_input && !activity.pending_layout && !activity.pending_navigation {
            return actions;
        }
    }
    panic!("ribbon input exceeded 32 frames: expected finite input and layout to settle");
}
fn click_ribbon(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    point: [f32; 2],
) -> Vec<GuiShellAction> {
    request.events = vec![GuiInputEvent::PointerMoved(point)];
    let mut actions = engine.frame(request).unwrap().actions;
    for pressed in [true, false] {
        request.events = vec![GuiInputEvent::PointerButton {
            button: GuiPointerButton::Left,
            pressed,
        }];
        actions.extend(engine.frame(request).unwrap().actions);
    }
    actions.extend(settle_ribbon(engine, request));
    actions
}
fn enter_part(request: &mut GuiFrameRequest) {
    request.commands.document = Some(7);
    request.commands.document_type = Some(DOCUMENT_PART);
    request.commands.authoring = GuiAuthoringContext::Part;
}
fn expect_ribbon(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    point: [f32; 2],
    expected: GuiShellAction,
) {
    let actions = click_ribbon(engine, request, point);
    assert_eq!(actions, [expected]);
}
#[test]
fn context_changes_select_the_supported_ribbon_commands() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    settle_ribbon(&mut engine, &mut request);
    expect_ribbon(&mut engine, &mut request, [50., 90.], NewPart);
    enter_part(&mut request);
    settle_ribbon(&mut engine, &mut request);
    expect_ribbon(&mut engine, &mut request, [50., 90.], StartSketchXY);
    request.commands.authoring = GuiAuthoringContext::Sketch;
    settle_ribbon(&mut engine, &mut request);
    expect_ribbon(
        &mut engine,
        &mut request,
        [50., 90.],
        GuiShellAction::PlaceSketchTool(GuiSketchTool::Rectangle),
    );
    expect_ribbon(&mut engine, &mut request, [440., 90.], FinishSketch);
}
#[test]
fn manual_view_selection_survives_frames_until_sketch_exit() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    enter_part(&mut request);
    request.commands.authoring = GuiAuthoringContext::Sketch;
    settle_ribbon(&mut engine, &mut request);
    assert!(click_ribbon(&mut engine, &mut request, [450., 44.]).is_empty());
    expect_ribbon(&mut engine, &mut request, [50., 90.], Home);
    expect_ribbon(&mut engine, &mut request, [50., 90.], Home);
    request.commands.authoring = GuiAuthoringContext::Part;
    settle_ribbon(&mut engine, &mut request);
    expect_ribbon(&mut engine, &mut request, [50., 90.], StartSketchXY);
    request.commands = Default::default();
    settle_ribbon(&mut engine, &mut request);
    expect_ribbon(&mut engine, &mut request, [50., 90.], NewPart);
}
#[test]
fn the_ribbon_captures_the_full_logical_window_width_at_high_dpi() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    request.extent = [2000, 1400];
    request.framebuffer_scale = [2.; 2];
    settle_ribbon(&mut engine, &mut request);
    request.events = vec![GuiInputEvent::PointerMoved([1800., 80.])];
    assert!(engine.frame(&request).unwrap().capture.pointer);
    request.events = vec![GuiInputEvent::PointerMoved([1800., 1000.])];
    assert!(!engine.frame(&request).unwrap().capture.pointer);
}

#[test]
fn pointer_wheel_does_not_scroll_the_fixed_ribbon_band() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    settle_ribbon(&mut engine, &mut request);
    request.events = vec![GuiInputEvent::PointerMoved([900., 80.])];
    engine.frame(&request).unwrap();
    settle_ribbon(&mut engine, &mut request);
    let before = engine.frame(&request).unwrap().draws;
    request.events = vec![GuiInputEvent::Wheel([0., -10.])];
    engine.frame(&request).unwrap();
    settle_ribbon(&mut engine, &mut request);
    let after = engine.frame(&request).unwrap().draws;
    assert!(
        before == after,
        "ribbon wheel [0, -10]: expected unchanged fixed-band draws"
    );
}
#[test]
fn every_supported_tool_dispatches_from_the_sketch_ribbon_at_both_scales() {
    use GuiSketchTool::*;
    for scale in [1., 2.] {
        let mut engine = GuiFrameEngine::new().unwrap();
        let mut request = ribbon_request();
        request.extent = [640 * scale as u32, 360 * scale as u32];
        request.framebuffer_scale = [scale; 2];
        enter_part(&mut request);
        request.commands.authoring = GuiAuthoringContext::Sketch;
        settle_ribbon(&mut engine, &mut request);
        for (tool, point) in [
            (Line, [180., 128.]),
            (Rectangle, [50., 90.]),
            (Circle, [180., 68.]),
            (Point, [300., 68.]),
            (CenterPointArc, [180., 88.]),
            (Ellipse, [300., 88.]),
            (CenterRectangle, [180., 108.]),
            (ThreePointRectangle, [300., 108.]),
        ] {
            expect_ribbon(
                &mut engine,
                &mut request,
                point.map(|value| value * scale),
                GuiShellAction::PlaceSketchTool(tool),
            );
        }
        expect_ribbon(
            &mut engine,
            &mut request,
            [440. * scale, 90. * scale],
            FinishSketch,
        );
    }
}
#[test]
fn sketch_tool_captions_identify_the_existing_construction_choices() {
    use GuiSketchTool::*;
    for (tool, label) in [
        (Line, "Line"),
        (Rectangle, "Rectangle"),
        (Circle, "Circle"),
        (Point, "Point"),
        (CenterPointArc, "Center Arc"),
        (Ellipse, "Ellipse"),
        (CenterRectangle, "Center Rect."),
        (ThreePointRectangle, "3-Point Rect."),
        (ThreePointCircle, "3-Point Circle"),
        (ThreePointArc, "3-Point Arc"),
    ] {
        assert_eq!(tool.label(), label);
    }
}

#[test]
fn geometric_tools_dispatch_at_both_scales_and_refuse_open_transactions() {
    use GuiSketchConstraint::{
        Coincident, Collinear, Concentric, Equal, Fix, Horizontal, Parallel, Perpendicular,
        Tangent, Vertical,
    };
    for scale in [1., 2.] {
        let mut engine = GuiFrameEngine::new().unwrap();
        let mut request = ribbon_request();
        request.extent = [1000 * scale as u32, 360 * scale as u32];
        request.framebuffer_scale = [scale; 2];
        enter_part(&mut request);
        request.commands.authoring = GuiAuthoringContext::Sketch;
        settle_ribbon(&mut engine, &mut request);
        for (choice, y) in [(Horizontal, 68.), (Vertical, 88.), (Coincident, 108.)] {
            expect_ribbon(
                &mut engine,
                &mut request,
                [550. * scale, y * scale],
                GuiShellAction::ConstrainSketch(choice),
            );
        }
        expect_ribbon(
            &mut engine,
            &mut request,
            [670. * scale, 128. * scale],
            GuiShellAction::ConstrainSketch(Fix),
        );
        for (choice, y) in [(Parallel, 68.), (Perpendicular, 88.), (Collinear, 108.)] {
            expect_ribbon(
                &mut engine,
                &mut request,
                [790. * scale, y * scale],
                GuiShellAction::ConstrainSketch(choice),
            );
        }
        for (choice, y) in [(Equal, 68.), (Concentric, 88.), (Tangent, 108.)] {
            expect_ribbon(
                &mut engine,
                &mut request,
                [900. * scale, y * scale],
                GuiShellAction::ConstrainSketch(choice),
            );
        }
        request.commands.transaction_open = true;
        settle_ribbon(&mut engine, &mut request);
        assert!(click_ribbon(&mut engine, &mut request, [550. * scale, 68. * scale]).is_empty());
        assert!(click_ribbon(&mut engine, &mut request, [670. * scale, 128. * scale]).is_empty());
    }
}

#[test]
fn general_dimension_dispatches_at_both_scales_with_transaction_guards() {
    for scale in [1., 2.] {
        let mut engine = GuiFrameEngine::new().unwrap();
        let mut request = ribbon_request();
        request.extent = [1200 * scale as u32, 800 * scale as u32];
        request.framebuffer_scale = [scale; 2];
        enter_part(&mut request);
        request.commands.authoring = GuiAuthoringContext::Sketch;
        settle_ribbon(&mut engine, &mut request);
        expect_ribbon(
            &mut engine,
            &mut request,
            [670. * scale, 85. * scale],
            GuiShellAction::Dimension(GuiDimensionAction::Start),
        );
        request.commands.transaction_open = true;
        settle_ribbon(&mut engine, &mut request);
        assert!(click_ribbon(&mut engine, &mut request, [670. * scale, 85. * scale]).is_empty());
    }
}
fn dimension_editor_key(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    key: GuiKey,
) -> Vec<GuiShellAction> {
    let mut actions = Vec::new();
    for pressed in [true, false] {
        request.events = vec![GuiInputEvent::Key { key, pressed }];
        actions.extend(engine.frame(request).unwrap().actions);
        actions.extend(settle_ribbon(engine, request));
    }
    actions
}
#[test]
fn dimension_editor_enter_uses_the_current_seed_and_escape_cancels() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    request.extent = [1200, 800];
    enter_part(&mut request);
    request.commands.authoring = GuiAuthoringContext::Sketch;
    for (id, expression) in [(42, "10 mm"), (43, "20 mm")] {
        request.commands.dimension_draft = Some(GuiDimensionDraft {
            id,
            kind: "distance".into(),
            orientation: "horizontal".into(),
            expression: expression.into(),
        });
        settle_ribbon(&mut engine, &mut request);
        assert_eq!(
            dimension_editor_key(&mut engine, &mut request, GuiKey::Enter),
            [GuiShellAction::Dimension(GuiDimensionAction::Apply {
                id,
                expression: expression.into(),
                driven: false
            })]
        );
    }
    request.commands.transaction_open = true;
    assert!(dimension_editor_key(&mut engine, &mut request, GuiKey::Enter).is_empty());
    assert_eq!(
        dimension_editor_key(&mut engine, &mut request, GuiKey::Escape),
        [GuiShellAction::Dimension(GuiDimensionAction::Cancel {
            id: 43
        })]
    );
}

fn annotation_theme() -> GuiThemeState {
    use oblikovati_api::types::*;
    let mut library = oblikovati_model::theme::ThemeLibrary::new(Vec::new(), "Dark").unwrap();
    library.duplicate("Dark", "Dimension test").unwrap();
    library.set_active("Dimension test");
    for (token, [r, g, b]) in [
        (TOKEN_DIMENSION_DRIVING, [1., 0., 0.]),
        (TOKEN_DIMENSION_DRIVEN, [0., 0., 1.]),
        (TOKEN_DIMENSION_SKETCH, [0., 1., 0.]),
    ] {
        library
            .edit_active_color(&token, Rgba { r, g, b, a: 1. })
            .unwrap();
    }
    GuiThemeState::from_library(&library)
}
#[test]
fn dimension_labels_follow_theme_colours_and_physical_projection_at_both_scales() {
    use GuiDimensionLabelRole::{Driven, Driving, Preview};
    for scale in [1., 2.] {
        let mut engine = GuiFrameEngine::new().unwrap();
        let mut request = ribbon_request();
        request.extent = [1200 * scale as u32, 800 * scale as u32];
        request.framebuffer_scale = [scale; 2];
        request.commands.theme = Some(annotation_theme());
        for (role, y) in [(Driving, 400), (Driven, 500), (Preview, 600)] {
            request.commands.dimension_labels.push(GuiDimensionLabel {
                pixel: [800 * scale as i32, y * scale as i32],
                text: "d0 = 10 mm".into(),
                role,
            });
        }
        settle_ribbon(&mut engine, &mut request);
        let frame = engine.frame(&request).unwrap().draws;
        for (y, colour) in [
            (400, [255, 0, 0, 255]),
            (500, [0, 0, 255, 255]),
            (600, [0, 255, 0, 255]),
        ] {
            assert!(
                frame
                    .batches
                    .iter()
                    .flat_map(|batch| &batch.vertices)
                    .any(|vertex| vertex.position[0] > 700. * scale
                        && vertex.position[0] < 900. * scale
                        && (vertex.position[1] - y as f32 * scale).abs() < 20. * scale
                        && vertex.srgb == colour),
                "{scale}: missing annotation colour {colour:?} at {y}"
            );
        }
        request.commands.dimension_labels[0].text = "bad\0label".into();
        assert!(engine.frame(&request).unwrap_err().contains("without NUL"));
    }
}

#[test]
fn malformed_dimension_drafts_are_refused_before_gui_input() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    request.extent = [1200, 800];
    enter_part(&mut request);
    request.commands.authoring = GuiAuthoringContext::Sketch;
    let valid = GuiDimensionDraft {
        id: 1,
        kind: "distance".into(),
        orientation: "horizontal".into(),
        expression: "10 mm".into(),
    };
    for invalid in [
        GuiDimensionDraft {
            id: 0,
            ..valid.clone()
        },
        GuiDimensionDraft {
            kind: "unknown".into(),
            ..valid.clone()
        },
        GuiDimensionDraft {
            orientation: "skew".into(),
            ..valid.clone()
        },
        GuiDimensionDraft {
            expression: "bad\0source".into(),
            ..valid.clone()
        },
    ] {
        request.commands.dimension_draft = Some(invalid);
        assert!(
            engine
                .frame(&request)
                .unwrap_err()
                .contains("expected a positive owner draft")
        );
    }
    request.commands.dimension_draft = Some(valid);
    engine.frame(&request).unwrap();
    request.commands.document = None;
    assert!(engine.frame(&request).is_err());
}

#[test]
fn line_commands_pass_ribbon_keyboard_focus_without_reactivating_buttons() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    enter_part(&mut request);
    request.commands.authoring = GuiAuthoringContext::Sketch;
    request.commands.undo_label = Some("Create sketch".into());
    settle_ribbon(&mut engine, &mut request);
    expect_ribbon(
        &mut engine,
        &mut request,
        [180., 128.],
        GuiShellAction::PlaceSketchTool(GuiSketchTool::Line),
    );
    request.commands.transaction_open = true;
    request.commands.sketch_tool_transaction = true;
    settle_ribbon(&mut engine, &mut request);
    assert!(click_ribbon(&mut engine, &mut request, [75., 45.]).is_empty());
    for key in [GuiKey::Backspace, GuiKey::Enter] {
        for pressed in [true, false] {
            request.events = vec![GuiInputEvent::Key { key, pressed }];
            let output = engine.frame(&request).unwrap();
            assert!(!output.capture.keyboard && output.actions.is_empty());
        }
    }
}

#[path = "ribbon_click/sketch_delete.rs"]
mod sketch_delete;

#[path = "ribbon_click/circular_flyouts.rs"]
mod circular_flyouts;
