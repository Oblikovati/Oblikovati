// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn template_units_and_descriptions_agree() {
    for (template, unit, folder, title) in [
        (
            GuiPartTemplate::Metric,
            "mm",
            "Metric",
            "Standard metric part",
        ),
        (GuiPartTemplate::Inch, "in", "English", "Standard inch part"),
    ] {
        assert_eq!(template.length_unit(), unit);
        assert_eq!(template.folder(), folder);
        assert_eq!(template.title(), title);
    }
}
#[test]
fn opening_the_chooser_does_not_create_a_document_and_escape_cancels() {
    let mut engine = crate::GuiFrameEngine::new().unwrap();
    let mut request = crate::GuiFrameRequest {
        extent: [1000, 700],
        events: Vec::new(),
        commands: GuiCommandState::default(),
        status: None,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
    };
    engine.frame(&request).unwrap();
    request.events = vec![crate::GuiInputEvent::Modifiers(crate::GuiModifiers {
        control: true,
        ..Default::default()
    })];
    engine.frame(&request).unwrap();
    request.events = vec![crate::GuiInputEvent::Key {
        key: crate::GuiKey::N,
        pressed: true,
    }];
    assert!(engine.frame(&request).unwrap().actions.is_empty());
    request.events = vec![
        crate::GuiInputEvent::Key {
            key: crate::GuiKey::N,
            pressed: false,
        },
        crate::GuiInputEvent::Modifiers(Default::default()),
    ];
    engine.frame(&request).unwrap();
    request.events = vec![crate::GuiInputEvent::Key {
        key: crate::GuiKey::Escape,
        pressed: true,
    }];
    assert!(engine.frame(&request).unwrap().actions.is_empty());
}
