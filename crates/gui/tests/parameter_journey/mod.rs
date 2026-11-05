// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::ParameterSetArgs;

fn parameter_request() -> GuiFrameRequest {
    GuiFrameRequest {
        extent: [1024, 768],
        events: Vec::new(),
        status: None,
        commands: GuiCommandState {
            document: Some(1),
            authoring: GuiAuthoringContext::Part,
            document_type: Some(oblikovati_api::types::DOCUMENT_PART),
            ..Default::default()
        },
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
    }
}
fn deliver(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    events: Vec<GuiInputEvent>,
) -> GuiFrameOutput {
    request.events = events;
    let mut output = engine.frame(request).unwrap();
    let mut actions = std::mem::take(&mut output.actions);
    let mut attempts = 0;
    while output.activity.queued_input {
        assert!(attempts < 8, "expected finite parameter input to drain");
        request.events.clear();
        output = engine.frame(request).unwrap();
        actions.append(&mut output.actions);
        attempts += 1;
    }
    output.actions = actions;
    output
}
fn click(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    position: [f32; 2],
) -> Vec<GuiShellAction> {
    deliver(engine, request, vec![GuiInputEvent::PointerMoved(position)]);
    deliver(
        engine,
        request,
        vec![GuiInputEvent::PointerButton {
            button: GuiPointerButton::Left,
            pressed: true,
        }],
    );
    deliver(
        engine,
        request,
        vec![GuiInputEvent::PointerButton {
            button: GuiPointerButton::Left,
            pressed: false,
        }],
    )
    .actions
}
fn type_at(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    position: [f32; 2],
    text: &str,
) {
    assert!(click(engine, request, position).is_empty());
    let output = deliver(engine, request, vec![GuiInputEvent::Text(text.into())]);
    assert!(output.capture.text && output.actions.is_empty());
}
fn replace_equation(engine: &mut GuiFrameEngine, request: &mut GuiFrameRequest, mac_osx: bool) {
    click(engine, request, [400., 66.]);
    deliver(
        engine,
        request,
        vec![
            // #3795: select-all follows the configured widget platform, including Command on macOS.
            GuiInputEvent::Modifiers(GuiModifiers {
                control: !mac_osx,
                super_key: mac_osx,
                ..Default::default()
            }),
            GuiInputEvent::Key {
                key: GuiKey::A,
                pressed: true,
            },
        ],
    );
    deliver(
        engine,
        request,
        vec![
            GuiInputEvent::Key {
                key: GuiKey::A,
                pressed: false,
            },
            GuiInputEvent::Modifiers(Default::default()),
        ],
    );
    deliver(engine, request, vec![GuiInputEvent::Text("20 mm".into())]);
}
fn expected_parameter(mutation: GuiParameterMutation, expression: &str) -> GuiShellAction {
    GuiShellAction::Parameter {
        mutation,
        request: ParameterSetArgs {
            name: "size".into(),
            expression: expression.into(),
            ..Default::default()
        },
    }
}
/// Drive the real parameter widgets; example: `exercise_parameter_journey(engine, true)`.
pub(crate) fn exercise_parameter_journey(mut engine: GuiFrameEngine, mac_osx: bool) {
    let mut request = parameter_request();
    deliver(&mut engine, &mut request, Vec::new());
    deliver(&mut engine, &mut request, Vec::new());
    open_parameters(&mut engine, &mut request);
    deliver(&mut engine, &mut request, Vec::new());
    type_at(&mut engine, &mut request, [400., 43.], "size");
    type_at(&mut engine, &mut request, [400., 66.], "10 mm");
    assert_eq!(
        click(&mut engine, &mut request, [345., 90.]),
        [expected_parameter(GuiParameterMutation::Add, "10 mm")]
    );
    replace_equation(&mut engine, &mut request, mac_osx);
    assert_eq!(
        click(&mut engine, &mut request, [380., 90.]),
        [expected_parameter(GuiParameterMutation::Set, "20 mm")]
    );
    request.commands.document = Some(2);
    deliver(&mut engine, &mut request, Vec::new());
    open_parameters(&mut engine, &mut request);
    assert!(click(&mut engine, &mut request, [345., 90.]).is_empty());
}

fn open_parameters(engine: &mut GuiFrameEngine, request: &mut GuiFrameRequest) {
    assert!(click(engine, request, [350., 44.]).is_empty());
    request.events.clear();
    engine.frame(request).unwrap();
    assert!(click(engine, request, [50., 90.]).is_empty());
}
