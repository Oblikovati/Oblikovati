// SPDX-License-Identifier: GPL-2.0-only
//! Clear the selected body's own appearance override through the host.
use super::*;

pub(super) fn action(state: &GuiCommandState) -> Option<GuiShellAction> {
    Some(GuiShellAction::OpenClearBodyAppearance {
        document: state.document?,
        key: state.selected_body_key.clone()?,
    })
}

pub(super) fn button(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let action = action(state);
    let enabled = action.as_ref().is_some_and(|value| value.enabled(state));
    let disabled = ui.begin_disabled(!enabled);
    let clicked = ui.button("Clear");
    drop(disabled);
    if !enabled && ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
        ui.tooltip_text("Select a body with its own appearance override to clear it.");
    }
    if clicked && let Some(action) = action {
        actions.push(action);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_model::feature::FeatureId;

    fn selected_body() -> GuiCommandState {
        GuiCommandState {
            document: Some(7),
            authoring: crate::GuiAuthoringContext::Part,
            selected_body_key: Some("body-key".into()),
            browser: Some(crate::GuiModelBrowser {
                bodies: vec![crate::GuiBrowserBody {
                    name: "Body".into(),
                    feature: FeatureId(1),
                    key: "body-key".into(),
                    index: 0,
                    visible: true,
                    faces: 6,
                    appearance: "black".into(),
                    physical: None,
                }],
                ..Default::default()
            }),
            ..Default::default()
        }
    }
    #[test]
    fn clear_requires_the_exact_current_body_and_its_own_override() {
        let mut state = selected_body();
        let action = action(&state).unwrap();
        assert!(action.enabled(&state));
        state.selected_body_key = Some("another".into());
        assert!(!action.enabled(&state));
        state.selected_body_key = Some("body-key".into());
        state.browser.as_mut().unwrap().bodies[0].appearance.clear();
        assert!(!action.enabled(&state));
        state.browser.as_mut().unwrap().bodies[0].appearance = "black".into();
        state.transaction_open = true;
        assert!(!action.enabled(&state));
    }
}
