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
    ordered.sort_by(|a, b| compare_kinds(*a, *b, sort, state, units));
    ordered
}

fn compare_kinds(
    a: GuiTemplateKind,
    b: GuiTemplateKind,
    sort: TemplateSort,
    state: &GuiCommandState,
    units: GuiTemplateUnits,
) -> std::cmp::Ordering {
    let (comparison, order) = match sort {
        TemplateSort::Name(order) => {
            let title = |kind| GuiDocumentTemplate { kind, units }.title().to_lowercase();
            (title(a).cmp(&title(b)), order)
        }
        TemplateSort::Size(order) => (template_size(state, a).cmp(&template_size(state, b)), order),
    };
    match order {
        SortOrder::Increasing => comparison,
        SortOrder::Decreasing => comparison.reverse(),
    }
}
