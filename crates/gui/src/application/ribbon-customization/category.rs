// SPDX-License-Identifier: GPL-2.0-only
//! Keyboard and mouse selection in the command category list.
use super::{Category, RibbonCustomization};
use imgui::{Key, Ui};

const CATEGORIES: [Category; 3] = [Category::All, Category::Macros, Category::Sketch];

pub(super) fn category_index(category: Category) -> usize {
    CATEGORIES
        .iter()
        .position(|item| *item == category)
        .unwrap_or(0)
}

impl RibbonCustomization {
    pub(super) fn category_menu(&mut self, ui: &Ui) {
        if ui.is_key_pressed_no_repeat(Key::DownArrow) {
            self.category_cursor = (self.category_cursor + 1).min(CATEGORIES.len() - 1);
        }
        if ui.is_key_pressed_no_repeat(Key::UpArrow) {
            self.category_cursor = self.category_cursor.saturating_sub(1);
        }
        let enter = ui.is_key_pressed_no_repeat(Key::Enter);
        for (index, category) in CATEGORIES.into_iter().enumerate() {
            let clicked = ui
                .selectable_config(category.label())
                .selected(index == self.category_cursor)
                .build();
            if clicked || (enter && index == self.category_cursor) {
                self.draft.set_category(category);
                ui.close_current_popup();
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn category_indices_match_the_visible_order() {
        assert_eq!(category_index(Category::All), 0);
        assert_eq!(category_index(Category::Macros), 1);
        assert_eq!(category_index(Category::Sketch), 2);
    }
}
