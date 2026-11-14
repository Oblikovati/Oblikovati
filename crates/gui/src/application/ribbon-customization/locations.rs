// SPDX-License-Identifier: GPL-2.0-only
//! Locations of commands available from the standard ribbon.
use super::Command;
use imgui::Ui;

fn ribbon_locations(command: Command) -> &'static [&'static str] {
    match command {
        Command::Sketches => &["Part | View | Visibility", "Assembly | View | Visibility"],
        _ => &[],
    }
}

pub(super) fn show(ui: &Ui, command: Command) {
    let locations = ribbon_locations(command);
    if locations.is_empty() || !crate::widget_abi::item_mouse_hovered(ui) {
        return;
    }
    ui.tooltip(|| {
        ui.text(command.label());
        for location in locations {
            ui.text(location);
        }
    });
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn sketch_visibility_lists_both_view_ribbons() {
        assert_eq!(
            ribbon_locations(Command::Sketches),
            ["Part | View | Visibility", "Assembly | View | Visibility"]
        );
        assert!(ribbon_locations(Command::Separator).is_empty());
    }
}
