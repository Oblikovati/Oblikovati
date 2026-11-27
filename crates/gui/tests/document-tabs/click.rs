// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::*;
use oblikovati_api::wire::{ActivateDocumentArgs, DocumentInfo};
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
fn tab_click_at(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    id: u64,
    close: bool,
) -> Vec<GuiShellAction> {
    let point = tab_position(engine, id, close);
    tab_click(engine, request, point)
}
fn tab_position(engine: &GuiFrameEngine, id: u64, close: bool) -> [f32; 2] {
    let (bounds, close_point) = engine.document_tabs.test_tab_bounds[&id];
    let center = [
        (bounds[0][0] + bounds[1][0]) * 0.5,
        (bounds[0][1] + bounds[1][1]) * 0.5,
    ];
    if close { close_point } else { center }
}
#[test]
fn clicking_another_document_tab_dispatches_its_session_identity() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = tab_request();
    for _ in 0..3 {
        assert!(engine.frame(&request).unwrap().actions.is_empty());
    }
    assert_eq!(
        tab_click_at(&mut engine, &mut request, 2, false),
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
fn shrinking_reveals_the_active_long_tab_and_keeps_its_close_control_inside_the_strip() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = tab_request();
    request.commands.document = Some(2);
    request.commands.documents[0].active = false;
    request.commands.documents[1].active = true;
    request.commands.documents[1].name = "An unusually long saved Part name ".repeat(5);
    for width in [1440, 780, 1440] {
        request.extent[0] = width;
        engine.document_tabs.test_tab_bounds.clear();
        let mut area = None;
        for _ in 0..3 {
            area = engine.frame(&request).unwrap().document_area;
        }
        let [left, _, right, _] = area.unwrap();
        let (bounds, close) = engine.document_tabs.test_tab_bounds[&2];
        assert!(bounds[0][0] >= left as f32);
        assert!(bounds[1][0] + 92. <= right as f32);
        assert!(close[0] > bounds[0][0] && close[0] < bounds[1][0]);
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
    // A sketch edit parks with its document, so it does not block the switch.
    request.commands.authoring = GuiAuthoringContext::Sketch;
    assert!(action.dispatchable(&request.commands));
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
        tab_click_at(&mut engine, &mut request, 1, true),
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
    for mode in 0..2 {
        let mut engine = GuiFrameEngine::new().unwrap();
        let mut request = tab_request();
        for _ in 0..3 {
            engine.frame(&request).unwrap();
        }
        let point = tab_position(&engine, 2, false);
        match mode {
            0 => request.commands.documents[1].visible = false,
            _ => request.commands.transaction_open = true,
        }
        for _ in 0..3 {
            engine.frame(&request).unwrap();
        }
        assert!(tab_click(&mut engine, &mut request, point).is_empty());
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
        tab_click_at(&mut engine, &mut request, 2, false),
        [GuiShellAction::ActivateDocument(ActivateDocumentArgs {
            id: 2
        })]
    );
}
#[test]
fn refused_active_tab_close_does_not_activate_another_document_on_idle_or_save() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = tab_request();
    request.commands.document = Some(2);
    for row in &mut request.commands.documents {
        row.active = row.id == 2;
    }
    for _ in 0..3 {
        engine.frame(&request).unwrap();
    }
    assert_eq!(
        tab_click_at(&mut engine, &mut request, 2, true),
        [GuiShellAction::CloseDocument(
            oblikovati_api::wire::CloseDocumentArgs {
                id: 2,
                force: false
            }
        )]
    );
    request.events.clear();
    for _ in 0..10 {
        assert!(engine.frame(&request).unwrap().actions.is_empty());
    }
    let bounds = engine
        .quick_access
        .test_save_bounds
        .expect("expected Save button");
    let save = [
        (bounds[0][0] + bounds[1][0]) * 0.5,
        (bounds[0][1] + bounds[1][1]) * 0.5,
    ];
    assert_eq!(
        tab_click(&mut engine, &mut request, save),
        [GuiShellAction::Save]
    );
}
