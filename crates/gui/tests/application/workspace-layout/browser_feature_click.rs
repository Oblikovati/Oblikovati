// SPDX-License-Identifier: GPL-2.0-only
//! Real pointer interaction for the feature row and its model-state control (#4128).
use super::*;
use crate::{GuiInputEvent, GuiPointerButton, input::apply_input};
use oblikovati_model::{feature::FeatureId, model_state::ModelStateMembers};

struct FeatureRowFixture {
    context: crate::test_context::TestContext,
    view: BrowserView,
    browser: GuiModelBrowser,
    feature: GuiBrowserFeature,
    button: [f32; 2],
    popup_open: bool,
}
impl FeatureRowFixture {
    fn new() -> Self {
        let mut context = crate::test_context::create();
        context.set_ini_filename(None);
        context.io_mut().display_size = [600., 400.];
        context.io_mut().delta_time = 1. / 60.;
        context.fonts().build_rgba32_texture();
        let mut states = ModelStateMembers::default();
        states.insert(2, "Other").unwrap();
        states.set_suppressed_for(2, 7, false, true).unwrap();
        Self {
            context,
            view: BrowserView::default(),
            browser: GuiModelBrowser {
                model_states: Some(states),
                ..Default::default()
            },
            feature: GuiBrowserFeature {
                id: FeatureId(7),
                name: "Extrusion1".into(),
                suppressed: false,
                extended: None,
                kind: "Extrusion",
            },
            button: [0.; 2],
            popup_open: false,
        }
    }
    fn frame(&mut self, events: &[GuiInputEvent]) -> Vec<GuiShellAction> {
        for event in events {
            apply_input(self.context.io_mut(), event, [1.; 2]);
        }
        let ui = self.context.frame();
        let mut actions = Vec::new();
        ui.window("Feature row probe")
            .position([8., 8.], Condition::Always)
            .size([400., 180.], Condition::Always)
            .build(|| {
                let state = GuiCommandState {
                    document: Some(12),
                    ..Default::default()
                };
                let _token = feature_row(
                    ui,
                    &state,
                    &self.browser,
                    &self.feature,
                    None,
                    &mut self.view,
                    &mut actions,
                );
                let lo = ui.item_rect_min();
                let hi = ui.item_rect_max();
                self.button = [(lo[0] + hi[0]) / 2., (lo[1] + hi[1]) / 2.];
                self.popup_open = ui.begin_popup("##feature-model-states").is_some();
            });
        self.context.render();
        actions
    }
    fn click(&mut self, point: [f32; 2]) -> Vec<GuiShellAction> {
        self.frame(&[GuiInputEvent::PointerMoved(point)]);
        let mut actions = Vec::new();
        for pressed in [true, false] {
            actions.extend(self.frame(&[GuiInputEvent::PointerButton {
                button: GuiPointerButton::Left,
                pressed,
            }]));
        }
        actions
    }
}
#[test]
fn states_button_opens_popup_and_feature_row_keeps_selection() {
    let mut probe = FeatureRowFixture::new();
    probe.frame(&[]);
    probe.frame(&[]);
    let button = probe.button;
    let actions = probe.click(button);
    assert!(probe.popup_open, "States click must open its member list");
    assert!(
        actions.is_empty(),
        "States click must not select the feature: {actions:?}"
    );
}

#[test]
fn feature_row_selects_in_empty_space_with_or_without_indicator() {
    for show_indicator in [false, true] {
        let mut probe = FeatureRowFixture::new();
        if !show_indicator {
            probe.browser.model_states = None;
        }
        probe.frame(&[]);
        probe.frame(&[]);
        let actions = probe.click([390., probe.button[1]]);
        assert_eq!(
            actions,
            [GuiShellAction::SelectBrowserFeature {
                document: 12,
                feature: FeatureId(7),
            }]
        );
    }
}
