// SPDX-License-Identifier: GPL-2.0-only
//! Keyboard navigation of the available ribbon commands.
use super::{Command, ListFocus, RibbonCustomization};
use imgui::{Key, Ui};

#[derive(Clone, Copy)]
enum Navigation {
    First,
    Last,
    Previous,
    Next,
    Initial(char),
}

impl RibbonCustomization {
    pub(super) fn focus_add_from_source(&mut self, ui: &Ui) {
        let in_source = matches!(self.list_focus, Some(ListFocus::Source)) && self.source_focused;
        let io = ui.io();
        let plain_tab = !io.key_shift
            && !io.key_ctrl
            && !io.key_alt
            && !io.key_super
            && ui.is_key_pressed_no_repeat(Key::Tab);
        if in_source && plain_tab && self.draft.source.is_some() {
            ui.set_keyboard_focus_here();
            self.navigation_consumed = true;
        }
    }

    pub(super) fn navigate_source(&mut self, ui: &Ui) -> bool {
        if ui.io().key_ctrl || ui.io().key_alt || ui.io().key_super {
            return false;
        }
        let next = if let Some(down) = pressed_page(ui) {
            let commands = self.draft.available_commands();
            let current = commands
                .iter()
                .position(|command| Some(*command) == self.draft.source)
                .unwrap_or(0);
            let last = commands.len().saturating_sub(1);
            commands
                .get(page_row(current, last, self.visible_source.len(), down))
                .copied()
        } else if let Some(navigation) = pressed_navigation(ui) {
            navigate(
                &self.draft.available_commands(),
                self.draft.source,
                navigation,
            )
        } else {
            return false;
        };
        let Some(command) = next else {
            return true;
        };
        self.draft.reset_source_selection(command);
        self.scroll_source_to_selection = true;
        true
    }

    pub(super) fn navigate_target(&mut self, ui: &Ui) -> bool {
        if ui.io().key_ctrl || ui.io().key_alt || ui.io().key_super {
            return false;
        }
        let (down, count) = if let Some(down) = pressed_page(ui) {
            (down, self.visible_target_rows)
        } else if ui.is_key_pressed_no_repeat(Key::DownArrow) {
            (true, 1)
        } else if ui.is_key_pressed_no_repeat(Key::UpArrow) {
            (false, 1)
        } else {
            return false;
        };
        if let Some(last) = self.draft.rows().len().checked_sub(1) {
            let current = self.draft.selected.unwrap_or(0);
            let selected = page_row(current, last, count, down);
            self.draft.reset_target_selection(selected);
            self.scroll_target_to_selection = true;
        }
        true
    }
}

fn pressed_page(ui: &Ui) -> Option<bool> {
    if ui.is_key_pressed_no_repeat(Key::PageDown) {
        Some(true)
    } else {
        ui.is_key_pressed_no_repeat(Key::PageUp).then_some(false)
    }
}

fn page_row(current: usize, last: usize, visible: usize, down: bool) -> usize {
    let page = visible.max(1);
    if down {
        current.saturating_add(page).min(last)
    } else {
        current.saturating_sub(page)
    }
}

fn pressed_navigation(ui: &Ui) -> Option<Navigation> {
    let keys = [
        (Key::Home, Navigation::First),
        (Key::End, Navigation::Last),
        (Key::UpArrow, Navigation::Previous),
        (Key::DownArrow, Navigation::Next),
    ];
    keys.into_iter()
        .find_map(|(key, action)| ui.is_key_pressed_no_repeat(key).then_some(action))
        .or_else(|| pressed_initial(ui).map(Navigation::Initial))
}

fn pressed_initial(ui: &Ui) -> Option<char> {
    const LETTERS: [(Key, char); 26] = [
        (Key::A, 'a'),
        (Key::B, 'b'),
        (Key::C, 'c'),
        (Key::D, 'd'),
        (Key::E, 'e'),
        (Key::F, 'f'),
        (Key::G, 'g'),
        (Key::H, 'h'),
        (Key::I, 'i'),
        (Key::J, 'j'),
        (Key::K, 'k'),
        (Key::L, 'l'),
        (Key::M, 'm'),
        (Key::N, 'n'),
        (Key::O, 'o'),
        (Key::P, 'p'),
        (Key::Q, 'q'),
        (Key::R, 'r'),
        (Key::S, 's'),
        (Key::T, 't'),
        (Key::U, 'u'),
        (Key::V, 'v'),
        (Key::W, 'w'),
        (Key::X, 'x'),
        (Key::Y, 'y'),
        (Key::Z, 'z'),
    ];
    LETTERS
        .into_iter()
        .find_map(|(key, letter)| ui.is_key_pressed_no_repeat(key).then_some(letter))
}

fn navigate(commands: &[Command], current: Option<Command>, action: Navigation) -> Option<Command> {
    let index = current.and_then(|command| commands.iter().position(|item| *item == command));
    match action {
        Navigation::First => commands.first().copied(),
        Navigation::Last => commands.last().copied(),
        Navigation::Previous => commands.get(index.unwrap_or(0).saturating_sub(1)).copied(),
        Navigation::Next => commands
            .get(index.map_or(0, |index| (index + 1).min(commands.len().saturating_sub(1))))
            .copied(),
        Navigation::Initial(initial) => initial_match(commands, index, initial),
    }
}

fn initial_match(commands: &[Command], index: Option<usize>, initial: char) -> Option<Command> {
    let start = index.map_or(0, |index| index + 1);
    commands
        .iter()
        .cycle()
        .skip(start)
        .take(commands.len())
        .copied()
        .find(|command| {
            command
                .label()
                .chars()
                .next()
                .is_some_and(|first| first.eq_ignore_ascii_case(&initial))
        })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn target_pages_use_visible_rows_and_stop_at_either_end() {
        assert_eq!(page_row(2, 19, 7, true), 9);
        assert_eq!(page_row(9, 19, 7, false), 2);
        assert_eq!(page_row(16, 19, 7, true), 19);
        assert_eq!(page_row(3, 19, 7, false), 0);
        assert_eq!(page_row(0, 1, 2, true), 1);
        assert_eq!(page_row(1, 1, 2, false), 0);
    }

    #[test]
    fn navigation_clamps_and_initial_search_wraps() {
        let commands = [
            Command::Separator,
            Command::Sketches,
            Command::Annotations,
            Command::Measure,
        ];
        assert_eq!(
            navigate(&commands, Some(Command::Sketches), Navigation::First),
            Some(Command::Separator)
        );
        assert_eq!(
            navigate(&commands, Some(Command::Sketches), Navigation::Last),
            Some(Command::Measure)
        );
        assert_eq!(
            navigate(&commands, Some(Command::Separator), Navigation::Previous),
            Some(Command::Separator)
        );
        assert_eq!(
            navigate(&commands, Some(Command::Measure), Navigation::Next),
            Some(Command::Measure)
        );
        assert_eq!(
            navigate(&commands, Some(Command::Measure), Navigation::Initial('a')),
            None
        );
        assert_eq!(
            navigate(&commands, Some(Command::Measure), Navigation::Initial('2')),
            Some(Command::Sketches)
        );
        assert_eq!(navigate(&[], None, Navigation::Next), None);
    }
}
