// SPDX-License-Identifier: GPL-2.0-only
//! Keyboard and mouse selection of the customized ribbon target.
use super::{RibbonCustomization, RibbonTab, category::move_cursor};
use imgui::{Key, Ui};

const TARGETS: [RibbonTab; 6] = [
    RibbonTab::Model,
    RibbonTab::Sketch,
    RibbonTab::Annotate,
    RibbonTab::Tools,
    RibbonTab::Manage,
    RibbonTab::View,
];

pub(super) fn target_index(target: RibbonTab) -> usize {
    TARGETS.iter().position(|tab| *tab == target).unwrap_or(0)
}

impl RibbonCustomization {
    pub(super) fn target_menu(&mut self, ui: &Ui) {
        self.target_cursor = move_cursor(ui, self.target_cursor, TARGETS.len());
        let enter = ui.is_key_pressed_no_repeat(Key::Enter);
        for (index, target) in TARGETS.into_iter().enumerate() {
            let clicked = ui
                .selectable_config(format!("Part | {}", target.label()))
                .selected(index == self.target_cursor)
                .build();
            if clicked || (enter && index == self.target_cursor) {
                self.draft.switch(target);
                ui.close_current_popup();
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn target_indices_follow_the_visible_order() {
        assert_eq!(target_index(RibbonTab::Tools), 3);
        assert_eq!(target_index(RibbonTab::Manage), 4);
        assert_eq!(target_index(RibbonTab::View), 5);
    }
}
