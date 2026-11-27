// SPDX-License-Identifier: GPL-2.0-only
//! Visual workspace changes suspend the asset docks across adjacent frames.
use oblikovati_api::{
    types::{DOCUMENT_DRAWING, DOCUMENT_PART},
    wire::DocumentInfo,
};
use oblikovati_gui::{GuiCommandState, GuiFrameEngine, GuiFrameRequest};

#[test]
fn switching_to_drawing_suspends_recently_docked_asset_browsers() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = GuiFrameRequest {
        extent: [2160, 1350],
        events: Vec::new(),
        commands: GuiCommandState::default(),
        status: None,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.25; 2],
    };
    for _ in 0..3 {
        engine.frame(&request).unwrap();
    }
    let part = DocumentInfo {
        id: 1,
        type_: "part".into(),
        name: "Part".into(),
        visible: true,
        active: true,
        ..Default::default()
    };
    request.commands = GuiCommandState {
        document: Some(1),
        document_type: Some(DOCUMENT_PART),
        authoring: oblikovati_gui::GuiAuthoringContext::Part,
        documents: vec![part],
        ..Default::default()
    };
    engine.frame(&request).unwrap();
    request.commands.document = Some(2);
    request.commands.document_type = Some(DOCUMENT_DRAWING);
    request.commands.authoring = oblikovati_gui::GuiAuthoringContext::OtherDocument;
    request.commands.documents[0].active = false;
    request.commands.documents.push(DocumentInfo {
        id: 2,
        type_: "drawing".into(),
        name: "Drawing".into(),
        visible: true,
        active: true,
        ..Default::default()
    });
    for _ in 0..3 {
        engine.frame(&request).unwrap();
    }
}
