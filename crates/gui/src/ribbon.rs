// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_api::types::{
    ASSEMBLY_RIBBON, DOCUMENT_ASSEMBLY, DOCUMENT_DRAWING, DOCUMENT_PART, DOCUMENT_PRESENTATION,
    DRAWING_RIBBON, PART_RIBBON, PRESENTATION_RIBBON, RibbonKey, UNKNOWN_DOCUMENT_RIBBON,
    ZERO_DOC_RIBBON,
};

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, PartialOrd, Ord)]
pub(crate) enum RibbonTab {
    #[default]
    GetStarted,
    CreateModify,
    Sketch,
    Manage,
    View,
}
impl RibbonTab {
    pub(crate) fn label(self) -> &'static str {
        match self {
            Self::GetStarted => "Get Started",
            Self::CreateModify => "Create & Modify",
            Self::Sketch => "Sketch",
            Self::Manage => "Manage",
            Self::View => "View",
        }
    }
}
#[derive(Default)]
pub(crate) struct DocumentRibbon {
    previous: Option<(RibbonKey, bool)>,
    pub active: RibbonTab,
    pub panels: crate::ribbon_panels::RibbonPanelLayout,
}
pub(crate) struct RibbonFrame {
    pub tabs: Vec<RibbonTab>,
    pub select: Option<RibbonTab>,
}
impl DocumentRibbon {
    pub(crate) fn prepare(&mut self, state: &GuiCommandState) -> RibbonFrame {
        let key = document_ribbon_key(state);
        let sketch = state.authoring == GuiAuthoringContext::Sketch;
        let tabs = document_tabs(&key, sketch);
        let identity = (key, sketch);
        let changed = self.previous.as_ref() != Some(&identity);
        if changed {
            self.active = contextual_tab(&tabs, sketch);
            self.previous = Some(identity);
        }
        RibbonFrame {
            tabs,
            select: changed.then_some(self.active),
        }
    }
}
fn document_ribbon_key(state: &GuiCommandState) -> RibbonKey {
    if state.document.is_none() {
        return ZERO_DOC_RIBBON;
    }
    match state.document_type {
        Some(DOCUMENT_PART) => PART_RIBBON,
        Some(DOCUMENT_ASSEMBLY) => ASSEMBLY_RIBBON,
        Some(DOCUMENT_DRAWING) => DRAWING_RIBBON,
        Some(DOCUMENT_PRESENTATION) => PRESENTATION_RIBBON,
        _ => UNKNOWN_DOCUMENT_RIBBON,
    }
}
fn document_tabs(key: &RibbonKey, sketch: bool) -> Vec<RibbonTab> {
    if *key == ZERO_DOC_RIBBON {
        return vec![RibbonTab::GetStarted];
    }
    let mut tabs = Vec::new();
    if *key == PART_RIBBON {
        tabs.push(RibbonTab::CreateModify);
    }
    if *key == PART_RIBBON || *key == ASSEMBLY_RIBBON {
        if sketch {
            tabs.push(RibbonTab::Sketch);
        }
        tabs.push(RibbonTab::Manage);
    }
    tabs.push(RibbonTab::View);
    tabs
}
fn contextual_tab(tabs: &[RibbonTab], sketch: bool) -> RibbonTab {
    // Go chrome_ribbon.go: switch on entry/exit only, preserving manual tab choices afterwards.
    if sketch && tabs.contains(&RibbonTab::Sketch) {
        return RibbonTab::Sketch;
    }
    tabs[0]
}

#[cfg(test)]
#[path = "../tests/ribbon/unit.rs"]
mod tests;
