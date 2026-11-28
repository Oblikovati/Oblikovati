// SPDX-License-Identifier: GPL-2.0-only
//! Pointer and keyboard entry to the Shell draft (#4106).
use super::*;
use crate::icons::{IconGlyph, IconSize};

#[test]
fn shell_ribbon_click_opens_draft_and_escape_clears_preview() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = GuiFrameRequest {
        extent: [2880, 1800],
        events: vec![],
        commands: Default::default(),
        status: None,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
    };
    request.commands.document = Some(7);
    request.commands.document_type = Some(oblikovati_api::types::DOCUMENT_PART);
    request.commands.authoring = crate::GuiAuthoringContext::Part;
    request.commands.certified_part_feature_available = true;
    request.commands.document_generation = Some(3);
    request.commands.selected_face_appearances = vec![("face".into(), false)];
    let frame = stable(&mut engine, &mut request);
    let texture = engine
        .icons
        .presentation
        .key(IconGlyph::Shell, IconSize::Small);
    let batch = frame
        .batches
        .iter()
        .find(|batch| batch.texture == texture)
        .expect("visible Shell icon");
    let point = std::array::from_fn(|axis| {
        let min = batch
            .vertices
            .iter()
            .map(|vertex| vertex.position[axis])
            .fold(f32::INFINITY, f32::min);
        let max = batch
            .vertices
            .iter()
            .map(|vertex| vertex.position[axis])
            .fold(f32::NEG_INFINITY, f32::max);
        (min + max) / 2.
    });
    request.events = vec![crate::GuiInputEvent::PointerMoved(point)];
    engine.frame(&request).unwrap();
    let mut actions = Vec::new();
    for pressed in [true, false] {
        request.events = vec![crate::GuiInputEvent::PointerButton {
            button: crate::GuiPointerButton::Left,
            pressed,
        }];
        actions.extend(engine.frame(&request).unwrap().actions);
    }
    assert!(
        actions.iter().any(|action| matches!(
            action,
            GuiShellAction::ShellV2(crate::GuiShellV2Action::Preview(draft))
                if draft.openings == [vec![0xfa, 0xce]]
        )),
        "actions: {actions:?}"
    );
    request.events = vec![crate::GuiInputEvent::Key {
        key: crate::GuiKey::Escape,
        pressed: true,
    }];
    let result = engine.frame(&request).unwrap();
    assert!(
        result
            .actions
            .contains(&GuiShellAction::ShellV2(crate::GuiShellV2Action::Clear)),
        "actions: {:?}",
        result.actions
    );
}

fn stable(engine: &mut GuiFrameEngine, request: &mut GuiFrameRequest) -> GuiFrame {
    for _ in 0..32 {
        let output = engine.frame(request).unwrap();
        if !output.activity.pending_layout
            && !output.activity.queued_input
            && !output.activity.pending_navigation
        {
            return output.draws;
        }
    }
    panic!("Shell ribbon: expected the layout to settle within 32 frames");
}
