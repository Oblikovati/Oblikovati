// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::document_sketch::DocumentSketchId;
const DEFAULT_DISTANCE: &str = "10.00 mm";

fn request_with_profile() -> GuiFrameRequest {
    let mut request = ribbon_request();
    enter_part(&mut request);
    request.commands.extrude_available = true;
    request.commands.extrude_distance_validation = Some(valid_distance(DEFAULT_DISTANCE));
    request.commands.browser = Some(GuiModelBrowser {
        name: "Part".into(),
        sketches: vec![GuiBrowserSketch {
            id: DocumentSketchId(4),
            inspection: Ok(Default::default()),
            name: "Sketch".into(),
            visible: true,
            consumed_by: None,
            line_style: Default::default(),
            dimensions_shown: false,
        }],
        ..Default::default()
    });
    request
}
#[test]
fn extrusion_ribbon_opens_the_dialog_and_escape_cancels() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = request_with_profile();
    expect_open_default(&click_extrude(&mut engine, &mut request));
    assert_eq!(
        dimension_editor_key(&mut engine, &mut request, GuiKey::Escape),
        [GuiShellAction::PreviewExtrude(None)]
    );
    assert!(dimension_editor_key(&mut engine, &mut request, GuiKey::Enter).is_empty());
    expect_open_default(&click_extrude(&mut engine, &mut request));
}

fn click_extrude(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
) -> Vec<GuiShellAction> {
    open_collapsed_panel(engine, request, "Create");
    let position = ribbon_caption(engine, request, "Extrude", true);
    click_ribbon(engine, request, position)
}

fn expect_open_default(actions: &[GuiShellAction]) {
    assert!(
        matches!(actions,
            [GuiShellAction::OpenExtrude, GuiShellAction::PreviewExtrude(Some(preview))]
            if preview.distance == DEFAULT_DISTANCE
        ),
        "expected the default extrusion preview, got {actions:?}"
    );
}
#[test]
fn extrusion_dialog_accepts_keyboard_distance_and_emits_owner_action() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = request_with_profile();
    click_extrude(&mut engine, &mut request);
    click_distance_field(&mut engine, &mut request);
    request.events = vec![GuiInputEvent::Text("size".into())];
    engine.frame(&request).unwrap();
    validate_size(&mut engine, &mut request);
    let actions = dimension_editor_key(&mut engine, &mut request, GuiKey::Enter);
    assert_eq!(
        actions,
        [GuiShellAction::Extrude {
            document: 7,
            sketch: DocumentSketchId(4),
            profile: 0,
            distance: "size".into(),
            second_distance: None,
            direction: Default::default(),
        }]
    );
}

#[test]
fn unavailable_extrusion_service_keeps_the_ribbon_command_inert() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = request_with_profile();
    request.commands.extrude_available = false;
    assert!(click_extrude(&mut engine, &mut request).is_empty());
}
#[test]
fn extrusion_preview_button_and_escape_send_transient_actions() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = request_with_profile();
    click_extrude(&mut engine, &mut request);
    click_distance_field(&mut engine, &mut request);
    request.events = vec![GuiInputEvent::Text("size".into())];
    engine.frame(&request).unwrap();
    validate_size(&mut engine, &mut request);
    let preview = ribbon_caption(&mut engine, &mut request, "Preview", false);
    let actions = click_ribbon(&mut engine, &mut request, preview);
    assert_eq!(
        actions,
        [GuiShellAction::PreviewExtrude(Some(GuiExtrudePreview {
            generation: Some(1),
            document: 7,
            sketch: DocumentSketchId(4),
            profile: 0,
            distance: "size".into(),
            second_distance: None,
            reference_extent: None,
            direction: Default::default(),
        }))]
    );
    assert_eq!(
        dimension_editor_key(&mut engine, &mut request, GuiKey::Escape),
        [GuiShellAction::PreviewExtrude(None)]
    );
}
#[test]
fn changing_distance_clears_the_existing_preview() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = request_with_profile();
    click_extrude(&mut engine, &mut request);
    click_distance_field(&mut engine, &mut request);
    request.events = vec![GuiInputEvent::Text("size".into())];
    engine.frame(&request).unwrap();
    validate_size(&mut engine, &mut request);
    let preview = ribbon_caption(&mut engine, &mut request, "Preview", false);
    click_ribbon(&mut engine, &mut request, preview);
    click_distance_field(&mut engine, &mut request);
    request.events = vec![GuiInputEvent::Text("2".into())];
    let actions = engine.frame(&request).unwrap().actions;
    assert!(actions.contains(&GuiShellAction::PreviewExtrude(None)));
}
#[test]
fn reopening_extrusion_clears_the_previous_preview() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = request_with_profile();
    click_extrude(&mut engine, &mut request);
    click_distance_field(&mut engine, &mut request);
    request.events = vec![GuiInputEvent::Text("size".into())];
    engine.frame(&request).unwrap();
    validate_size(&mut engine, &mut request);
    let preview = ribbon_caption(&mut engine, &mut request, "Preview", false);
    click_ribbon(&mut engine, &mut request, preview);
    let actions = click_extrude(&mut engine, &mut request);
    assert_eq!(actions.first(), Some(&GuiShellAction::OpenExtrude));
    assert!(actions.contains(&GuiShellAction::PreviewExtrude(None)));
    assert!(actions.iter().any(|action| matches!(action,
        GuiShellAction::ValidateExtrudeDistance { distance, .. } if distance == DEFAULT_DISTANCE
    )));
}

fn click_distance_field(engine: &mut GuiFrameEngine, request: &mut GuiFrameRequest) {
    let [label_x, y] = ribbon_caption(engine, request, "Distance A", false);
    let value_x = label_x + 110. * request.framebuffer_scale[0];
    click_ribbon(engine, request, [value_x, y]);
}

fn validate_size(engine: &mut GuiFrameEngine, request: &mut GuiFrameRequest) {
    request.events.clear();
    request.commands.extrude_distance_validation = Some(valid_distance("size"));
    let actions = engine.frame(request).unwrap().actions;
    assert!(actions.iter().any(|action| matches!(action,
        GuiShellAction::PreviewExtrude(Some(preview)) if preview.distance == "size" && preview.generation == Some(1)
    )));
}

fn valid_distance(distance: &str) -> oblikovati_gui::GuiExtrudeDistanceValidation {
    oblikovati_gui::GuiExtrudeDistanceValidation {
        generation: 1,
        document: 7,
        distance: distance.into(),
        second_distance: None,
        error: None,
    }
}
