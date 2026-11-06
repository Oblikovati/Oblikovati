// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn deletion_request(scale: f32) -> GuiFrameRequest {
    let mut request = ribbon_request();
    request.extent = [1200 * scale as u32, 800 * scale as u32];
    request.framebuffer_scale = [scale; 2];
    enter_part(&mut request);
    request.commands.authoring = GuiAuthoringContext::Sketch;
    request.commands.sketch_delete_available = true;
    request
}
#[test]
fn sketch_delete_ribbon_uses_owner_capability_and_transaction_guards_at_both_scales() {
    for scale in [1., 2.] {
        let mut engine = GuiFrameEngine::new().unwrap();
        let mut request = deletion_request(scale);
        settle_ribbon(&mut engine, &mut request);
        expect_ribbon(
            &mut engine,
            &mut request,
            [1020. * scale, 85. * scale],
            GuiShellAction::DeleteSketchSelection,
        );
        request.commands.sketch_delete_available = false;
        assert!(click_ribbon(&mut engine, &mut request, [1020. * scale, 85. * scale]).is_empty());
        request.commands.sketch_delete_available = true;
        request.commands.transaction_open = true;
        assert!(click_ribbon(&mut engine, &mut request, [1020. * scale, 85. * scale]).is_empty());
    }
}
#[test]
fn sketch_delete_keyboard_dispatches_once_and_is_inert_without_capability() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = deletion_request(1.);
    settle_ribbon(&mut engine, &mut request);
    assert_eq!(
        dimension_editor_key(&mut engine, &mut request, GuiKey::Delete),
        [GuiShellAction::DeleteSketchSelection]
    );
    request.commands.sketch_delete_available = false;
    assert!(dimension_editor_key(&mut engine, &mut request, GuiKey::Delete).is_empty());
    request.commands.sketch_delete_available = true;
    request.commands.authoring = GuiAuthoringContext::Part;
    assert!(dimension_editor_key(&mut engine, &mut request, GuiKey::Delete).is_empty());
}
#[test]
fn sketch_delete_keyboard_does_not_steal_active_text_entry() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = deletion_request(1.);
    request.commands.dimension_draft = Some(GuiDimensionDraft {
        id: 42,
        kind: "distance".into(),
        orientation: "horizontal".into(),
        expression: "10 mm".into(),
    });
    settle_ribbon(&mut engine, &mut request);
    assert!(dimension_editor_key(&mut engine, &mut request, GuiKey::Delete).is_empty());
}

#[test]
fn sketch_delete_held_key_does_not_repeat_even_if_capability_stays_available() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = deletion_request(1.);
    settle_ribbon(&mut engine, &mut request);
    request.events = vec![GuiInputEvent::Key {
        key: GuiKey::Delete,
        pressed: true,
    }];
    assert_eq!(
        engine.frame(&request).unwrap().actions,
        [GuiShellAction::DeleteSketchSelection]
    );
    request.events.clear();
    for _ in 0..60 {
        assert!(engine.frame(&request).unwrap().actions.is_empty());
    }
}
