// SPDX-License-Identifier: GPL-2.0-only
//! What narrows the recent list: the applied search, a date choice and file types.
use super::GuiRecentDocument;

/// The modification-date radio choices; `All` applies no limit.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(crate) enum DateFilter {
    #[default]
    All,
    Today,
    LastWeek,
    LastMonth,
}
impl DateFilter {
    pub(crate) const ALL: [Self; 4] = [Self::All, Self::Today, Self::LastWeek, Self::LastMonth];
    pub(crate) fn label(self) -> &'static str {
        match self {
            Self::All => "All",
            Self::Today => "Today",
            Self::LastWeek => "Last 7 days",
            Self::LastMonth => "Last 30 days",
        }
    }
    fn admits(self, age_days: Option<u64>) -> bool {
        let limit = match self {
            Self::All => return true,
            Self::Today => 0,
            Self::LastWeek => 6,
            Self::LastMonth => 29,
        };
        age_days.is_some_and(|age| age <= limit)
    }
}
/// The file-type checkboxes.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum TypeFilter {
    Part,
    Assembly,
    Drawing,
}
impl TypeFilter {
    pub(crate) fn label(self) -> &'static str {
        match self {
            Self::Part => "Part",
            Self::Assembly => "Assembly",
            Self::Drawing => "Drawing",
        }
    }
    fn matches(self, kind: crate::GuiTemplateKind) -> bool {
        use crate::GuiTemplateKind::{Assembly, Drawing, Part, SheetMetal};
        match self {
            Self::Part => matches!(kind, Part | SheetMetal),
            Self::Assembly => kind == Assembly,
            Self::Drawing => kind == Drawing,
        }
    }
}
const TYPES: [TypeFilter; 3] = [TypeFilter::Part, TypeFilter::Assembly, TypeFilter::Drawing];
/// One date choice and any file types; no type checked means every type.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub(crate) struct RecentFilter {
    pub date: DateFilter,
    types: Vec<TypeFilter>,
}
impl RecentFilter {
    /// A date choice counts once, each checked type once: the badge beside Filter.
    pub(crate) fn active_count(&self) -> usize {
        usize::from(self.date != DateFilter::All) + self.types.len()
    }
    pub(crate) fn type_rows(&self) -> [(TypeFilter, bool); 3] {
        TYPES.map(|choice| (choice, self.types.contains(&choice)))
    }
    pub(crate) fn set_type(&mut self, choice: TypeFilter, chosen: bool) {
        self.types.retain(|known| *known != choice);
        if chosen {
            self.types.push(choice);
        }
    }
    fn admits(&self, row: &GuiRecentDocument) -> bool {
        self.date.admits(row.modified_age_days)
            && (self.types.is_empty() || self.types.iter().any(|choice| choice.matches(row.kind)))
    }
}
/// Rows the list shows, keeping the list's own order; search ignores case.
pub(crate) fn matching_rows<'a>(
    rows: &'a [GuiRecentDocument],
    query: &str,
    filter: &RecentFilter,
) -> Vec<&'a GuiRecentDocument> {
    let wanted = query.to_lowercase();
    rows.iter()
        .filter(|row| filter.admits(row))
        .filter(|row| wanted.is_empty() || row.name.to_lowercase().contains(&wanted))
        .collect()
}

#[cfg(test)]
#[path = "../../../tests/application/home-recent-documents/filter.rs"]
mod tests;
