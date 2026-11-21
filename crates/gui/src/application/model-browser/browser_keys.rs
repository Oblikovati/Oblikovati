// SPDX-License-Identifier: GPL-2.0-only
//! The arrow keys in the browser: they walk the Origin branch and its rows.
use crate::browser::ORIGIN_ROWS;
use imgui::{Key, Ui};

/// Where the arrow keys stand in the Origin branch.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(crate) struct OriginCursor {
    /// The row chosen under Origin, by its place.
    pub row: Option<usize>,
    /// Origin's own row is chosen.
    pub folder: bool,
    /// The branch shows its rows.
    pub open: bool,
}
/// What a press asks of the branch itself, beside where the cursor goes.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(crate) struct OriginMove {
    pub cursor: OriginCursor,
    pub set_open: Option<bool>,
}
/// One press. Down and Up walk the rows; Left goes to the folder, then folds it; Right opens the
/// folder, then enters it. Example: `moved(cursor, Key::DownArrow)`.
pub(crate) fn moved(cursor: OriginCursor, key: Key) -> OriginMove {
    let last = ORIGIN_ROWS.len() - 1;
    let chosen = |row| OriginCursor {
        row: Some(row),
        folder: false,
        ..cursor
    };
    let folder = OriginCursor {
        row: None,
        folder: true,
        ..cursor
    };
    let stay = |cursor| OriginMove {
        cursor,
        set_open: None,
    };
    match (key, cursor.row, cursor.folder, cursor.open) {
        (Key::DownArrow, Some(row), ..) => stay(chosen((row + 1).min(last))),
        (Key::UpArrow, Some(0), ..) | (Key::LeftArrow, Some(_), ..) => stay(folder),
        (Key::UpArrow, Some(row), ..) => stay(chosen(row - 1)),
        (Key::LeftArrow, None, true, true) => OriginMove {
            cursor: OriginCursor {
                open: false,
                ..folder
            },
            set_open: Some(false),
        },
        (Key::RightArrow, None, true, false) => OriginMove {
            cursor: OriginCursor {
                open: true,
                ..folder
            },
            set_open: Some(true),
        },
        (Key::RightArrow, None, true, true) | (Key::DownArrow, None, true, true) => stay(chosen(0)),
        _ => stay(cursor),
    }
}
/// The press this frame, while the browser has the keyboard and a row of Origin is chosen.
pub(crate) fn pressed_arrow(ui: &Ui) -> Option<Key> {
    let focus = imgui::WindowFocusedFlags::ROOT_AND_CHILD_WINDOWS;
    if !ui.is_window_focused_with_flags(focus) || crate::widget_abi::any_popup_open(ui) {
        return None;
    }
    [
        Key::DownArrow,
        Key::UpArrow,
        Key::LeftArrow,
        Key::RightArrow,
    ]
    .into_iter()
    .find(|key| ui.is_key_pressed(*key))
}
#[cfg(test)]
mod tests {
    use super::*;
    fn on_row(row: usize) -> OriginCursor {
        OriginCursor {
            row: Some(row),
            open: true,
            ..Default::default()
        }
    }
    #[test]
    fn down_and_up_walk_the_rows_and_stop_at_the_last() {
        assert_eq!(moved(on_row(0), Key::DownArrow).cursor.row, Some(1));
        assert_eq!(moved(on_row(1), Key::UpArrow).cursor.row, Some(0));
        let last = ORIGIN_ROWS.len() - 1;
        assert_eq!(moved(on_row(last), Key::DownArrow).cursor.row, Some(last));
    }
    #[test]
    fn left_goes_to_the_folder_then_folds_it_and_right_undoes_each() {
        let at_folder = moved(on_row(2), Key::LeftArrow);
        assert!(at_folder.cursor.folder && at_folder.cursor.row.is_none());
        let folded = moved(at_folder.cursor, Key::LeftArrow);
        assert_eq!(folded.set_open, Some(false));
        let opened = moved(folded.cursor, Key::RightArrow);
        assert_eq!(opened.set_open, Some(true));
        assert_eq!(moved(opened.cursor, Key::RightArrow).cursor.row, Some(0));
    }
}
