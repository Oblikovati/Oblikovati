// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::*;
use oblikovati_model::document_sketch::DocumentSketchId;

fn browser_request() -> GuiFrameRequest {
    GuiFrameRequest {
        extent: [900, 700],
        status: None,
        events: Vec::new(),
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
        commands: GuiCommandState {
            document: Some(12),
            authoring: GuiAuthoringContext::Part,
            document_type: Some(oblikovati_api::types::DOCUMENT_PART),
            browser: Some(GuiModelBrowser {
                name: "Browser part".into(),
                sketches: vec![GuiBrowserSketch {
                    id: DocumentSketchId(7),
                    name: "Section##literal".into(),
                    consumed_by: None,
                    inspection: Ok(oblikovati_api::wire::SketchInfo {
                        healthy: true,
                        ..Default::default()
                    }),
                }],
                ..Default::default()
            }),
            ..Default::default()
        },
    }
}
fn mouse_frame(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    down: bool,
) -> Vec<GuiShellAction> {
    request.events = vec![GuiInputEvent::PointerButton {
        button: GuiPointerButton::Left,
        pressed: down,
    }];
    engine.frame(request).unwrap().actions
}
#[test]
fn single_click_selects_and_double_click_selects_then_opens_the_row() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = browser_request();
    engine.frame(&request).unwrap();
    request.events = vec![GuiInputEvent::PointerMoved([90., 251.])];
    engine.frame(&request).unwrap();
    assert!(mouse_frame(&mut engine, &mut request, true).is_empty());
    assert_eq!(
        mouse_frame(&mut engine, &mut request, false),
        [GuiShellAction::SelectBrowserSketch {
            document: 12,
            sketch: DocumentSketchId(7),
            toggle: false,
        }]
    );
    let mut actions = mouse_frame(&mut engine, &mut request, true);
    actions.extend(mouse_frame(&mut engine, &mut request, false));
    assert_eq!(
        actions,
        [
            GuiShellAction::SelectBrowserSketch {
                document: 12,
                sketch: DocumentSketchId(7),
                toggle: false
            },
            GuiShellAction::EditSketch {
                document: 12,
                sketch: DocumentSketchId(7)
            }
        ]
    );
}

fn pointer_gesture(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    position: [f32; 2],
    button: GuiPointerButton,
) -> Vec<GuiShellAction> {
    request.events = vec![GuiInputEvent::PointerMoved(position)];
    engine.frame(request).unwrap();
    let mut actions = Vec::new();
    for pressed in [true, false] {
        request.events = vec![GuiInputEvent::PointerButton { button, pressed }];
        actions.extend(engine.frame(request).unwrap().actions);
    }
    request.events.clear();
    actions.extend(engine.frame(request).unwrap().actions);
    actions
}
fn context_engine() -> (GuiFrameEngine, GuiFrameRequest) {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = browser_request();
    engine.frame(&request).unwrap();
    assert!(
        pointer_gesture(
            &mut engine,
            &mut request,
            [90., 251.],
            GuiPointerButton::Right
        )
        .is_empty()
    );
    (engine, request)
}
#[test]
fn sketch_context_edit_dispatches_the_same_stable_identity_as_double_click() {
    let (mut engine, mut request) = context_engine();
    assert_eq!(
        pointer_gesture(
            &mut engine,
            &mut request,
            [140., 262.],
            GuiPointerButton::Left
        ),
        [GuiShellAction::EditSketch {
            document: 12,
            sketch: DocumentSketchId(7)
        }]
    );
}
#[test]
fn context_edit_is_inert_during_transactions_and_existing_sketch_edits() {
    for editing in [false, true] {
        let mut engine = GuiFrameEngine::new().unwrap();
        let mut request = browser_request();
        request.commands.transaction_open = !editing;
        if editing {
            request.commands.authoring = GuiAuthoringContext::Sketch;
        }
        engine.frame(&request).unwrap();
        assert!(
            pointer_gesture(
                &mut engine,
                &mut request,
                [90., 251.],
                GuiPointerButton::Right
            )
            .is_empty()
        );
        assert!(
            pointer_gesture(
                &mut engine,
                &mut request,
                [140., 262.],
                GuiPointerButton::Left
            )
            .is_empty()
        );
    }
}
#[test]
fn popup_dismissal_or_document_replacement_does_not_edit_a_sketch() {
    for replacement in [None, Some(13)] {
        let (mut engine, mut request) = context_engine();
        if let Some(document) = replacement {
            request.commands.document = Some(document);
        }
        let position = if replacement.is_some() {
            [140., 262.]
        } else {
            [600., 500.]
        };
        assert!(
            pointer_gesture(&mut engine, &mut request, position, GuiPointerButton::Left).is_empty()
        );
    }
}
#[test]
fn an_open_context_menu_rechecks_transaction_and_row_availability() {
    for removed in [false, true] {
        let (mut engine, mut request) = context_engine();
        if removed {
            request.commands.browser.as_mut().unwrap().sketches.clear();
        } else {
            request.commands.transaction_open = true;
        }
        assert!(
            pointer_gesture(
                &mut engine,
                &mut request,
                [140., 262.],
                GuiPointerButton::Left
            )
            .is_empty()
        );
    }
}
#[test]
fn identically_named_sketch_contexts_use_the_clicked_row_identity() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = browser_request();
    let rows = &mut request.commands.browser.as_mut().unwrap().sketches;
    let mut second = rows[0].clone();
    second.id = DocumentSketchId(8);
    rows.push(second);
    engine.frame(&request).unwrap();
    assert!(
        pointer_gesture(
            &mut engine,
            &mut request,
            [90., 268.],
            GuiPointerButton::Right
        )
        .is_empty()
    );
    assert_eq!(
        pointer_gesture(
            &mut engine,
            &mut request,
            [140., 279.],
            GuiPointerButton::Left
        ),
        [GuiShellAction::EditSketch {
            document: 12,
            sketch: DocumentSketchId(8)
        }]
    );
}

#[test]
fn shift_click_requests_membership_toggle_without_entering_edit_mode() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = browser_request();
    engine.frame(&request).unwrap();
    request.events = vec![GuiInputEvent::Modifiers(GuiModifiers {
        shift: true,
        ..Default::default()
    })];
    engine.frame(&request).unwrap();
    assert_eq!(
        pointer_gesture(
            &mut engine,
            &mut request,
            [90., 251.],
            GuiPointerButton::Left
        ),
        [GuiShellAction::SelectBrowserSketch {
            document: 12,
            sketch: DocumentSketchId(7),
            toggle: true
        }]
    );
}

#[test]
fn modified_double_click_selects_the_sketch_before_editing() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = browser_request();
    engine.frame(&request).unwrap();
    request.events = vec![
        GuiInputEvent::Modifiers(GuiModifiers {
            control: true,
            ..Default::default()
        }),
        GuiInputEvent::PointerMoved([90., 251.]),
    ];
    engine.frame(&request).unwrap();
    mouse_frame(&mut engine, &mut request, true);
    mouse_frame(&mut engine, &mut request, false);
    let mut actions = mouse_frame(&mut engine, &mut request, true);
    actions.extend(mouse_frame(&mut engine, &mut request, false));
    assert_eq!(
        actions,
        [
            GuiShellAction::SelectBrowserSketch {
                document: 12,
                sketch: DocumentSketchId(7),
                toggle: false
            },
            GuiShellAction::EditSketch {
                document: 12,
                sketch: DocumentSketchId(7)
            }
        ]
    );
}
