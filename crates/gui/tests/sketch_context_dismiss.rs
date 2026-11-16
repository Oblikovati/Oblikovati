// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::*;

fn popup() -> (GuiFrameEngine, GuiFrameRequest) {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = GuiFrameRequest {
        extent: [1000, 700],
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
        commands: GuiCommandState {
            authoring: GuiAuthoringContext::Sketch,
            sketch_at_ready: true,
            sketch_context: Some(GuiSketchContext {
                token: 42,
                ..Default::default()
            }),
            ..Default::default()
        },
        events: Vec::new(),
        status: None,
    };
    for _ in 0..5 {
        engine.frame(&request).unwrap();
    }
    request.events = vec![GuiInputEvent::PointerMoved([600., 400.])];
    engine.frame(&request).unwrap();
    for pressed in [true, false] {
        request.events = vec![GuiInputEvent::PointerButton {
            button: GuiPointerButton::Right,
            pressed,
        }];
        engine.frame(&request).unwrap();
    }
    (engine, request)
}
fn frame(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    event: GuiInputEvent,
) -> Vec<GuiShellAction> {
    request.events = vec![event];
    engine.frame(request).unwrap().actions
}
fn escape_retires_the_popup_token() {
    let (mut engine, mut request) = popup();
    let mut actions = frame(
        &mut engine,
        &mut request,
        GuiInputEvent::Key {
            key: GuiKey::Escape,
            pressed: true,
        },
    );
    actions.extend(frame(
        &mut engine,
        &mut request,
        GuiInputEvent::Key {
            key: GuiKey::Escape,
            pressed: false,
        },
    ));
    request.events.clear();
    actions.extend(engine.frame(&request).unwrap().actions);
    assert!(
        actions.contains(&GuiShellAction::DismissSketchContext { token: 42 }),
        "{actions:?}"
    );
}
fn outside_click_retires_the_popup_token() {
    let (mut engine, mut request) = popup();
    frame(
        &mut engine,
        &mut request,
        GuiInputEvent::PointerMoved([900., 600.]),
    );
    let mut actions = Vec::new();
    for pressed in [true, false] {
        actions.extend(frame(
            &mut engine,
            &mut request,
            GuiInputEvent::PointerButton {
                button: GuiPointerButton::Left,
                pressed,
            },
        ));
    }
    request.events.clear();
    actions.extend(engine.frame(&request).unwrap().actions);
    assert!(
        actions.contains(&GuiShellAction::DismissSketchContext { token: 42 }),
        "{actions:?}"
    );
}

#[test]
fn dismissing_canvas_popup_retires_its_token() {
    escape_retires_the_popup_token();
    outside_click_retires_the_popup_token();
}
