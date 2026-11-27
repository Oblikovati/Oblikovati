// SPDX-License-Identifier: GPL-2.0-only
//! Edit the ordered overflow commands in a marking-menu draft.
use imgui::Ui;
use oblikovati_api::wire::MarkingMenuView;

pub(super) fn editor(ui: &Ui, menu: &mut MarkingMenuView, available: &[(&str, &str)]) {
    ui.text("Overflow commands");
    ui.set_next_item_width(190.);
    if let Some(_combo) = ui.begin_combo("##add-marking-overflow", "Add command") {
        for &(id, name) in available {
            if ui.selectable(name) {
                menu.overflow.get_or_insert_default().push(id.into());
            }
        }
    }
    let Some(commands) = &mut menu.overflow else {
        return;
    };
    let rows_height = commands.len() as f32 * ui.frame_height_with_spacing();
    let padding = 2. * ui.clone_style().window_padding[1];
    let mut removed_last = false;
    ui.child_window("##marking-overflow-list")
        .size([255., (rows_height + padding).min(100.)])
        .border(true)
        .build(|| {
            let mut remove = None;
            for (index, command) in commands.iter_mut().enumerate() {
                if overflow_row(ui, index, command, available) {
                    remove = Some(index);
                }
            }
            if let Some(index) = remove {
                commands.remove(index);
                removed_last = commands.is_empty();
            }
        });
    if removed_last {
        menu.overflow = None;
    }
}
fn overflow_row(ui: &Ui, index: usize, command: &mut String, available: &[(&str, &str)]) -> bool {
    let preview = super::marking_menu::command_label(command, available);
    ui.set_next_item_width(160.);
    if let Some(_combo) = ui.begin_combo(format!("##marking-overflow-{index}"), preview) {
        for &(id, name) in available {
            if ui.selectable(name) {
                *command = id.into();
            }
        }
    }
    ui.same_line();
    ui.button(format!("Remove##marking-overflow-{index}"))
}
