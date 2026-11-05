// SPDX-License-Identifier: GPL-2.0-only
use imgui::Ui;

pub(crate) fn appearance_choice(
    ui: &Ui,
    label: &str,
    active: &str,
    names: &[String],
) -> Option<String> {
    ui.set_next_item_width(160.);
    let _combo = ui.begin_combo(label, active)?;
    let mut selected = None;
    for name in names {
        if ui.menu_item_config(name).selected(name == active).build() {
            selected = Some(name.clone());
        }
    }
    selected
}
