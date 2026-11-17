// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::{ParameterDetail, ParameterInfo};

// This test uses a fixed 1024×768, unit-scale frame on both widget platforms.
const PARAMETERS_SHORTCUT: [f32; 2] = [820., 14.];
const NAME_FIELD: [f32; 2] = [295., 200.];
const EQUATION_FIELD: [f32; 2] = [440., 200.];
const ADD_BUTTON: [f32; 2] = [535., 200.];
const ROW_EQUATION: [f32; 2] = [350., 198.];

fn parameter_request() -> GuiFrameRequest {
    GuiFrameRequest {
        extent: [1024, 768],
        events: Vec::new(),
        status: None,
        commands: GuiCommandState {
            theme: None,
            color_scheme: None,
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
    let mut parameter_actions = std::mem::take(&mut output.parameter_actions);
    let mut attempts = 0;
    while output.activity.queued_input {
        assert!(attempts < 8, "expected finite parameter input to drain");
        request.events.clear();
        output = engine.frame(request).unwrap();
        actions.append(&mut output.actions);
        parameter_actions.append(&mut output.parameter_actions);
        attempts += 1;
    }
    output.actions = actions;
    output.parameter_actions = parameter_actions;
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
    click(engine, request, ROW_EQUATION);
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
fn click_parameter(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    position: [f32; 2],
) -> Vec<GuiParameterAction> {
    let mut actions =
        deliver(engine, request, vec![GuiInputEvent::PointerMoved(position)]).parameter_actions;
    actions.extend(
        deliver(
            engine,
            request,
            vec![GuiInputEvent::PointerButton {
                button: GuiPointerButton::Left,
                pressed: true,
            }],
        )
        .parameter_actions,
    );
    actions.extend(
        deliver(
            engine,
            request,
            vec![GuiInputEvent::PointerButton {
                button: GuiPointerButton::Left,
                pressed: false,
            }],
        )
        .parameter_actions,
    );
    actions
}
fn parameter_snapshot(document: u64, rows: Vec<GuiParameterRow>) -> GuiParameterSnapshot {
    GuiParameterSnapshot {
        document,
        rows,
        linked_tables: Vec::new(),
        link_sources: Vec::new(),
        groups: Vec::new(),
    }
}
fn authored_size() -> GuiParameterRow {
    GuiParameterRow {
        parameter: 11,
        tolerance_text: String::new(),
        has_groups: false,
        detail: ParameterDetail {
            parameter_info: ParameterInfo {
                name: "size".into(),
                kind: "user".into(),
                expression: "10 mm".into(),
                value: "10 mm".into(),
                ..Default::default()
            },
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
    engine
        .set_parameter_snapshot(Some(parameter_snapshot(1, Vec::new())))
        .unwrap();
    deliver(&mut engine, &mut request, Vec::new());
    type_at(&mut engine, &mut request, NAME_FIELD, "size");
    type_at(&mut engine, &mut request, EQUATION_FIELD, "10 mm");
    let created = click_parameter(&mut engine, &mut request, ADD_BUTTON);
    let [GuiParameterAction::Add(creation)] = created.as_slice() else {
        panic!("expected one creation, got {created:?}")
    };
    assert_eq!(creation.document, 1);
    assert_eq!(creation.name, "size");
    assert_eq!(
        creation.value,
        GuiParameterInitialValue::Numeric("10 mm".into())
    );
    engine.acknowledge_parameter_action(&created[0], Ok(()));
    engine
        .set_parameter_snapshot(Some(parameter_snapshot(1, vec![authored_size()])))
        .unwrap();
    deliver(&mut engine, &mut request, Vec::new());
    replace_equation(&mut engine, &mut request, mac_osx);
    let edited = click_parameter(&mut engine, &mut request, [700., 350.]);
    let [GuiParameterAction::Text(edit)] = edited.as_slice() else {
        panic!("expected one equation edit, got {edited:?}");
    };
    assert_eq!(edit.target.document, 1);
    assert_eq!(edit.target.parameter, 11);
    assert_eq!(edit.target.field, GuiParameterTextField::Equation);
    assert_eq!(edit.value, "20 mm");
    request.commands.document = Some(2);
    deliver(&mut engine, &mut request, Vec::new());
    open_parameters(&mut engine, &mut request);
    assert!(click(&mut engine, &mut request, ADD_BUTTON).is_empty());
}

fn open_parameters(engine: &mut GuiFrameEngine, request: &mut GuiFrameRequest) {
    let actions = click(engine, request, PARAMETERS_SHORTCUT);
    assert!(actions.is_empty(), "unexpected actions: {actions:?}");
    assert!(
        engine.parameters_open(),
        "Parameters shortcut must open the editor"
    );
}
