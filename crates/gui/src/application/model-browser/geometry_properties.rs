// SPDX-License-Identifier: GPL-2.0-only
//! Geometry Properties: how a sketch's lines are drawn, set from its browser row.
use super::color_dialog::{ColorAnswer, ColorDialog, TITLE as COLOR_TITLE};
use crate::application::document_saving::folder_browser::center_dialog;
use crate::{GuiCommandState, GuiShellAction, GuiSketchLineStyle};
use imgui::{ComboBoxFlags, Ui};
use oblikovati_model::document_sketch::DocumentSketchId;

const TITLE: &str = "Geometry Properties";
const SIZE: [f32; 2] = [420., 260.];
const CONTROL_WIDTH: f32 = 200.;
const DEFAULT: &str = "Default";
const SELECT: &str = "Select";
const CUSTOM: &str = "Custom";
const NAMED_COLORS: [(&str, [u8; 3]); 8] = [
    ("Black", [0, 0, 0]),
    ("Red", [255, 0, 0]),
    ("Yellow", [255, 255, 0]),
    ("Green", [0, 255, 0]),
    ("Cyan", [0, 255, 255]),
    ("Blue", [0, 0, 255]),
    ("Magenta", [255, 0, 255]),
    ("White", [255, 255, 255]),
];
const LINE_TYPES: [&str; 7] = [
    DEFAULT,
    "Continuous",
    "Dashed",
    "Dotted",
    "Dash Dot",
    "Long Dash Dot",
    "Double Dash Dot",
];
const LINE_WEIGHTS: [f32; 7] = [0.13, 0.18, 0.25, 0.35, 0.5, 0.7, 1.];
/// How thick a sample stands for its weight, in pixels per millimeter.
const SAMPLE_PIXELS_PER_MILLIMETER: f32 = 4.;
const SAMPLE_WIDTH: f32 = 90.;
const SAMPLE_ROW: [f32; 2] = [0., 20.];
const SCALE_FORMAT: &str = "%.2f";

