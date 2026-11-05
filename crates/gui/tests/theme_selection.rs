// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::*;
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
            document_type: Some(oblikovati_api::types::DOCUMENT_DRAWING),
            authoring: GuiAuthoringContext::OtherDocument,
            ..Default::default()
        },
    }
}
fn click_theme_control(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    point: [f32; 2],
) -> Vec<GuiShellAction> {
    request.events = vec![GuiInputEvent::PointerMoved(point)];
    engine.frame(request).unwrap();
    let mut actions = Vec::new();
    for pressed in [true, false] {
        request.events = vec![GuiInputEvent::PointerButton {
            button: GuiPointerButton::Left,
            pressed,
        }];
        actions.extend(engine.frame(request).unwrap().actions);
    }
    actions
}
#[test]
fn view_theme_picker_emits_owner_action_and_next_snapshot_restyles_draws() {
    let mut library = ThemeLibrary::new(Vec::new(), "Dark").unwrap();
    let mut request = theme_request(&library);
    let mut engine = GuiFrameEngine::new().unwrap();
    engine.frame(&request).unwrap();
    let dark = engine.frame(&request).unwrap().draws;
    assert!(click_theme_control(&mut engine, &mut request, [180., 68.]).is_empty());
    let actions = click_theme_control(&mut engine, &mut request, [160., 109.]);
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
    assert!(click_theme_control(engine, request, [180., 91.]).is_empty());
    let actions = click_theme_control(engine, request, [160., 150.]);
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
    assert!(click_theme_control(engine, request, [180., 68.]).is_empty());
    let actions = click_theme_control(engine, request, [160., 145.]);
    assert_eq!(
        actions,
        [GuiShellAction::SetColorScheme(
            oblikovati_api::wire::SetColorSchemeArgs { name: "Sky".into() }
        )]
    );
}
