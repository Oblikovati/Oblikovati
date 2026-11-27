// SPDX-License-Identifier: GPL-2.0-only
//! How the recent list is ordered and laid out, and which rows are checked.
use super::GuiRecentDocument;
use std::cmp::Ordering;

/// What the list can sort by; a column header or the grid's Sort by menu chooses one.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum SortField {
    Name,
    Location,
    Modified,
    Opened,
    Size,
    /// A file's model area is not stored in it, so this choice keeps the list's order.
    Area,
}
impl SortField {
    pub(crate) const MENU: [Self; 6] = [
        Self::Name,
        Self::Location,
        Self::Modified,
        Self::Opened,
        Self::Size,
        Self::Area,
    ];
    pub(crate) fn label(self) -> &'static str {
        match self {
            Self::Name => "Name",
            Self::Location => "Location",
            Self::Modified => "Date Modified",
            Self::Opened => "Last Opened",
            Self::Size => "Size",
            Self::Area => "Area",
        }
    }
    fn compare(self, first: &GuiRecentDocument, second: &GuiRecentDocument) -> Ordering {
        match self {
            Self::Name => first.name.cmp(&second.name),
            Self::Location => first.location.cmp(&second.location),
            Self::Modified => first.modified.cmp(&second.modified),
            Self::Opened => first.opened.cmp(&second.opened),
            Self::Size => first.size_bytes.cmp(&second.size_bytes),
            Self::Area => Ordering::Equal,
        }
    }
}
/// The current order: newest opened first until a person chooses otherwise.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) struct RecentSort {
    pub field: SortField,
    pub descending: bool,
}
impl Default for RecentSort {
    fn default() -> Self {
        Self {
            field: SortField::Opened,
            descending: true,
        }
    }
}
impl RecentSort {
    /// A new field starts descending; its header chosen again reverses it.
    pub(crate) fn choose(&mut self, field: SortField) {
        if self.field == field {
            self.descending = !self.descending;
            return;
        }
        *self = Self {
            field,
            descending: true,
        };
    }
    /// Pinned rows lead; ties keep the list's own order, so a result never flickers.
    pub(crate) fn apply(self, rows: &mut [&GuiRecentDocument]) {
        rows.sort_by(|first, second| {
            let order = self.field.compare(first, second);
            let order = if self.descending {
                order.reverse()
            } else {
                order
            };
            second.pinned.cmp(&first.pinned).then(order)
        });
    }
    /// The header's direction mark; example: `▾` for descending.
    pub(crate) fn mark(self) -> &'static str {
        if self.descending { "▾" } else { "▴" }
    }
}
/// The columns after Name, in display order; each can be removed and added back.
#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) struct RecentColumns {
    pub shown: Vec<SortField>,
}
pub(crate) const ADJUSTABLE: [SortField; 4] = [
    SortField::Location,
    SortField::Modified,
    SortField::Opened,
    SortField::Size,
];
impl Default for RecentColumns {
    fn default() -> Self {
        Self {
            shown: ADJUSTABLE.to_vec(),
        }
    }
}
impl RecentColumns {
    /// A column added back joins the end of the adjustable ones.
    pub(crate) fn set_shown(&mut self, column: SortField, shown: bool) {
        self.shown.retain(|known| *known != column);
        if shown {
            self.shown.push(column);
        }
    }
    pub(crate) fn move_to(&mut self, from: usize, to: usize) {
        if from < self.shown.len() && to < self.shown.len() {
            let column = self.shown.remove(from);
            self.shown.insert(to, column);
        }
    }
}
pub(crate) fn column_width(column: SortField) -> f32 {
    match column {
        SortField::Location => 240.,
        SortField::Modified | SortField::Opened => 190.,
        _ => 90.,
    }
}
pub(crate) fn cell_text(row: &GuiRecentDocument, column: SortField) -> &str {
    match column {
        SortField::Name => &row.name,
        SortField::Location => &row.location,
        SortField::Modified => &row.modified_text,
        SortField::Opened => &row.opened_text,
        SortField::Size | SortField::Area => &row.size_text,
    }
}
/// Rows checked for a bulk action; a row the filters hide also loses its check.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub(crate) struct RecentChecks {
    pub paths: Vec<String>,
}
/// How the list header's box reads.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum HeaderCheck {
    None,
    Partial,
    All,
}
impl RecentChecks {
    pub(crate) fn contains(&self, path: &str) -> bool {
        self.paths.iter().any(|known| known == path)
    }
    pub(crate) fn set(&mut self, path: &str, checked: bool) {
        self.paths.retain(|known| known != path);
        if checked {
            self.paths.push(path.to_owned());
        }
    }
    pub(crate) fn keep_only(&mut self, rows: &[&GuiRecentDocument]) {
        self.paths
            .retain(|path| rows.iter().any(|row| row.path == *path));
    }
    pub(crate) fn header(&self, rows: &[&GuiRecentDocument]) -> HeaderCheck {
        let checked = rows.iter().filter(|row| self.contains(&row.path)).count();
        match checked {
            0 => HeaderCheck::None,
            count if count == rows.len() => HeaderCheck::All,
            _ => HeaderCheck::Partial,
        }
    }
    /// A partial or clear header selects everything shown; a full one clears it.
    pub(crate) fn toggle_header(&mut self, rows: &[&GuiRecentDocument]) {
        if self.header(rows) == HeaderCheck::All {
            self.paths.clear();
            return;
        }
        self.paths = rows.iter().map(|row| row.path.clone()).collect();
    }
}

#[cfg(test)]
#[path = "../../../tests/application/home-recent-documents/view.rs"]
mod tests;
