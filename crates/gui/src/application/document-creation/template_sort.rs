// SPDX-License-Identifier: GPL-2.0-only
//! One ordering for grouped and ungrouped template galleries.
use super::{GuiCommandState, GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits};

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum SortOrder {
    Increasing,
    Decreasing,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum TemplateSort {
    Name(SortOrder),
    Size(SortOrder),
}
impl Default for TemplateSort {
    fn default() -> Self {
        Self::Name(SortOrder::Increasing)
    }
}
impl TemplateSort {
    pub(super) fn toggle_name(self) -> Self {
        Self::Name(match self {
            Self::Name(SortOrder::Increasing) => SortOrder::Decreasing,
            _ => SortOrder::Increasing,
        })
    }
    pub(super) fn toggle_size(self) -> Self {
        Self::Size(match self {
            Self::Size(SortOrder::Increasing) => SortOrder::Decreasing,
            _ => SortOrder::Increasing,
        })
    }
    pub(super) fn name_heading(self) -> &'static str {
        match self {
            Self::Name(SortOrder::Increasing) => "Name ^##template-name",
            Self::Name(SortOrder::Decreasing) => "Name v##template-name",
            Self::Size(_) => "Name##template-name",
        }
    }
}
pub(super) fn size_heading(sort: TemplateSort) -> &'static str {
    match sort {
        TemplateSort::Size(SortOrder::Increasing) => "Size ^##template-size",
        TemplateSort::Size(SortOrder::Decreasing) => "Size v##template-size",
        TemplateSort::Name(_) => "Size##template-size",
    }
}
pub(super) fn template_size(state: &GuiCommandState, kind: GuiTemplateKind) -> Option<u64> {
    state
        .template_sizes
        .iter()
        .find(|(sized, _)| *sized == kind)
        .map(|(_, bytes)| *bytes)
}
pub(super) fn ordered_kinds(
    kinds: &[GuiTemplateKind],
    sort: TemplateSort,
    state: &GuiCommandState,
    units: GuiTemplateUnits,
) -> Vec<GuiTemplateKind> {
    let mut ordered = kinds.to_vec();
    sort_templates(&mut ordered, sort, |kind| {
        (
            GuiDocumentTemplate { kind: *kind, units }.title(),
            template_size(state, *kind),
        )
    });
    ordered
}

pub(super) fn sort_templates<T>(
    entries: &mut [T],
    sort: TemplateSort,
    values: impl Fn(&T) -> (String, Option<u64>),
) {
    use std::cmp::Reverse;
    match sort {
        TemplateSort::Name(SortOrder::Increasing) => {
            entries.sort_by_cached_key(|entry| values(entry).0.to_lowercase());
        }
        TemplateSort::Name(SortOrder::Decreasing) => {
            entries.sort_by_cached_key(|entry| Reverse(values(entry).0.to_lowercase()));
        }
        TemplateSort::Size(SortOrder::Increasing) => entries.sort_by_key(|entry| values(entry).1),
        TemplateSort::Size(SortOrder::Decreasing) => {
            entries.sort_by_key(|entry| Reverse(values(entry).1));
        }
    }
}
