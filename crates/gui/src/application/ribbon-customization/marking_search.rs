// SPDX-License-Identifier: GPL-2.0-only
//! Find commands in the selected profile and stage their assignments.
use super::{LOCATIONS, replace_slot};
use imgui::Ui;
use oblikovati_api::wire::MarkingMenuView;

pub(super) fn editor(
    ui: &Ui,
    query: &mut String,
    menu: &mut MarkingMenuView,
    commands: &[(&str, &str)],
) -> bool {
    ui.text("Search command");
    ui.set_next_item_width(190.);
    ui.input_text("##marking-search", query)
        .hint("Type keyword")
        .build();
    ui.same_line();
    if ui.small_button("Clear##marking-search-clear") {
        query.clear();
    }
    ui.text("All Commands");
    let mut popup_open = false;
    ui.child_window("##marking-command-results")
        .size([255., 100.])
        .border(true)
        .build(|| {
            let mut results = matching(commands, query).peekable();
            if results.peek().is_none() {
                ui.text_disabled("No matching commands");
            }
            for (id, name) in results {
                popup_open |= command_row(ui, menu, id, name);
            }
        });
    popup_open
}

fn matching<'a>(
    commands: &'a [(&'a str, &'a str)],
    query: &str,
) -> impl Iterator<Item = (&'a str, &'a str)> + use<'a> {
    let keyword = query.trim().to_lowercase();
    commands
        .iter()
        .copied()
        .filter(move |(_, name)| name.to_lowercase().contains(&keyword))
}

fn command_row(ui: &Ui, menu: &mut MarkingMenuView, id: &str, name: &str) -> bool {
    let _command = ui.push_id(id);
    if ui.selectable(name) {
        ui.open_popup("##marking-command-location");
    }
    let Some(_popup) = ui.begin_popup("##marking-command-location") else {
        return false;
    };
    for (slot, label) in LOCATIONS.into_iter().enumerate() {
        if ui.menu_item(format!("Assign to {label}")) {
            replace_slot(menu, slot as u8, id);
        }
    }
    if ui.menu_item("Add to overflow") {
        menu.overflow.get_or_insert_default().push(id.into());
    }
    true
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn matching_uses_case_insensitive_names_and_clearing_restores_the_catalog() {
        let commands = [
            ("arc.center", "Center Point Arc"),
            ("arc.three", "Three Point Arc"),
            ("line", "Line"),
        ];
        assert_eq!(
            matching(&commands, "  POINT arc ").collect::<Vec<_>>(),
            commands[..2]
        );
        assert_eq!(matching(&commands, "").collect::<Vec<_>>(), commands);
        assert!(matching(&commands, "unknown").next().is_none());
    }
}
