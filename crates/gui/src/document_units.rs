// SPDX-License-Identifier: GPL-2.0-only
//! Document unit controls send partial updates through the public units contract.
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Key, Ui};
use oblikovati_api::wire::{DocumentUnitsInfo, SetDocumentUnitsArgs};

#[derive(Default)]
pub(crate) struct GuiDocumentUnits {
    document: Option<u64>,
}
impl GuiDocumentUnits {
    pub(crate) fn request(&mut self, document: u64) {
        self.document = Some(document);
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
        home_active: bool,
    ) {
        if self.document != state.document || home_active {
            self.document = None;
        }
        let (Some(document), Some(units)) = (self.document, state.document_units.as_ref()) else {
            return;
        };
        crate::widget_abi::fit_dialog(ui, "Document Settings", [400., 170.], [390., 350.]);
        let mut open = true;
        let mut done = false;
        ui.window("Document Settings")
            .opened(&mut open)
            .collapsible(false)
            .build(|| {
                ui.text("Units");
                let disabled = ui.begin_disabled(state.transaction_open);
                unit_controls(ui, document, units, actions);
                precision_controls(ui, document, units, actions);
                drop(disabled);
                ui.text_disabled("Angle as DMS: unavailable");
                ui.separator();
                done = crate::widget_abi::button(ui, "Done", [0.; 2])
                    || ui.is_window_focused() && ui.is_key_pressed_no_repeat(Key::Escape);
            });
        if !open || done {
            self.document = None;
        }
    }
}
#[derive(Clone, Copy)]
enum UnitField {
    Length,
    Angle,
    Mass,
    Time,
    LengthFormat,
}
fn unit_controls(
    ui: &Ui,
    document: u64,
    units: &DocumentUnitsInfo,
    actions: &mut Vec<GuiShellAction>,
) {
    for (field, label, current, options) in [
        (
            UnitField::Length,
            "Length",
            units.length_unit.as_str(),
            &["µm", "mm", "cm", "m", "in", "ft"][..],
        ),
        (
            UnitField::Angle,
            "Angle",
            units.angle_unit.as_str(),
            &["deg", "rad"][..],
        ),
        (
            UnitField::Mass,
            "Mass",
            units.mass_unit.as_str(),
            &["kg", "g", "lb"][..],
        ),
        (
            UnitField::Time,
            "Time",
            units.time_unit.as_str(),
            &["s", "ms", "min", "hr"][..],
        ),
        (
            UnitField::LengthFormat,
            "Length format",
            units.length_display_format.as_str(),
            &["decimal", "fractional", "architectural"][..],
        ),
    ] {
        if let Some(selected) = select_unit(ui, label, current, options) {
            let mut request = SetDocumentUnitsArgs::default();
            let value = Some(Box::new(selected.to_string()));
            match field {
                UnitField::Length => request.length_unit = value,
                UnitField::Angle => request.angle_unit = value,
                UnitField::Mass => request.mass_unit = value,
                UnitField::Time => request.time_unit = value,
                UnitField::LengthFormat => request.length_display_format = value,
            }
            actions.push(GuiShellAction::SetDocumentUnits { document, request });
        }
    }
}
fn select_unit<'a>(ui: &Ui, label: &str, current: &str, options: &'a [&str]) -> Option<&'a str> {
    let _combo = ui.begin_combo(label, current)?;
    options.iter().copied().find(|name| {
        ui.selectable_config(name)
            .selected(*name == current)
            .build()
            && *name != current
    })
}
fn precision_controls(
    ui: &Ui,
    document: u64,
    units: &DocumentUnitsInfo,
    actions: &mut Vec<GuiShellAction>,
) {
    for (label, current) in [
        ("Length precision", units.length_display_precision),
        ("Angle precision", units.angle_display_precision),
    ] {
        let Ok(mut value) = i32::try_from(current) else {
            ui.text_disabled(format!("{label}: {current}"));
            continue;
        };
        if !ui.input_int(label, &mut value).build() || value < 0 || i64::from(value) == current {
            continue;
        }
        let mut request = SetDocumentUnitsArgs::default();
        if label == "Length precision" {
            request.length_display_precision = Some(Box::new(i64::from(value)));
        } else {
            request.angle_display_precision = Some(Box::new(i64::from(value)));
        }
        actions.push(GuiShellAction::SetDocumentUnits { document, request });
    }
}
