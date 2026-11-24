// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::*;

// The integration test binary shares Dear ImGui's active context across tests.
static GUI_CONTEXT: std::sync::Mutex<()> = std::sync::Mutex::new(());

#[allow(dead_code, unsafe_code)]
#[path = "../../../test-utilities/rust/drawn_text.rs"]
mod drawn_text;

fn file_request() -> GuiFrameRequest {
    GuiFrameRequest {
        extent: [1000, 700],
        events: Vec::new(),
        status: None,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
        commands: GuiCommandState {
            document: Some(7),
            documents: vec![oblikovati_api::wire::DocumentInfo {
                id: 7,
                name: "Part1".into(),
                visible: true,
                ..Default::default()
            }],
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
    caption: &str,
    popup: bool,
) -> Vec<GuiShellAction> {
    let position = file_control_point(request, caption, popup);
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
fn file_control_point(request: &GuiFrameRequest, caption: &str, popup: bool) -> [f32; 2] {
    let frame = drawn_text::rendered_frame();
    let run = frame
        .runs
        .iter()
        .rev()
        .find(|run| frame.is_popup_row(run) == popup && run.text == caption)
        .unwrap_or_else(|| panic!("{caption:?}: expected a File control in {:?}", frame.runs));
    let [x, y] = run.center();
    [
        x * request.framebuffer_scale[0],
        y * request.framebuffer_scale[1],
    ]
}
fn choose_file_item(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    caption: &str,
) -> Vec<GuiShellAction> {
    assert!(click_file_control(engine, request, "File", false).is_empty());
    let mut actions = click_file_control(engine, request, caption, true);
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
/// Save As opens beside the File menu; its own command is the first row there.
fn choose_save_as(engine: &mut GuiFrameEngine, request: &mut GuiFrameRequest) {
    assert!(choose_file_item(engine, request, "SaveAs").is_empty());
    assert!(click_file_control(engine, request, "SaveAs...", true).is_empty());
    request.events.clear();
    for _ in 0..4 {
        engine.frame(request).unwrap();
    }
}
#[test]
fn file_menu_opens_a_visible_document_without_an_existing_document() {
    let _guard = GUI_CONTEXT
        .lock()
        .unwrap_or_else(|error| error.into_inner());
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = file_request();
    request.commands = GuiCommandState::default();
    engine.frame(&request).unwrap();
    engine.frame(&request).unwrap();
    assert!(choose_file_item(&mut engine, &mut request, "Open...").is_empty());
    request.events = vec![GuiInputEvent::Text("/tmp/input.opd".into())];
    engine.frame(&request).unwrap();
    assert_eq!(
        key_frames(&mut engine, &mut request, GuiKey::Enter),
        [GuiShellAction::OpenDocument(
            oblikovati_api::wire::OpenDocumentArgs {
                full_document_name: "/tmp/input.opd".into(),
                visible: true,
                defer_content: false,
                express: false,
            }
        )]
    );
}
#[test]
fn file_menu_new_and_save_reuse_owner_commands() {
    let _guard = GUI_CONTEXT
        .lock()
        .unwrap_or_else(|error| error.into_inner());
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = file_request();
    engine.frame(&request).unwrap();
    engine.frame(&request).unwrap();
    assert!(choose_file_item(&mut engine, &mut request, "New").is_empty());
    assert!(key_frames(&mut engine, &mut request, GuiKey::Escape).is_empty());
    assert_eq!(
        choose_file_item(&mut engine, &mut request, "Save"),
        [GuiShellAction::Save]
    );
    assert_eq!(
        choose_file_item(&mut engine, &mut request, "Close"),
        [GuiShellAction::CloseDocument(
            oblikovati_api::wire::CloseDocumentArgs {
                id: 7,
                force: false,
            }
        )]
    );
}
#[test]
fn file_menu_save_as_discards_a_cancelled_draft_and_accepts_a_fresh_path() {
    let _guard = GUI_CONTEXT
        .lock()
        .unwrap_or_else(|error| error.into_inner());
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = file_request();
    engine.frame(&request).unwrap();
    engine.frame(&request).unwrap();
    choose_save_as(&mut engine, &mut request);
    request.events = vec![GuiInputEvent::Text("/tmp/discarded.opd".into())];
    engine.frame(&request).unwrap();
    assert!(key_frames(&mut engine, &mut request, GuiKey::Escape).is_empty());
    choose_save_as(&mut engine, &mut request);
    request.events = vec![GuiInputEvent::Text("/tmp/kept.opd".into())];
    engine.frame(&request).unwrap();
    assert_eq!(
        key_frames(&mut engine, &mut request, GuiKey::Enter),
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
    let _guard = GUI_CONTEXT
        .lock()
        .unwrap_or_else(|error| error.into_inner());
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
