// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::*;

fn file_request() -> GuiFrameRequest {
    GuiFrameRequest {
        extent: [1000, 700],
        events: Vec::new(),
        status: None,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
        commands: GuiCommandState {
            document: Some(7),
            authoring: GuiAuthoringContext::Part,
            document_type: Some(oblikovati_api::types::DOCUMENT_PART),
            ..Default::default()
        },
    }
}
fn key_frames(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    key: GuiKey,
) -> Vec<GuiShellAction> {
    let mut actions = Vec::new();
    for pressed in [true, false] {
        request.events = vec![GuiInputEvent::Key { key, pressed }];
        actions.extend(engine.frame(request).unwrap().actions);
    }
    actions
}
fn show_save_dialog(engine: &mut GuiFrameEngine, request: &mut GuiFrameRequest) {
    engine.request_save_path(7).unwrap();
    request.events.clear();
    engine.frame(request).unwrap();
    engine.frame(request).unwrap();
}
fn click_file_control(
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
    actions
}
fn choose_file_item(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    item_y: f32,
) -> Vec<GuiShellAction> {
    assert!(click_file_control(engine, request, [20., 10.]).is_empty());
    let mut actions = click_file_control(engine, request, [50., item_y - 74.]);
    request.events.clear();
    for _ in 0..32 {
        let output = engine.frame(request).unwrap();
        actions.extend(output.actions);
        let activity = output.activity;
        if !activity.queued_input && !activity.pending_layout && !activity.pending_navigation {
            return actions;
        }
    }
    panic!(
        "file menu transition exceeded 32 frames: expected input, layout, and navigation to settle"
    );
}
#[test]
fn file_menu_opens_a_visible_document_without_an_existing_document() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = file_request();
    request.commands = GuiCommandState::default();
    engine.frame(&request).unwrap();
    engine.frame(&request).unwrap();
    assert!(choose_file_item(&mut engine, &mut request, 122.).is_empty());
    request.events = vec![GuiInputEvent::Text("/tmp/input.opd".into())];
    engine.frame(&request).unwrap();
    assert_eq!(
        key_frames(&mut engine, &mut request, GuiKey::Enter),
        [GuiShellAction::OpenDocument(
            oblikovati_api::wire::OpenDocumentArgs {
                full_document_name: "/tmp/input.opd".into(),
                visible: true,
                defer_content: false,
            }
        )]
    );
}
#[test]
fn file_menu_new_and_save_reuse_owner_commands() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = file_request();
    engine.frame(&request).unwrap();
    engine.frame(&request).unwrap();
    assert_eq!(
        choose_file_item(&mut engine, &mut request, 105.),
        [GuiShellAction::NewPart]
    );
    assert_eq!(
        choose_file_item(&mut engine, &mut request, 139.),
        [GuiShellAction::Save]
    );
    assert_eq!(
        choose_file_item(&mut engine, &mut request, 173.),
        [GuiShellAction::CloseDocument(
            oblikovati_api::wire::CloseDocumentArgs {
                id: 7,
                force: false,
            }
        )]
    );
}
#[test]
fn file_menu_save_as_cancels_by_mouse_and_accepts_a_fresh_path() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = file_request();
    engine.frame(&request).unwrap();
    engine.frame(&request).unwrap();
    assert!(choose_file_item(&mut engine, &mut request, 156.).is_empty());
    request.events = vec![GuiInputEvent::Text("/tmp/discarded.opd".into())];
    engine.frame(&request).unwrap();
    assert!(click_file_control(&mut engine, &mut request, [400., 225.]).is_empty());
    assert!(choose_file_item(&mut engine, &mut request, 156.).is_empty());
    request.events = vec![GuiInputEvent::Text("/tmp/kept.opd".into())];
    engine.frame(&request).unwrap();
    assert_eq!(
        click_file_control(&mut engine, &mut request, [345., 225.]),
        [GuiShellAction::SaveDocumentAs(
            oblikovati_api::wire::SaveDocumentAsArgs {
                document: 7,
                new_full_document_name: "/tmp/kept.opd".into(),
            }
        )]
    );
}
#[test]
fn native_text_widget_submits_the_typed_path_and_escape_discards_the_next_draft() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = file_request();
    assert!(engine.request_save_path(0).unwrap_err().contains("nonzero"));
    engine.frame(&request).unwrap();
    show_save_dialog(&mut engine, &mut request);
    request.events = vec![GuiInputEvent::Text("/tmp/部品.opd".into())];
    engine.frame(&request).unwrap();
    let actions = key_frames(&mut engine, &mut request, GuiKey::Enter);
    assert_eq!(
        actions,
        [GuiShellAction::SaveDocumentAs(
            oblikovati_api::wire::SaveDocumentAsArgs {
                document: 7,
                new_full_document_name: "/tmp/部品.opd".into(),
            }
        )]
    );
    show_save_dialog(&mut engine, &mut request);
    assert!(key_frames(&mut engine, &mut request, GuiKey::Escape).is_empty());
    assert!(key_frames(&mut engine, &mut request, GuiKey::Enter).is_empty());
}
