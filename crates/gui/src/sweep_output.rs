// SPDX-License-Identifier: GPL-2.0-only
//! Sweep result controls and their current document names.
use super::*;

impl SweepDialog {
    pub(super) fn draw_operation(&mut self, ui: &Ui, state: &GuiCommandState) {
        let Some(browser) = &state.browser else {
            return;
        };
        let existing = browser
            .bodies
            .iter()
            .find(|body| Some(body.feature) == self.feature);
        let saved_join = self
            .feature
            .and_then(|id| browser.sweep_edits.get(&id))
            .is_some_and(|seed| seed.join);
        ui.radio_button("New Solid", &mut self.join, false);
        ui.same_line_with_pos(135.);
        let join_available = self.join
            || browser
                .bodies
                .iter()
                .filter(|body| Some(body.feature) != self.feature)
                .count()
                == 1
            || saved_join;
        let disabled = ui.begin_disabled(!join_available);
        ui.radio_button("Join", &mut self.join, true);
        drop(disabled);
        if !self.join {
            crate::theme::property_label(ui, "Body Name");
            if let Some(body) = existing.filter(|_| !saved_join) {
                ui.text(&body.name);
            } else {
                ui.text(format!("Solid{}", browser.bodies.len() + 1));
            }
        }
    }
}
