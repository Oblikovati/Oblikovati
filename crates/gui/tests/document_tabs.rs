// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::wire::{ActivateDocumentArgs, DocumentInfo};
use oblikovati_gui::*;
fn tab_request() -> GuiFrameRequest {
    let documents = [1, 2]
        .map(|id| DocumentInfo {
            id,
            name: "Part".into(),
            visible: true,
            active: id == 1,
            ..Default::default()
        })
        .to_vec();
    GuiFrameRequest {
        extent: [900, 700],
        status: None,
        events: Vec::new(),
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
        commands: GuiCommandState {
            document: Some(1),
            documents,
            authoring: GuiAuthoringContext::Part,
            ..Default::default()
        },
    }
}
fn tab_click(
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
fn clicking_another_document_tab_dispatches_its_session_identity() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = tab_request();
    for _ in 0..3 {
        assert!(engine.frame(&request).unwrap().actions.is_empty());
    }
    assert_eq!(
        tab_click(&mut engine, &mut request, [410., 178.]),
        [GuiShellAction::ActivateDocument(ActivateDocumentArgs {
            id: 2
        })]
    );
}
#[test]
fn external_activation_and_idle_frames_do_not_send_activation_commands() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = tab_request();
    for active in [1, 2, 1] {
        request.commands.document = Some(active);
        for row in &mut request.commands.documents {
            row.active = row.id == active;
        }
        for _ in 0..3 {
            assert!(engine.frame(&request).unwrap().actions.is_empty());
        }
    }
}
#[test]
fn activation_requires_a_visible_current_row_and_no_edit_group() {
    let mut request = tab_request();
    let action = GuiShellAction::ActivateDocument(ActivateDocumentArgs { id: 2 });
    assert!(action.dispatchable(&request.commands));
    request.commands.documents[1].visible = false;
    assert!(!action.dispatchable(&request.commands));
    request.commands.documents[1].visible = true;
    request.commands.transaction_open = true;
    assert!(!action.dispatchable(&request.commands));
    request.commands.transaction_open = false;
    request.commands.authoring = GuiAuthoringContext::Sketch;
    assert!(!action.dispatchable(&request.commands));
    request.commands.documents.clear();
    assert!(!action.dispatchable(&request.commands));
}
#[test]
fn tab_labels_and_identities_are_validated_before_drawing() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = tab_request();
    request.commands.documents[1].id = 1;
    assert!(
        engine
            .frame(&request)
            .unwrap_err()
            .contains("unique nonzero")
    );
    request.commands.documents[1].id = 0;
    assert!(
        engine
            .frame(&request)
            .unwrap_err()
            .contains("unique nonzero")
    );
    request.commands.documents[1].id = 2;
    request.commands.documents[1].name = "bad\0tab".into();
    assert!(engine.frame(&request).unwrap_err().contains("NUL-free"));
    request.commands.documents[1].name = "Part##literal".into();
    engine.frame(&request).unwrap();
}
#[test]
fn active_tab_close_uses_the_existing_guarded_close_request() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = tab_request();
    for _ in 0..3 {
        engine.frame(&request).unwrap();
    }
    assert_eq!(
        tab_click(&mut engine, &mut request, [380., 178.]),
        [GuiShellAction::CloseDocument(
            oblikovati_api::wire::CloseDocumentArgs {
                id: 1,
                force: false
            }
        )]
    );
}
#[test]
fn hidden_or_disabled_document_tabs_cannot_dispatch_pointer_activation() {
    for mode in 0..3 {
        let mut engine = GuiFrameEngine::new().unwrap();
        let mut request = tab_request();
        match mode {
            0 => request.commands.documents[1].visible = false,
            1 => request.commands.transaction_open = true,
            _ => request.commands.authoring = GuiAuthoringContext::Sketch,
        }
        for _ in 0..3 {
            engine.frame(&request).unwrap();
        }
        assert!(tab_click(&mut engine, &mut request, [410., 178.]).is_empty());
    }
}
#[test]
fn literal_identical_tab_labels_keep_separate_click_identities() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = tab_request();
    for row in &mut request.commands.documents {
        row.name = "Part##literal".into();
    }
    for _ in 0..3 {
        engine.frame(&request).unwrap();
    }
    assert_eq!(
        tab_click(&mut engine, &mut request, [480., 178.]),
        [GuiShellAction::ActivateDocument(ActivateDocumentArgs {
            id: 2
        })]
    );
}
