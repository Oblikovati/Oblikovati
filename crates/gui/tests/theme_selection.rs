// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::*;
#[allow(dead_code, unsafe_code)]
#[path = "../../../test-utilities/rust/drawn_text.rs"]
mod drawn_text;
#[path = "support/view_tab.rs"]
mod view_tab;
use oblikovati_model::theme::ThemeLibrary;
fn theme_request(library: &ThemeLibrary) -> GuiFrameRequest {
    GuiFrameRequest {
        extent: [900, 600],
        events: Vec::new(),
        status: None,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
        commands: GuiCommandState {
            theme: Some(GuiThemeState::from_library(library)),
            document: Some(1),
            document_type: Some(oblikovati_api::types::DOCUMENT_UNKNOWN),
            authoring: GuiAuthoringContext::OtherDocument,
            ..Default::default()
        },
    }
}
fn click_position(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    position: [f32; 2],
) -> Vec<GuiShellAction> {
    request.events = vec![GuiInputEvent::PointerMoved(position)];
    engine.frame(request).unwrap();
    let mut actions = Vec::new();
    for pressed in [true, false] {
        request.events = vec![GuiInputEvent::PointerButton {
            button: GuiPointerButton::Left,
            pressed,
        }];
        actions.extend(engine.frame(request).unwrap().actions);
    }
    request.events.clear();
    actions.extend(engine.frame(request).unwrap().actions);
    actions
}
fn click_label(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    label: &str,
) -> Vec<GuiShellAction> {
    let run = drawn_text::rendered_frame()
        .runs
        .into_iter()
        .rev()
        .find(|run| run.text == label)
        .unwrap_or_else(|| panic!("expected rendered label {label:?}"));
    click_position(engine, request, run.center())
}
#[test]
fn view_theme_picker_emits_owner_action_and_next_snapshot_restyles_draws() {
    let mut library = ThemeLibrary::new(Vec::new(), "Dark").unwrap();
    let mut request = theme_request(&library);
    let mut engine = GuiFrameEngine::new().unwrap();
    engine.frame(&request).unwrap();
    engine.frame(&request).unwrap();
    view_tab::select_view_tab(&mut engine, &mut request);
    request.events.clear();
    let dark = engine.frame(&request).unwrap().draws;
    assert!(click_label(&mut engine, &mut request, "Dark").is_empty());
    let actions = click_label(&mut engine, &mut request, "Light");
    assert_eq!(actions, [GuiShellAction::SetTheme("Light".into())]);
    assert_eq!(library.active().name(), "Dark");
    library.set_active("Light");
    request = theme_request(&library);
    let light = engine.frame(&request).unwrap().draws;
    assert_ne!(dark, light);
    verify_colour_picker(&mut engine, &mut request);
    let text = library
        .active()
        .color(&oblikovati_api::types::TOKEN_CHROME_TEXT);
    let text_bytes = [text.r, text.g, text.b, text.a].map(|channel| (channel * 255. + 0.5) as u8);
    assert!(
        light
            .batches
            .iter()
            .flat_map(|batch| &batch.vertices)
            .any(|vertex| vertex.srgb == text_bytes)
    );
}

fn verify_colour_picker(engine: &mut GuiFrameEngine, request: &mut GuiFrameRequest) {
    let mut registry = oblikovati_model::color_scheme::ColorSchemeRegistry::default();
    request.commands.color_scheme = Some(GuiColorSchemeState::from_registry(&registry).unwrap());
    engine.frame(request).unwrap();
    assert!(click_label(engine, request, "Default").is_empty());
    let actions = click_label(engine, request, "HighContrast");
    let selection = oblikovati_api::wire::SetColorSchemeArgs {
        name: "High Contrast".into(),
    };
    assert_eq!(actions, [GuiShellAction::SetColorScheme(selection.clone())]);
    assert!(actions[0].enabled(&request.commands));
    assert_eq!(registry.active().name, "Default");
    registry.set_active(&selection.name).unwrap();
    request.commands.color_scheme = Some(GuiColorSchemeState::from_registry(&registry).unwrap());
    engine.frame(request).unwrap();
    assert_eq!(
        request.commands.color_scheme.as_ref().unwrap().background(),
        oblikovati_viewport::ViewportBackground::Solid([0; 3])
    );
    verify_palette_picker_without_a_theme_library(engine, request);
}

fn verify_palette_picker_without_a_theme_library(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
) {
    request.commands.theme = None;
    engine.frame(request).unwrap();
    assert!(click_label(engine, request, "HighContrast").is_empty());
    let actions = click_label(engine, request, "Sky");
    assert_eq!(
        actions,
        [GuiShellAction::SetColorScheme(
            oblikovati_api::wire::SetColorSchemeArgs { name: "Sky".into() }
        )]
    );
}
