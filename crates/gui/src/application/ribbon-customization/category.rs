// SPDX-License-Identifier: GPL-2.0-only
//! Keyboard and mouse selection in the command category list.
use super::{Command, RibbonCustomization};
use imgui::{Key, Ui};

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(super) enum Category {
    #[default]
    All,
    Macros,
    Sketch,
    View,
    ExitSketch,
}
impl Category {
    pub fn label(self) -> &'static str {
        match self {
            Self::All => "All Commands",
            Self::Macros => "Macros",
            Self::Sketch => "Sketch Tab",
            Self::View => "View Tab",
            Self::ExitSketch => "Exit 2D Sketch Tab",
        }
    }
    pub(super) fn contains(self, command: Command) -> bool {
        match self {
            Self::All => true,
            Self::Macros => command == Command::Separator,
            Self::Sketch => matches!(
                command,
                Command::Separator | Command::Tool(_) | Command::Constraint(_) | Command::Dimension
            ),
            Self::View => matches!(
                command,
                Command::Separator | Command::Sketches | Command::Zoom(_)
            ),
            Self::ExitSketch => matches!(
                command,
                Command::Separator | Command::CancelSketch | Command::FinishSketch
            ),
        }
    }
}

const CATEGORIES: [Category; 5] = [
    Category::All,
    Category::Macros,
    Category::Sketch,
    Category::View,
    Category::ExitSketch,
];

pub(super) fn category_index(category: Category) -> usize {
    CATEGORIES
        .iter()
        .position(|item| *item == category)
        .unwrap_or(0)
}

impl RibbonCustomization {
    pub(super) fn category_menu(&mut self, ui: &Ui) {
        self.category_cursor = move_cursor(ui, self.category_cursor, CATEGORIES.len());
        let enter = ui.is_key_pressed_no_repeat(Key::Enter);
        for (index, category) in CATEGORIES.into_iter().enumerate() {
            let clicked = ui
                .selectable_config(category.label())
                .selected(index == self.category_cursor)
                .build();
            if clicked || (enter && index == self.category_cursor) {
                self.draft.set_category(category);
                self.scroll_source_to_selection = self.draft.source.is_some();
                ui.close_current_popup();
            }
        }
    }
}

pub(super) fn move_cursor(ui: &Ui, current: usize, count: usize) -> usize {
    if ui.is_key_pressed_no_repeat(Key::DownArrow) {
        return (current + 1).min(count.saturating_sub(1));
    }
    if ui.is_key_pressed_no_repeat(Key::UpArrow) {
        return current.saturating_sub(1);
    }
    current
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn category_indices_match_the_visible_order() {
        assert_eq!(category_index(Category::All), 0);
        assert_eq!(category_index(Category::Macros), 1);
        assert_eq!(category_index(Category::Sketch), 2);
        assert_eq!(category_index(Category::View), 3);
    }
}
