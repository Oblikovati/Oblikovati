// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::*;
#[allow(dead_code, unsafe_code)]
#[path = "../../../test-utilities/rust/drawn_text.rs"]
mod drawn_text;
#[path = "support/view_tab.rs"]
mod view_tab;
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
            document_type: Some(oblikovati_api::types::DOCUMENT_UNKNOWN),
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
    deliver(
        engine,
        request,
        vec![GuiInputEvent::PointerMoved([1300., 800.])],
    );
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
fn find_label(label: &str) -> drawn_text::RenderedRun {
    let frame = drawn_text::rendered_frame();
    frame
        .runs
        .into_iter()
        .find(|run| run.text == label)
        .unwrap_or_else(|| panic!("expected rendered label {label:?}"))
}
fn click_label(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    label: &str,
) -> Vec<GuiShellAction> {
    let center = find_label(label).center();
    click(engine, request, center)
}
fn control_click_red_channel(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    row_label: &str,
) {
    let row = find_label(row_label);
    let y = row.center()[1];
    let red = drawn_text::rendered_frame()
        .runs
        .into_iter()
        .find(|run| run.text.starts_with("R:") && (run.center()[1] - y).abs() < 8.)
        .unwrap_or_else(|| panic!("expected R: input beside {row_label:?}"));
    deliver(
        engine,
        request,
        vec![GuiInputEvent::Modifiers(GuiModifiers {
            control: true,
            ..Default::default()
        })],
    );
    assert!(click(engine, request, red.center()).is_empty());
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
    view_tab::select_view_tab(&mut engine, &mut request);
    assert!(click_label(&mut engine, &mut request, "EditTheme").is_empty());
    let opened = settle(&mut engine, &mut request);
    assert_eq!(
        modal_dim_alpha(&opened),
        0,
        "expected the dockable editor to leave the workspace undimmed"
    );
    verify_custom_edit(&mut engine, &mut request, &mut library);
    set_document_presence(&mut request, false);
    deliver(&mut engine, &mut request, Vec::new());
    assert_eq!(
        click_label(&mut engine, &mut request, "Savetheme"),
        [GuiShellAction::SaveTheme {
            name: "Workshop".into()
        }]
    );
    assert!(click_label(&mut engine, &mut request, "Close").is_empty());
    assert_eq!(modal_dim_alpha(&settle(&mut engine, &mut request)), 0);
    set_document_presence(&mut request, true);
    settle(&mut engine, &mut request);
    view_tab::select_view_tab(&mut engine, &mut request);
    verify_builtin_read_only(&mut engine, &mut request, &mut library);
    verify_lifecycle_controls(&mut engine, &mut request, &mut library);
    verify_editor_retires_with_its_theme_owner(&mut engine, &mut request);
}
fn verify_custom_edit(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    library: &mut ThemeLibrary,
) {
    control_click_red_channel(engine, request, "Windowbackground");
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
    assert!(click_label(engine, request, "EditTheme").is_empty());
    settle(engine, request);
    control_click_red_channel(engine, request, "Windowbackground");
    assert!(
        deliver(engine, request, vec![GuiInputEvent::Text("80".into())])
            .actions
            .is_empty()
    );
    assert!(click_label(engine, request, "Savetheme").is_empty());
    assert!(click_label(engine, request, "Close").is_empty());
    settle(engine, request);
}

fn set_document_presence(request: &mut GuiFrameRequest, present: bool) {
    request.commands.document = present.then_some(1);
    request.commands.document_type = present.then_some(oblikovati_api::types::DOCUMENT_UNKNOWN);
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
    assert!(click_label(engine, request, "EditTheme").is_empty());
    request.commands.theme = None;
    // Without a theme the panels shift, and a button can move under the launcher's old position:
    // a hovered button is rightly not idle, so the pointer leaves first.
    deliver(
        engine,
        request,
        vec![GuiInputEvent::PointerMoved([1000., 600.])],
    );
    assert_eq!(modal_dim_alpha(&settle(engine, request)), 0);
}

fn verify_lifecycle_controls(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    library: &mut ThemeLibrary,
) {
    assert!(click_label(engine, request, "EditTheme").is_empty());
    settle(engine, request);
    assert!(
        click_label(engine, request, "Delete").is_empty(),
        "built-in Delete must be disabled"
    );
    let new_name = find_label("Newname");
    assert!(
        click(
            engine,
            request,
            [new_name.maximum[0] + 40., new_name.center()[1]]
        )
        .is_empty()
    );
    deliver(
        engine,
        request,
        vec![GuiInputEvent::Text("My Theme".into())],
    );
    assert_eq!(
        click_label(engine, request, "Duplicate"),
        [GuiShellAction::DuplicateTheme {
            base: "Dark".into(),
            name: "My Theme".into()
        }]
    );
    library.duplicate("Dark", "My Theme").unwrap();
    request.commands.theme = Some(GuiThemeState::from_library(library));
    deliver(engine, request, Vec::new());
    assert_eq!(
        click_label(engine, request, "Delete"),
        [GuiShellAction::DeleteTheme {
            name: "My Theme".into()
        }]
    );
    library.remove("My Theme").unwrap();
    request.commands.theme = Some(GuiThemeState::from_library(library));
    deliver(engine, request, Vec::new());
    assert!(click_label(engine, request, "Close").is_empty());
    settle(engine, request);
}
