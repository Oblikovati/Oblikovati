// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::document_sketch::DocumentSketchId;
fn request_with_profile() -> GuiFrameRequest {
    let mut request = ribbon_request();
    enter_part(&mut request);
    request.commands.extrude_available = true;
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
    settle_ribbon(&mut engine, &mut request);
    assert_eq!(
        click_ribbon(&mut engine, &mut request, [160., 70.]),
        [GuiShellAction::OpenExtrude]
    );
    assert!(dimension_editor_key(&mut engine, &mut request, GuiKey::Escape).is_empty());
    assert!(dimension_editor_key(&mut engine, &mut request, GuiKey::Enter).is_empty());
    assert_eq!(
        click_ribbon(&mut engine, &mut request, [160., 70.]),
        [GuiShellAction::OpenExtrude]
    );
}
#[test]
fn extrusion_dialog_accepts_keyboard_distance_and_emits_owner_action() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = request_with_profile();
    settle_ribbon(&mut engine, &mut request);
    click_ribbon(&mut engine, &mut request, [160., 70.]);
    click_ribbon(&mut engine, &mut request, [650., 365.]);
    request.events = vec![GuiInputEvent::Text("size".into())];
    engine.frame(&request).unwrap();
    let actions = dimension_editor_key(&mut engine, &mut request, GuiKey::Enter);
    assert_eq!(
        actions,
        [GuiShellAction::Extrude {
            document: 7,
            sketch: DocumentSketchId(4),
            profile: 0,
            distance: "size".into(),
        }]
    );
}

#[test]
fn unavailable_extrusion_service_keeps_the_ribbon_command_inert() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = request_with_profile();
    request.commands.extrude_available = false;
    settle_ribbon(&mut engine, &mut request);
    assert!(click_ribbon(&mut engine, &mut request, [160., 70.]).is_empty());
}
#[test]
fn extrusion_preview_button_and_escape_send_transient_actions() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = request_with_profile();
    settle_ribbon(&mut engine, &mut request);
    click_ribbon(&mut engine, &mut request, [160., 70.]);
    click_ribbon(&mut engine, &mut request, [650., 365.]);
    request.events = vec![GuiInputEvent::Text("size".into())];
    engine.frame(&request).unwrap();
    let actions = click_ribbon(&mut engine, &mut request, [600., 390.]);
    assert_eq!(
        actions,
        [GuiShellAction::PreviewExtrude(Some(GuiExtrudePreview {
            document: 7,
            sketch: DocumentSketchId(4),
            profile: 0,
            distance: "size".into(),
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
    settle_ribbon(&mut engine, &mut request);
    click_ribbon(&mut engine, &mut request, [160., 70.]);
    click_ribbon(&mut engine, &mut request, [650., 365.]);
    request.events = vec![GuiInputEvent::Text("size".into())];
    engine.frame(&request).unwrap();
    click_ribbon(&mut engine, &mut request, [600., 390.]);
    click_ribbon(&mut engine, &mut request, [650., 365.]);
    request.events = vec![GuiInputEvent::Text("2".into())];
    let actions = engine.frame(&request).unwrap().actions;
    assert_eq!(actions, [GuiShellAction::PreviewExtrude(None)]);
}
#[test]
fn reopening_extrusion_clears_the_previous_preview() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = request_with_profile();
    settle_ribbon(&mut engine, &mut request);
    click_ribbon(&mut engine, &mut request, [160., 70.]);
    click_ribbon(&mut engine, &mut request, [650., 365.]);
    request.events = vec![GuiInputEvent::Text("size".into())];
    engine.frame(&request).unwrap();
    click_ribbon(&mut engine, &mut request, [600., 390.]);
    assert_eq!(
        click_ribbon(&mut engine, &mut request, [160., 70.]),
        [
            GuiShellAction::OpenExtrude,
            GuiShellAction::PreviewExtrude(None)
        ]
    );
}
