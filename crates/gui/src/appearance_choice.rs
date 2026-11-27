// SPDX-License-Identifier: GPL-2.0-only
use imgui::Ui;

pub(crate) fn appearance_choice(
    ui: &Ui,
    label: &str,
    active: &str,
    names: &[String],
) -> Option<String> {
    ui.align_text_to_frame_padding();
    ui.text(label);
    let label_width = ui.calc_text_size("Colour scheme")[0] + 12.;
    ui.same_line_with_spacing(0., label_width - ui.calc_text_size(label)[0]);
    let _id = ui.push_id(label);
    ui.set_next_item_width(160.);
    let _combo = ui.begin_combo("##appearance-choice", active)?;
    let mut selected = None;
    for name in names {
        if ui.menu_item_config(name).selected(name == active).build() {
            selected = Some(name.clone());
        }
    }
    selected
}
