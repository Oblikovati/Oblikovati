// SPDX-License-Identifier: GPL-2.0-only
//! Show feature suppression differences between model states (#4119).
use super::*;

pub(super) fn model_state_indicator(
    ui: &Ui,
    browser: &GuiModelBrowser,
    feature: &GuiBrowserFeature,
) -> bool {
    let Some(states) = browser.model_states.as_ref() else {
        return false;
    };
    let Some(active) = states.active() else {
        return false;
    };
    let Ok(override_value) = states.suppression_override(active.id, feature.id.0) else {
        return false;
    };
    let authored = override_value.map_or(feature.suppressed, |value| !value);
    let values = states
        .list()
        .iter()
        .map(|member| {
            states
                .suppressed_for(member.id, feature.id.0, authored)
                .map(|suppressed| (&member.name, suppressed))
        })
        .collect::<Result<Vec<_>, _>>();
    let Ok(values) = values else {
        ui.same_line();
        ui.text_disabled("Model state error");
        return false;
    };
    if values
        .iter()
        .all(|(_, suppressed)| *suppressed == feature.suppressed)
    {
        return false;
    }
    ui.same_line();
    if ui.small_button("States") {
        ui.open_popup("##feature-model-states");
    }
    let indicator_hovered = ui.is_item_hovered();
    if let Some(_popup) = ui.begin_popup("##feature-model-states") {
        for (name, suppressed) in values {
            ui.text(format!(
                "{}: {}",
                name,
                if suppressed { "Suppressed" } else { "Computed" }
            ));
        }
    }
    indicator_hovered
}
