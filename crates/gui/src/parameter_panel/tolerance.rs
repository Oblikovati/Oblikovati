// SPDX-License-Identifier: GPL-2.0-only
use super::request_state::ParameterEditRequest;
use super::*;
use oblikovati_api::types::{
    MODEL_VALUE_LOWER, MODEL_VALUE_MEDIAN, MODEL_VALUE_NOMINAL, MODEL_VALUE_UPPER, ModelValueType,
    TOLERANCE_DEVIATION, Tolerance,
};
const MODEL_SELECTIONS: [(ModelValueType, &str, &str); 4] = [
    (MODEL_VALUE_NOMINAL, "nominal", "Nominal"),
    (MODEL_VALUE_UPPER, "upper", "Upper"),
    (MODEL_VALUE_LOWER, "lower", "Lower"),
    (MODEL_VALUE_MEDIAN, "median", "Median"),
];
#[derive(Clone, Debug)]
pub struct GuiParameterToleranceEdit {
    pub target: GuiParameterTextTarget,
    pub request: u64,
    pub tolerance: Tolerance,
    pub selection: ModelValueType,
}
impl PartialEq for GuiParameterToleranceEdit {
    fn eq(&self, other: &Self) -> bool {
        // #3614: signed zero is part of the stored recipe and the matching owner result.
        self.target == other.target
            && self.request == other.request
            && self.selection == other.selection
            && self.tolerance == other.tolerance
            && self.tolerance.upper.to_bits() == other.tolerance.upper.to_bits()
            && self.tolerance.lower.to_bits() == other.tolerance.lower.to_bits()
    }
}
impl GuiParameterToleranceEdit {
    /// Check a deviation edit before mutation; example: `edit.validate()?` on the document owner.
    pub fn validate(&self) -> Result<(), String> {
        if self.target.document == 0
            || self.target.field != GuiParameterTextField::Equation
            || self.request == 0
        {
            return Err(format!(
                "GUI tolerance edit {:?}, request {}: expected an active equation target and positive request",
                self.target, self.request
            ));
        }
        validate_deviation(&self.tolerance)?;
        selection_label(self.selection).map(|_| ())
    }
}
#[derive(Default)]
pub(super) struct ParameterToleranceEditor {
    target: Option<GuiParameterTextTarget>,
    upper: f64,
    lower: f64,
    selection: Option<ModelValueType>,
    request: ParameterEditRequest<GuiParameterToleranceEdit>,
}
impl ParameterToleranceEditor {
    pub(super) fn open(&mut self, document: u64, row: &GuiParameterRow) -> Result<(), String> {
        if matches!(row.detail.units.as_str(), "text" | "boolean") {
            return Err(format!(
                "GUI tolerance target {}, units {:?}: expected a numeric parameter",
                row.parameter, row.detail.units
            ));
        }
        let selection = selection_from_name(&row.detail.model_value_type)?;
        let (upper, lower) = row
            .detail
            .tolerance
            .as_ref()
            .map(|band| (band.upper, band.lower))
            .unwrap_or((0., 0.));
        validate_deviation(&deviation_band(upper, lower))?;
        self.target = Some(GuiParameterTextTarget {
            document,
            parameter: row.parameter,
            field: GuiParameterTextField::Equation,
        });
        self.upper = upper;
        self.lower = lower;
        self.selection = Some(selection);
        self.request.retire();
        Ok(())
    }
    pub(super) fn synchronize(&mut self, snapshot: Option<&GuiParameterSnapshot>) {
        let valid = self.target.is_some_and(|target| {
            snapshot.is_some_and(|view| {
                view.document == target.document
                    && view
                        .rows
                        .iter()
                        .any(|row| row.parameter == target.parameter)
            })
        });
        if !valid {
            self.close();
        }
    }
    fn close(&mut self) {
        self.target = None;
        self.selection = None;
        self.request.retire();
    }
    fn submit(&mut self) -> Result<Option<GuiParameterToleranceEdit>, String> {
        if self.request.pending.is_some() {
            return Ok(None);
        }
        let target = self.target.ok_or_else(|| {
            "GUI tolerance editor without target: expected an active numeric parameter".to_owned()
        })?;
        let selection = self.selection.ok_or_else(|| {
            "GUI tolerance selection None: expected a model value choice".to_owned()
        })?;
        let request = self.request.prepare()?;
        let edit = GuiParameterToleranceEdit {
            target,
            request,
            selection,
            tolerance: deviation_band(self.upper, self.lower),
        };
        edit.validate()?;
        self.request.publish(request, edit.clone())?;
        Ok(Some(edit))
    }
    pub(super) fn acknowledge(
        &mut self,
        edit: &GuiParameterToleranceEdit,
        result: Result<(), String>,
    ) -> GuiParameterAck {
        let outcome = self.request.acknowledge(edit, result);
        if outcome == GuiParameterAck::Applied {
            self.close();
        }
        outcome
    }
    pub(super) fn draw(&mut self, ui: &Ui, actions: &mut Vec<GuiParameterAction>) {
        if self.target.is_none() {
            return;
        }
        let mut open = true;
        ui.window("Tolerance")
            .size([360., 230.], Condition::FirstUseEver)
            .opened(&mut open)
            .build(|| self.draw_body(ui, actions));
        if !open {
            self.close();
        }
    }
    fn draw_body(&mut self, ui: &Ui, actions: &mut Vec<GuiParameterAction>) {
        let Some(target) = self.target else {
            return;
        };
        let _identity =
            parameter_controls_identity(ui, Some(target.document), Some(target.parameter));
        let _disabled = ui.begin_disabled(self.request.pending.is_some());
        ui.text("Deviation tolerance in document working units:");
        ui.input_scalar("Upper", &mut self.upper)
            .display_format("%.17g")
            .build();
        ui.input_scalar("Lower", &mut self.lower)
            .display_format("%.17g")
            .build();
        self.draw_selection(ui);
        if ui.button("OK") {
            self.enqueue(actions);
        }
        ui.same_line();
        if ui.button("Cancel") {
            self.close();
        }
        if let Some(issue) = &self.request.issue {
            ui.text_wrapped(issue.replace('\0', "\\0"));
        }
    }
    fn draw_selection(&mut self, ui: &Ui) {
        let Some(selection) = self.selection else {
            return;
        };
        let Ok(label) = selection_label(selection) else {
            return;
        };
        let Some(_combo) = ui.begin_combo("Model value", label) else {
            return;
        };
        for (value, _, label) in MODEL_SELECTIONS {
            if ui
                .selectable_config(label)
                .selected(selection == value)
                .build()
            {
                self.selection = Some(value);
            }
        }
    }
    fn enqueue(&mut self, actions: &mut Vec<GuiParameterAction>) {
        match self.submit() {
            Ok(Some(edit)) => actions.push(GuiParameterAction::Tolerance(edit)),
            Ok(None) => {}
            Err(issue) => self.request.issue = Some(issue),
        }
    }
}
fn deviation_band(upper: f64, lower: f64) -> Tolerance {
    Tolerance {
        type_: TOLERANCE_DEVIATION,
        upper,
        lower,
        ..Default::default()
    }
}
fn validate_deviation(band: &Tolerance) -> Result<(), String> {
    if band.type_ != TOLERANCE_DEVIATION
        || !band.upper.is_finite()
        || !band.lower.is_finite()
        || band.upper < band.lower
        || !band.hole_tolerance.is_empty()
        || !band.shaft_tolerance.is_empty()
    {
        return Err(format!(
            "GUI tolerance {band:?}: expected finite deviation bounds with upper >= lower and no fit annotations"
        ));
    }
    Ok(())
}
fn selection_label(selection: ModelValueType) -> Result<&'static str, String> {
    MODEL_SELECTIONS
        .iter()
        .find(|(value, _, _)| *value == selection)
        .map(|(_, _, label)| *label)
        .ok_or_else(|| {
            format!(
                "GUI tolerance selection {selection:?}: expected Nominal, Upper, Lower, or Median"
            )
        })
}
fn selection_from_name(name: &str) -> Result<ModelValueType, String> {
    if name.is_empty() {
        return Ok(MODEL_VALUE_NOMINAL);
    }
    MODEL_SELECTIONS
        .iter()
        .find(|(_, value, _)| *value == name)
        .map(|(value, _, _)| *value)
        .ok_or_else(|| {
            format!("GUI tolerance model value {name:?}: expected nominal, upper, lower, or median")
        })
}
#[cfg(test)]
#[path = "../../tests/parameters/tolerance.rs"]
mod tests;
