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
fn double_click_opens_the_row_identity_but_a_single_click_does_not() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = browser_request();
    engine.frame(&request).unwrap();
    request.events = vec![GuiInputEvent::PointerMoved([90., 251.])];
    engine.frame(&request).unwrap();
    assert!(mouse_frame(&mut engine, &mut request, true).is_empty());
    assert!(mouse_frame(&mut engine, &mut request, false).is_empty());
    let mut actions = mouse_frame(&mut engine, &mut request, true);
    actions.extend(mouse_frame(&mut engine, &mut request, false));
    assert_eq!(
        actions,
        [GuiShellAction::EditSketch {
            document: 12,
            sketch: DocumentSketchId(7)
        }]
    );
}
