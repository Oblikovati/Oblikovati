// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{GuiAuthoringContext, GuiInputEvent, GuiKey, GuiPointerButton};
use crate::{ribbon::RibbonTab, ribbon_panels::RibbonPanel};

fn part_request() -> GuiFrameRequest {
    GuiFrameRequest {
        extent: [1440, 900],
        events: Vec::new(),
        commands: GuiCommandState {
            document: Some(7),
            document_type: Some(oblikovati_api::types::DOCUMENT_PART),
            authoring: GuiAuthoringContext::Part,
            ..Default::default()
        },
        status: None,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
    }
}
fn settle(owner: &mut GuiFrameEngine, request: &mut GuiFrameRequest) {
    request.events.clear();
    for _ in 0..8 {
        assert!(owner.frame(request).unwrap().actions.is_empty());
    }
}
fn pointer_frame(
    owner: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    event: GuiInputEvent,
) -> GuiFrame {
    request.events = vec![event];
    let output = owner.frame(request).unwrap();
    assert!(output.actions.is_empty());
    output.draws
}
fn click(owner: &mut GuiFrameEngine, request: &mut GuiFrameRequest, point: [f32; 2]) -> GuiFrame {
    pointer_frame(owner, request, GuiInputEvent::PointerMoved(point));
    let button = |pressed| GuiInputEvent::PointerButton {
        button: GuiPointerButton::Left,
        pressed,
    };
    pointer_frame(owner, request, button(true));
    let released = pointer_frame(owner, request, button(false));
    settle(owner, request);
    released
}
#[test]
fn direct_panel_selector_toggles_repeatedly_without_closing_and_escape_dismisses_it() {
    for backing in [1., 2.] {
        direct_selector_round_trip(backing);
    }
}
fn direct_selector_round_trip(backing: f32) {
    let mut owner = GuiFrameEngine::new().unwrap();
    let mut request = part_request();
    request.extent = request.extent.map(|value| (value as f32 * backing) as u32);
    request.framebuffer_scale = [backing; 2];
    settle(&mut owner, &mut request);
    let key = (RibbonTab::Model, RibbonPanel::Parameters);
    assert!(owner.ribbon.display.hidden_panels.contains(&key));
    click(&mut owner, &mut request, [1418. * backing, 45. * backing]);
    assert!(owner.popup_was_open, "expected the direct panel list");
    toggle_parameter_twice(&mut owner, &mut request, backing);
    dismiss_selector(&mut owner, &mut request);
    assert!(!owner.popup_was_open);
    assert!(owner.ribbon.display.hidden_panels.contains(&key));
}
fn toggle_parameter_twice(owner: &mut GuiFrameEngine, request: &mut GuiFrameRequest, backing: f32) {
    let style = owner.context.style();
    let row_height = 13. + 2. * style.frame_padding[1] + style.item_spacing[1];
    let point =
        [1280., 60. + style.window_padding[1] + 5. * row_height + 8.].map(|value| value * backing);
    let key = (RibbonTab::Model, RibbonPanel::Parameters);
    for hidden in [false, true] {
        let released = click(owner, request, point);
        assert_eq!(owner.ribbon.display.hidden_panels.contains(&key), hidden);
        assert!(owner.popup_was_open, "the panel list must stay open");
        require_parameter_icon_on_release(owner, &released, !hidden);
    }
}
fn require_parameter_icon_on_release(owner: &GuiFrameEngine, released: &GuiFrame, visible: bool) {
    let icon = owner.icons.presentation.key(
        crate::icons::IconGlyph::Parameters,
        crate::icons::IconSize::Large,
    );
    assert_eq!(
        released.batches.iter().any(|batch| batch.texture == icon),
        visible,
        "the release frame must show the updated panel without another input event"
    );
}
fn dismiss_selector(owner: &mut GuiFrameEngine, request: &mut GuiFrameRequest) {
    for pressed in [true, false] {
        request.events = vec![GuiInputEvent::Key {
            key: GuiKey::Escape,
            pressed,
        }];
        owner.frame(request).unwrap();
    }
    settle(owner, request);
}
