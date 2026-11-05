// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::*;
use oblikovati_model::theme::ThemeLibrary;
fn editor_request(library: &ThemeLibrary) -> GuiFrameRequest {
    GuiFrameRequest {
        extent: [1440, 900],
        events: Vec::new(),
        status: None,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
        commands: GuiCommandState {
            document: Some(1),
            document_type: Some(oblikovati_api::types::DOCUMENT_DRAWING),
            authoring: GuiAuthoringContext::OtherDocument,
            theme: Some(GuiThemeState::from_library(library).with_custom_save_available(true)),
            color_scheme: Some(GuiColorSchemeState::from_registry(&Default::default()).unwrap()),
            ..Default::default()
        },
    }
}
fn deliver(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    events: Vec<GuiInputEvent>,
) -> GuiFrameOutput {
    request.events = events;
    engine.frame(request).unwrap()
}
fn click(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    position: [f32; 2],
) -> Vec<GuiShellAction> {
    deliver(engine, request, vec![GuiInputEvent::PointerMoved(position)]);
    let mut actions = Vec::new();
    for pressed in [true, false] {
        actions.extend(
            deliver(
                engine,
                request,
                vec![GuiInputEvent::PointerButton {
                    button: GuiPointerButton::Left,
                    pressed,
                }],
            )
            .actions,
        );
    }
    actions
}
fn settle(engine: &mut GuiFrameEngine, request: &mut GuiFrameRequest) -> GuiFrameOutput {
    for _ in 0..40 {
        let output = deliver(engine, request, Vec::new());
        if !output.activity.needs_frame() {
            return output;
        }
    }
    panic!("theme editor: expected idle GUI after forty unchanged frames");
}
fn modal_dim_alpha(output: &GuiFrameOutput) -> u8 {
    output
        .draws
        .batches
        .iter()
        .flat_map(|batch| &batch.vertices)
        .filter(|vertex| vertex.position == [0., 0.] && vertex.srgb[..3] == [204; 3])
        .map(|vertex| vertex.srgb[3])
        .max()
        .unwrap_or(0)
}
fn control_click(engine: &mut GuiFrameEngine, request: &mut GuiFrameRequest, y: f32) {
    deliver(
        engine,
        request,
        vec![GuiInputEvent::Modifiers(GuiModifiers {
            control: true,
            ..Default::default()
        })],
    );
    assert!(click(engine, request, [480., y]).is_empty());
    deliver(
        engine,
        request,
        vec![GuiInputEvent::Modifiers(Default::default())],
    );
}
#[test]
fn editor_previews_custom_colours_emits_save_and_keeps_builtin_controls_read_only() {
    let mut library = ThemeLibrary::new(Vec::new(), "Dark").unwrap();
    library.duplicate("Dark", "Workshop").unwrap();
    let mut request = editor_request(&library);
    let mut engine = GuiFrameEngine::new().unwrap();
    engine.frame(&request).unwrap();
    engine.frame(&request).unwrap();
    assert!(click(&mut engine, &mut request, [440., 91.]).is_empty());
    let opened = settle(&mut engine, &mut request);
    assert_eq!(
        modal_dim_alpha(&opened),
        89,
        "expected the modal fade to complete before idling"
    );
    verify_custom_edit(&mut engine, &mut request, &mut library);
    set_document_presence(&mut request, false);
    deliver(&mut engine, &mut request, Vec::new());
    assert_eq!(
        click(&mut engine, &mut request, [490., 196.]),
        [GuiShellAction::SaveTheme {
            name: "Workshop".into()
        }]
    );
    assert!(click(&mut engine, &mut request, [590., 196.]).is_empty());
    assert_eq!(modal_dim_alpha(&settle(&mut engine, &mut request)), 0);
    set_document_presence(&mut request, true);
    verify_builtin_read_only(&mut engine, &mut request, &mut library);
    verify_editor_retires_with_its_theme_owner(&mut engine, &mut request);
}
fn verify_custom_edit(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    library: &mut ThemeLibrary,
) {
    control_click(engine, request, 259.);
    let output = deliver(engine, request, vec![GuiInputEvent::Text("80".into())]);
    let [GuiShellAction::EditThemeColor { name, token, value }] = output.actions.as_slice() else {
        panic!("expected one custom colour edit, got {:?}", output.actions);
    };
    assert_eq!(name, "Workshop");
    assert_eq!(token, &oblikovati_api::types::TOKEN_CHROME_WINDOW_BG);
    assert_eq!(value.r, 80. / 255.);
    assert_ne!(library.active().color(token).r, value.r);
    library.edit_active_color(token, value.clone()).unwrap();
    request.commands.theme =
        Some(GuiThemeState::from_library(library).with_custom_save_available(true));
    deliver(
        engine,
        request,
        vec![GuiInputEvent::Key {
            key: GuiKey::Enter,
            pressed: true,
        }],
    );
    deliver(
        engine,
        request,
        vec![GuiInputEvent::Key {
            key: GuiKey::Enter,
            pressed: false,
        }],
    );
}
fn verify_builtin_read_only(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    library: &mut ThemeLibrary,
) {
    library.set_active("Dark");
    request.commands.theme =
        Some(GuiThemeState::from_library(library).with_custom_save_available(true));
    deliver(engine, request, Vec::new());
    assert!(click(engine, request, [440., 91.]).is_empty());
    settle(engine, request);
    control_click(engine, request, 276.);
    assert!(
        deliver(engine, request, vec![GuiInputEvent::Text("80".into())])
            .actions
            .is_empty()
    );
    assert!(click(engine, request, [490., 196.]).is_empty());
    assert!(click(engine, request, [590., 196.]).is_empty());
    settle(engine, request);
}

fn set_document_presence(request: &mut GuiFrameRequest, present: bool) {
    request.commands.document = present.then_some(1);
    request.commands.document_type = present.then_some(oblikovati_api::types::DOCUMENT_DRAWING);
    request.commands.authoring = if present {
        GuiAuthoringContext::OtherDocument
    } else {
        GuiAuthoringContext::NoDocument
    };
}

fn verify_editor_retires_with_its_theme_owner(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
) {
    assert!(click(engine, request, [440., 91.]).is_empty());
    request.commands.theme = None;
    deliver(engine, request, Vec::new());
    assert_eq!(modal_dim_alpha(&settle(engine, request)), 0);
}
