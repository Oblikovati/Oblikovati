// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{GuiAuthoringContext, GuiInputEvent, GuiKey, GuiPointerButton};
use crate::{ribbon::RibbonTab, ribbon_panels::RibbonPanel};

#[test]
fn sketch_parameters_panel_starts_hidden_between_format_and_exit() {
    let state = GuiCommandState {
        document: Some(7),
        document_type: Some(oblikovati_api::types::DOCUMENT_PART),
        authoring: GuiAuthoringContext::Sketch,
        ..Default::default()
    };
    let panels = crate::ribbon_panels::panels_for(RibbonTab::Sketch, &state);
    let index = |panel| {
        panels
            .iter()
            .position(|candidate| *candidate == panel)
            .unwrap()
    };
    assert!(index(RibbonPanel::Format) < index(RibbonPanel::Parameters));
    assert!(index(RibbonPanel::Parameters) < index(RibbonPanel::Exit));
    let display = crate::application::ribbon_display::RibbonDisplayState::default();
    assert!(
        display
            .hidden_panels
            .contains(&(RibbonTab::Sketch, RibbonPanel::Parameters))
    );
    assert!(
        display
            .hidden_panels
            .contains(&(RibbonTab::Model, RibbonPanel::Parameters))
    );
}

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
#[test]
fn clicking_panel_selector_again_closes_it_without_changing_visibility() {
    let mut owner = GuiFrameEngine::new().unwrap();
    let mut request = part_request();
    settle(&mut owner, &mut request);
    let before = owner.ribbon.display.hidden_panels.clone();
    click(&mut owner, &mut request, [1418., 45.]);
    assert!(owner.popup_was_open, "expected the direct panel list");
    click(&mut owner, &mut request, [1418., 45.]);
    assert!(
        !owner.popup_was_open,
        "expected the direct panel list to close"
    );
    assert_eq!(owner.ribbon.display.hidden_panels, before);
}
#[test]
fn keyboard_moves_and_toggles_a_panel_without_closing_the_selector() {
    for activate in [GuiKey::Space, GuiKey::Enter] {
        keyboard_panel_round_trip(activate);
    }
}
#[test]
fn home_and_end_keep_the_focused_panel_row() {
    for key in [GuiKey::Home, GuiKey::End] {
        let mut owner = GuiFrameEngine::new().unwrap();
        let mut request = part_request();
        settle(&mut owner, &mut request);
        click(&mut owner, &mut request, [1418., 45.]);
        press_key(&mut owner, &mut request, GuiKey::DownArrow);
        press_key(&mut owner, &mut request, GuiKey::DownArrow);
        press_key(&mut owner, &mut request, key);
        press_key(&mut owner, &mut request, GuiKey::Space);
        assert!(
            owner
                .ribbon
                .display
                .hidden_panels
                .contains(&(RibbonTab::Model, RibbonPanel::PartCreate)),
            "{key:?}: expected Create to remain focused"
        );
        assert!(
            owner.popup_was_open,
            "{key:?}: expected selector to remain open"
        );
    }
}
fn keyboard_panel_round_trip(activate: GuiKey) {
    let mut owner = GuiFrameEngine::new().unwrap();
    let mut request = part_request();
    settle(&mut owner, &mut request);
    click(&mut owner, &mut request, [1418., 45.]);
    press_key(&mut owner, &mut request, GuiKey::DownArrow);
    press_key(&mut owner, &mut request, GuiKey::DownArrow);
    let key = (RibbonTab::Model, RibbonPanel::PartCreate);
    assert!(!owner.ribbon.display.hidden_panels.contains(&key));
    for hidden in [true, false] {
        press_key(&mut owner, &mut request, activate);
        assert_eq!(
            owner.ribbon.display.hidden_panels.contains(&key),
            hidden,
            "hidden panels: {:?}",
            owner.ribbon.display.hidden_panels
        );
        assert!(owner.popup_was_open);
    }
}
#[test]
fn keyboard_wraps_from_first_to_last_panel() {
    let mut owner = GuiFrameEngine::new().unwrap();
    let mut request = part_request();
    settle(&mut owner, &mut request);
    click(&mut owner, &mut request, [1418., 45.]);
    press_key(&mut owner, &mut request, GuiKey::DownArrow);
    press_key(&mut owner, &mut request, GuiKey::UpArrow);
    press_key(&mut owner, &mut request, GuiKey::Space);
    assert!(
        !owner
            .ribbon
            .display
            .hidden_panels
            .contains(&(RibbonTab::Model, RibbonPanel::Parameters)),
        "expected last panel to be selected"
    );
    assert!(owner.popup_was_open);
}
fn press_key(owner: &mut GuiFrameEngine, request: &mut GuiFrameRequest, key: GuiKey) {
    for pressed in [true, false] {
        pointer_frame(owner, request, GuiInputEvent::Key { key, pressed });
    }
    settle(owner, request);
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
    let key = (RibbonTab::Model, RibbonPanel::Parameters);
    let index = crate::ribbon_panels::panels_for(key.0, &request.commands)
        .iter()
        .position(|panel| *panel == key.1)
        .unwrap();
    let point = [
        1280.,
        60. + style.window_padding[1] + index as f32 * row_height + 8.,
    ]
    .map(|value| value * backing);
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

#[test]
fn finish_panel_follows_the_sketch_context_on_each_part_tab() {
    let mut state = GuiCommandState {
        document: Some(7),
        document_type: Some(oblikovati_api::types::DOCUMENT_PART),
        authoring: GuiAuthoringContext::Sketch,
        ..Default::default()
    };
    for tab in [
        RibbonTab::Model,
        RibbonTab::Sketch,
        RibbonTab::Tools,
        RibbonTab::Manage,
        RibbonTab::View,
        RibbonTab::Annotate,
    ] {
        let panels = crate::ribbon_panels::panels_for(tab, &state);
        assert_eq!(
            panels
                .iter()
                .filter(|panel| **panel == RibbonPanel::Exit)
                .count(),
            1,
            "{tab:?}"
        );
        assert_eq!(panels.last(), Some(&RibbonPanel::Exit));
    }
    state.authoring = GuiAuthoringContext::Part;
    for tab in [
        RibbonTab::Model,
        RibbonTab::Tools,
        RibbonTab::Manage,
        RibbonTab::View,
        RibbonTab::Annotate,
    ] {
        assert!(
            !crate::ribbon_panels::panels_for(tab, &state).contains(&RibbonPanel::Exit),
            "{tab:?}"
        );
    }
}

#[test]
fn manage_insert_is_available_for_part_documents() {
    let part = crate::GuiCommandState {
        document_type: Some(oblikovati_api::types::DOCUMENT_PART),
        ..Default::default()
    };
    let panels = crate::ribbon_panels::panels_for(RibbonTab::Manage, &part);
    assert_eq!(panels, [RibbonPanel::Parameters, RibbonPanel::ManageInsert]);
    let empty = crate::GuiCommandState::default();
    assert!(
        !crate::ribbon_panels::panels_for(RibbonTab::Manage, &empty)
            .contains(&RibbonPanel::ManageInsert)
    );
}
