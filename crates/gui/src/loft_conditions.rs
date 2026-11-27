// SPDX-License-Identifier: GPL-2.0-only
//! Edit the weighted takeoff of the first circular section.
use super::*;

#[derive(Default)]
pub(super) struct LoftConditions {
    angled: bool,
    angle: f64,
    weight: f64,
    submitted: Option<LoftTakeoffCondition>,
}

impl LoftConditions {
    pub(super) fn new(condition: Option<LoftTakeoffCondition>) -> Self {
        match condition {
            Some(condition) => Self {
                angled: true,
                angle: condition.angle_degrees,
                weight: condition.weight,
                submitted: Some(condition),
            },
            None => Self {
                angled: false,
                angle: 90.,
                weight: 0.,
                submitted: None,
            },
        }
    }

    pub(super) fn valid(&self) -> bool {
        !self.angled || self.condition().is_some()
    }

    fn condition(&self) -> Option<LoftTakeoffCondition> {
        LoftTakeoffCondition::new(self.angle, self.weight)
            .ok()
            .filter(|_| self.angled)
    }

    fn draw(
        &mut self,
        ui: &Ui,
        section: &GuiLoftSection,
        document: u64,
        actions: &mut Vec<GuiShellAction>,
    ) {
        ui.text(&section.name);
        ui.set_next_item_width(140.);
        let selected = if self.angled { "Angle" } else { "Free" };
        if let Some(_menu) = ui.begin_combo("Condition", selected) {
            for (angled, label) in [(false, "Free"), (true, "Angle")] {
                if crate::widget_abi::selectable(ui, label, self.angled == angled, false)
                    && self.angled != angled
                {
                    self.angled = angled;
                    self.weight = if angled { 1. } else { 0. };
                }
            }
        }
        {
            let _disabled = ui.begin_disabled(!self.angled);
            ui.set_next_item_width(140.);
            ui.input_scalar("Takeoff Angle", &mut self.angle)
                .display_format("%.2f deg")
                .build();
            ui.set_next_item_width(140.);
            ui.input_scalar("Weight", &mut self.weight)
                .display_format("%.3f ul")
                .build();
        }
        let condition = self.condition();
        if !self.valid() {
            ui.text_wrapped("Enter an angle from 0 to 180 degrees and a positive weight.");
        } else if self.submitted != condition {
            self.submitted = condition;
            actions.push(GuiShellAction::Loft(GuiLoftAction::SetTakeoff {
                document,
                condition,
            }));
        }
    }
}

impl LoftDialog {
    pub(super) fn conditions_tab(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if !self.editing
            || self.output != SketchLoftOutput::Surface
            || self.path_mode != GuiLoftPathMode::Rails
            || self.closed_loop
        {
            return;
        }
        if let Some(_conditions) = ui.tab_item("Conditions")
            && let Some((document, _)) = self.document
            && let Some(section) = state.loft_sections.first()
        {
            self.conditions.draw(ui, section, document, actions);
        }
    }
}
