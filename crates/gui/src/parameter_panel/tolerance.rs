// SPDX-License-Identifier: GPL-2.0-only
use super::request_state::ParameterEditRequest;
use super::tolerance_format as format;
use super::*;
use oblikovati_api::types::{
    MODEL_VALUE_LOWER, MODEL_VALUE_MEDIAN, MODEL_VALUE_NOMINAL, MODEL_VALUE_UPPER, ModelValueType,
    TOLERANCE_DEFAULT, TOLERANCE_SYMMETRIC, Tolerance, ToleranceType,
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
    pub precision: Option<i64>,
}
impl PartialEq for GuiParameterToleranceEdit {
    fn eq(&self, other: &Self) -> bool {
        // #3614: signed zero is part of the stored recipe and the matching owner result.
        self.target == other.target
            && self.request == other.request
            && self.selection == other.selection
            && self.precision == other.precision
            && self.tolerance == other.tolerance
            && self.tolerance.upper.to_bits() == other.tolerance.upper.to_bits()
            && self.tolerance.lower.to_bits() == other.tolerance.lower.to_bits()
    }
}
impl GuiParameterToleranceEdit {
    /// Check an engineering tolerance edit before mutation; example: `edit.validate()?` on the document owner.
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
        format::validate_band(&self.tolerance)?;
        if let Some(precision) = self.precision
            && !(0..=8).contains(&precision)
        {
            return Err(format!(
                "GUI tolerance precision {precision}: expected 0..=8"
            ));
        }
        selection_label(self.selection).map(|_| ())
    }
}
#[derive(Default)]
pub(super) struct ParameterToleranceEditor {
    target: Option<GuiParameterTextTarget>,
    inspecting: bool,
    dock_state: crate::dockable_lifecycle::DockableState,
    upper: f64,
    lower: f64,
    kind: ToleranceType,
    hole: String,
    shaft: String,
    precision: i64,
    original_precision: i64,
    engineering: Option<crate::GuiParameterEngineering>,
    selection: Option<ModelValueType>,
    request: ParameterEditRequest<GuiParameterToleranceEdit>,
}
impl DockableWindow for ParameterToleranceEditor {}
impl ParameterToleranceEditor {
    pub(super) fn is_open(&self) -> bool {
        self.target.is_some()
    }
    pub(super) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Tolerance", &self.dock_state)
    }
    pub(super) fn open(&mut self, document: u64, row: &GuiParameterRow) -> Result<(), String> {
        if matches!(row.detail.units.as_str(), "text" | "boolean") {
            return Err(format!(
                "GUI tolerance target {}, units {:?}: expected a numeric parameter",
                row.parameter, row.detail.units
            ));
        }
        let selection = selection_from_name(&row.detail.model_value_type)?;
        let source = row.detail.tolerance.as_deref();
        let kind = format::seed_kind(source.map_or("", |band| band.type_.as_str()))?;
        let (upper, lower) = row
            .detail
            .tolerance
            .as_ref()
            .map(|band| (band.upper, band.lower))
            .unwrap_or((0., 0.));
        if !upper.is_finite() || !lower.is_finite() || upper < lower {
            return Err(format!(
                "GUI tolerance bounds {upper}/{lower}: expected finite ordered values"
            ));
        }
        self.inspecting = false;
        self.target = Some(GuiParameterTextTarget {
            document,
            parameter: row.parameter,
            field: GuiParameterTextField::Equation,
        });
        self.dock_state.set_open(true);
        self.upper = upper;
        self.lower = lower;
        self.kind = kind;
        self.hole = source.map_or_else(String::new, |band| band.hole_tolerance.clone());
        self.shaft = source.map_or_else(String::new, |band| band.shaft_tolerance.clone());
        self.precision = row.detail.precision;
        self.original_precision = row.detail.precision;
        self.engineering = row.engineering.clone();
        self.selection = Some(selection);
        self.request.retire();
        Ok(())
    }
    pub(super) fn synchronize(&mut self, snapshot: Option<&GuiParameterSnapshot>) {
        if self.inspecting {
            return;
        }
        let row = self
            .target
            .zip(snapshot)
            .filter(|(target, view)| target.document == view.document)
            .and_then(|(target, view)| {
                view.rows
                    .iter()
                    .find(|row| row.parameter == target.parameter)
            });
        match row {
            Some(row) => self.engineering.clone_from(&row.engineering),
            None => self.close(),
        }
    }
    fn close(&mut self) {
        self.target = None;
        self.inspecting = false;
        self.dock_state.set_open(false);
        self.selection = None;
        self.engineering = None;
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
            tolerance: self.draft_band()?,
            precision: (self.precision != self.original_precision).then_some(self.precision),
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
        if self.target.is_none() && !self.inspecting {
            return;
        }
        let mut open = true;
        crate::widget_abi::window_size(ui, [440., 320.], Condition::FirstUseEver);
        Self::dockable_window(ui, "Tolerance", &mut open).build(|| self.draw_body(ui, actions));
        if !open {
            self.close();
        }
    }
    fn draw_body(&mut self, ui: &Ui, actions: &mut Vec<GuiParameterAction>) {
        let _identity = parameter_controls_identity(
            ui,
            self.target.map(|target| target.document),
            self.target.map(|target| target.parameter),
        );
        let _disabled = ui.begin_disabled(self.request.pending.is_some());
        self.draw_fields(ui);
        if !self.inspecting {
            if crate::widget_abi::button(ui, "OK", [0., 0.]) {
                self.enqueue(actions);
            }
            ui.same_line();
        }
        if crate::widget_abi::button(ui, "Cancel", [0., 0.]) {
            self.close();
        }
        if let Some(issue) = &self.request.issue {
            ui.text_wrapped(issue.replace('\0', "\\0"));
        }
    }
    /// Inspect the engineering values for a draft distance with no parameter target.
    pub(super) fn open_inspection(&mut self, engineering: crate::GuiParameterEngineering) {
        self.close();
        self.inspecting = true;
        self.dock_state.set_open(true);
        self.selection = Some(MODEL_VALUE_NOMINAL);
        self.kind = TOLERANCE_DEFAULT;
        self.engineering = Some(engineering);
        self.precision = 0;
        self.upper = 0.;
        self.lower = 0.;
        self.hole.clear();
        self.shaft.clear();
    }

    pub(super) fn close_inspection(&mut self) {
        if self.inspecting {
            self.close();
        }
    }

    fn draw_fields(&mut self, ui: &Ui) {
        let model = self.current_model_label();
        ui.text(format!("Model Value: {model}"));
        self.draw_bounds(ui);
        self.draw_type(ui);
        ui.input_scalar("Precision", &mut self.precision).build();
        if ui.is_item_hovered() {
            ui.tooltip_text("Use 0 for the document precision");
        }
        self.draw_fits(ui);
        self.draw_selection(ui);
    }

    fn draw_bounds(&mut self, ui: &Ui) {
        let _bounds = ui.begin_disabled(!format::bounds_editable(self.kind));
        ui.input_scalar("Upper", &mut self.upper)
            .display_format("%.17g")
            .build();
        let _symmetric = ui.begin_disabled(self.kind == TOLERANCE_SYMMETRIC);
        if self.kind == TOLERANCE_SYMMETRIC {
            ui.input_scalar("Lower", &mut -self.upper)
                .display_format("%.17g")
                .build();
        } else {
            ui.input_scalar("Lower", &mut self.lower)
                .display_format("%.17g")
                .build();
        }
    }
    fn draw_type(&mut self, ui: &Ui) {
        let Some(label) = format::kind_label(self.kind) else {
            return;
        };
        let Some(_list) =
            ui.begin_combo_with_flags("Type", label, imgui::ComboBoxFlags::HEIGHT_LARGEST)
        else {
            return;
        };
        for (kind, label) in format::FORMATS {
            if crate::widget_abi::selectable(ui, label, self.kind == kind, false) {
                self.change_kind(kind);
            }
        }
    }
    fn change_kind(&mut self, kind: ToleranceType) {
        self.kind = kind;
        if kind == TOLERANCE_SYMMETRIC {
            self.upper = self.upper.abs();
        }
        self.request.issue = None;
    }
    fn draw_fits(&mut self, ui: &Ui) {
        let _disabled = ui.begin_disabled(!format::is_fit(self.kind));
        ui.input_text("Hole", &mut self.hole).build();
        ui.input_text("Shaft", &mut self.shaft).build();
    }
    fn draft_band(&self) -> Result<Tolerance, String> {
        if format::is_fit(self.kind) {
            return format::fit_band(
                self.engineering.as_ref(),
                self.kind,
                &self.hole,
                &self.shaft,
            );
        }
        if self.kind == TOLERANCE_DEFAULT {
            return Ok(Tolerance::default());
        }
        let (upper, lower) = match self.kind {
            TOLERANCE_SYMMETRIC => (self.upper, -self.upper),
            kind if format::bounds_editable(kind) => (self.upper, self.lower),
            _ => (0., 0.),
        };
        Ok(Tolerance {
            type_: self.kind,
            upper,
            lower,
            ..Default::default()
        })
    }
    fn current_model_label(&self) -> String {
        self.engineering
            .as_ref()
            .zip(self.selection)
            .and_then(|(seed, selection)| {
                format::model_label(seed, &self.draft_band().ok()?, selection, self.precision)
            })
            .unwrap_or_else(|| "—".into())
    }
    fn draw_selection(&mut self, ui: &Ui) {
        let Some(selection) = self.selection else {
            return;
        };
        let Ok(label) = selection_label(selection) else {
            return;
        };
        let Some(_combo) = ui.begin_combo("Evaluated Size", label) else {
            return;
        };
        for (value, _, label) in MODEL_SELECTIONS {
            if crate::widget_abi::selectable(ui, label, selection == value, false) {
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