fn color_label(style: &GuiSketchLineStyle) -> String {
    let Some(color) = style.color else {
        return DEFAULT.to_owned();
    };
    NAMED_COLORS
        .iter()
        .find(|(_, named)| *named == color)
        .map_or_else(|| CUSTOM.to_owned(), |(name, _)| (*name).to_owned())
}
fn weight_label(weight: Option<f32>) -> String {
    weight.map_or_else(
        || DEFAULT.to_owned(),
        |millimeters| format!("{millimeters} mm"),
    )
}
#[derive(Debug)]
struct Editing {
    sketch: DocumentSketchId,
    applied: GuiSketchLineStyle,
    style: GuiSketchLineStyle,
    scale_shown: f32,
}
#[derive(Debug, Default)]
pub(crate) struct GeometryProperties {
    editing: Option<Editing>,
    opening: bool,
    color_opening: bool,
    colors: ColorDialog,
}
impl GeometryProperties {
    /// Open on a sketch's present style; example: from its row's Properties.
    pub(crate) fn request(&mut self, sketch: DocumentSketchId, style: GuiSketchLineStyle) {
        self.editing = Some(Editing {
            sketch,
            applied: style.clone(),
            scale_shown: style.scale.unwrap_or(1.),
            style,
        });
        self.opening = true;
    }
    /// A frame must follow so the dialog opens without waiting for an event.
    pub(crate) fn needs_frame(&self) -> bool {
        self.opening || self.color_opening
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if std::mem::take(&mut self.opening) {
            ui.open_popup(TITLE);
        }
        center_dialog(ui, SIZE);
        let Some(_dialog) = ui.begin_modal_popup(TITLE) else {
            return;
        };
        let Some(editing) = &mut self.editing else {
            ui.close_current_popup();
            return;
        };
        self.color_opening |= color_row(ui, editing);
        type_row(ui, editing);
        weight_row(ui, editing);
        scale_row(ui, editing);
        ui.separator();
        let closed = answer_row(ui, state, editing, actions);
        self.color_popup(ui);
        if closed {
            self.editing = None;
            ui.close_current_popup();
        }
    }
    fn color_popup(&mut self, ui: &Ui) {
        if std::mem::take(&mut self.color_opening) {
            self.colors.begin();
            ui.open_popup(COLOR_TITLE);
        }
        if let (Some(ColorAnswer::Chosen(color)), Some(editing)) =
            (self.colors.panel(ui), &mut self.editing)
        {
            editing.style.color = Some(color);
        }
    }
}
/// A list long enough for every row: these are short, and a scroll would hide some of them.
fn tall_list<'ui>(ui: &'ui Ui, label: &str, shown: &str) -> Option<imgui::ComboBoxToken<'ui>> {
    ui.begin_combo_with_flags(label, shown, ComboBoxFlags::HEIGHT_LARGEST)
}
/// A row of a list: choosing it closes the list and not the dialog under it.
fn list_row(ui: &Ui, label: &str, selected: bool) -> bool {
    ui.selectable_config(label).selected(selected).build()
}
/// True when Select asks for the color dialog.
fn color_row(ui: &Ui, editing: &mut Editing) -> bool {
    let mut selecting = false;
    ui.set_next_item_width(CONTROL_WIDTH);
    if let Some(_list) = tall_list(ui, "Line Color", &color_label(&editing.style)) {
        if list_row(ui, DEFAULT, editing.style.color.is_none()) {
            editing.style.color = None;
        }
        for (name, color) in NAMED_COLORS {
            if list_row(ui, name, editing.style.color == Some(color)) {
                editing.style.color = Some(color);
            }
        }
        selecting = list_row(ui, SELECT, false);
    }
    selecting
}
fn type_row(ui: &Ui, editing: &mut Editing) {
    ui.set_next_item_width(CONTROL_WIDTH);
    let shown = editing.style.line_type.clone();
    let shown = shown.as_deref().unwrap_or(DEFAULT);
    let Some(_list) = tall_list(ui, "Line Type", shown) else {
        return;
    };
    for name in LINE_TYPES {
        if list_row(ui, name, name == shown) {
            editing.style.line_type = (name != DEFAULT).then(|| name.to_owned());
        }
    }
}
fn weight_row(ui: &Ui, editing: &mut Editing) {
    ui.set_next_item_width(CONTROL_WIDTH);
    let shown = weight_label(editing.style.weight);
    let Some(_list) = tall_list(ui, "Line Weight", &shown) else {
        return;
    };
    if list_row(ui, DEFAULT, editing.style.weight.is_none()) {
        editing.style.weight = None;
    }
    for weight in LINE_WEIGHTS {
        if weight_sample_row(ui, weight, editing.style.weight == Some(weight)) {
            editing.style.weight = Some(weight);
        }
    }
}
fn weight_sample_row(ui: &Ui, weight: f32, selected: bool) -> bool {
    let chosen = ui
        .selectable_config(weight_label(Some(weight)))
        .selected(selected)
        .size(SAMPLE_ROW)
        .build();
    let [right, top] = [ui.item_rect_max()[0], ui.item_rect_min()[1]];
    let middle = top + SAMPLE_ROW[1] / 2.;
    let color = ui.style_color(imgui::StyleColor::Text);
    ui.get_window_draw_list()
        .add_line([right - SAMPLE_WIDTH, middle], [right, middle], color)
        .thickness((weight * SAMPLE_PIXELS_PER_MILLIMETER).max(1.))
        .build();
    chosen
}
/// The value stays beside its box, kept for when Scale is checked again.
fn scale_row(ui: &Ui, editing: &mut Editing) {
    let mut scaled = editing.style.scale.is_some();
    if ui.checkbox("Scale", &mut scaled) {
        editing.style.scale = scaled.then_some(editing.scale_shown);
    }
    ui.same_line();
    let _muted = ui.begin_disabled(!scaled);
    ui.set_next_item_width(CONTROL_WIDTH / 2.);
    let edited = ui
        .input_float("##line-scale", &mut editing.scale_shown)
        .display_format(SCALE_FORMAT)
        .build();
    if edited && scaled {
        editing.style.scale = Some(editing.scale_shown);
    }
}
/// True when the dialog closes.
fn answer_row(
    ui: &Ui,
    state: &GuiCommandState,
    editing: &mut Editing,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    let changed = editing.style != editing.applied;
    let apply = |editing: &mut Editing, actions: &mut Vec<GuiShellAction>| {
        actions.push(GuiShellAction::SetSketchLineStyle {
            document: state.document.unwrap_or(0),
            sketch: editing.sketch,
            style: editing.style.clone(),
        });
        editing.applied = editing.style.clone();
    };
    let ok = crate::widget_abi::button(ui, "OK", [80., 0.]);
    ui.same_line();
    let cancel = crate::widget_abi::button(ui, "Cancel", [80., 0.]);
    ui.same_line();
    let applied = {
        let _muted = ui.begin_disabled(!changed);
        crate::widget_abi::button(ui, "Apply", [80., 0.])
    };
    if changed && (ok || applied) {
        apply(editing, actions);
    }
    ok || cancel
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn labels_name_a_style_as_its_lists_do() {
        let mut style = GuiSketchLineStyle::default();
        assert_eq!(color_label(&style), DEFAULT);
        style.color = Some([255, 0, 0]);
        assert_eq!(color_label(&style), "Red");
        style.color = Some([1, 2, 3]);
        assert_eq!(color_label(&style), CUSTOM);
        assert_eq!(weight_label(None), DEFAULT);
        assert_eq!(weight_label(Some(0.25)), "0.25 mm");
    }
    #[test]
    fn a_request_starts_from_what_the_sketch_already_has() {
        let mut dialog = GeometryProperties::default();
        let style = GuiSketchLineStyle {
            scale: Some(2.),
            ..Default::default()
        };
        dialog.request(DocumentSketchId(1), style.clone());
        assert!(dialog.needs_frame());
        let editing = dialog.editing.as_ref().unwrap();
        assert_eq!(editing.scale_shown, 2.);
        assert_eq!(editing.applied, style);
    }
}
