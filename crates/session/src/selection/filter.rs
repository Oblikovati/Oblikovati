// SPDX-License-Identifier: GPL-2.0-only
use super::SelectionKind;
use std::collections::BTreeSet;

/// An explicit distinction between the default filter and the user's Deselect All (#1222).
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub enum SelectionFilter {
    #[default]
    All,
    None,
    Only(BTreeSet<SelectionKind>),
}
impl SelectionFilter {
    /// Restrict additions to these categories; an empty input retains the Go accept-all rule.
    /// Example: `SelectionFilter::only([SelectionKind::Edge])`.
    pub fn only(kinds: impl IntoIterator<Item = SelectionKind>) -> Self {
        let kinds: BTreeSet<SelectionKind> = kinds.into_iter().collect();
        if kinds.is_empty() {
            return Self::All;
        }
        Self::Only(kinds)
    }
    /// Check a category before a pick or addition; example: `filter.accepts(SelectionKind::Face)`.
    pub fn accepts(&self, kind: SelectionKind) -> bool {
        match self {
            Self::All => true,
            Self::None => false,
            Self::Only(kinds) => kinds.contains(&kind),
        }
    }
    /// Identify a tool or explicit restriction; example: `filter.is_restricted()`.
    pub fn is_restricted(&self) -> bool {
        !matches!(self, Self::All)
    }
}
