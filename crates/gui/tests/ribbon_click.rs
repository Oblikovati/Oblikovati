// SPDX-License-Identifier: GPL-2.0-only
use GuiShellAction::{FinishSketch, Home, NewPart, Rectangle, StartSketchXY};
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
    expect_ribbon(&mut engine, &mut request, [50., 90.], Rectangle);
    expect_ribbon(&mut engine, &mut request, [200., 90.], FinishSketch);
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
