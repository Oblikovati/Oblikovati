// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_api::types::{
    ASSEMBLY_RIBBON, DOCUMENT_ASSEMBLY, DOCUMENT_DRAWING, DOCUMENT_PART, DOCUMENT_PRESENTATION,
    DRAWING_RIBBON, PART_RIBBON, PRESENTATION_RIBBON, RibbonKey, SUB_TYPE_SHEET_METAL_PART,
    UNKNOWN_DOCUMENT_RIBBON, ZERO_DOC_RIBBON,
};

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, PartialOrd, Ord)]
pub(crate) enum RibbonTab {
    #[default]
    GetStarted,
    Model,
    SheetMetal,
    Assemble,
    PlaceViews,
    Presentation,
    Sketch,
    Manage,
    View,
}
impl RibbonTab {
    pub(crate) fn label(self) -> &'static str {
        match self {
            Self::GetStarted => "Get Started",
            Self::Model => "3D Model",
            Self::SheetMetal => "Sheet Metal",
            Self::Assemble => "Assemble",
            Self::PlaceViews => "Place Views",
            Self::Presentation => "Presentation",
            Self::Sketch => "Sketch",
            Self::Manage => "Manage",
            Self::View => "View",
        }
    }
}
#[derive(Default)]
pub(crate) struct DocumentRibbon {
    previous: Option<(RibbonKey, bool, bool)>,
    pub active: RibbonTab,
    pub panels: crate::ribbon_panels::RibbonPanelLayout,
    pub display: crate::application::ribbon_display::RibbonDisplayState,
}
pub(crate) struct RibbonFrame {
    pub tabs: Vec<RibbonTab>,
    pub select: Option<RibbonTab>,
}
impl DocumentRibbon {
    pub(crate) fn prepare(&mut self, state: &GuiCommandState) -> RibbonFrame {
        let key = document_ribbon_key(state);
        let sketch = state.authoring == GuiAuthoringContext::Sketch;
        let sheet_metal = active_sheet_metal(state);
        let tabs = document_tabs(&key, sketch, sheet_metal);
        let identity = (key, sketch, sheet_metal);
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
fn active_sheet_metal(state: &GuiCommandState) -> bool {
    state
        .documents
        .iter()
        .any(|row| Some(row.id) == state.document && row.sub_type == SUB_TYPE_SHEET_METAL_PART.0)
}
fn document_tabs(key: &RibbonKey, sketch: bool, sheet_metal: bool) -> Vec<RibbonTab> {
    if *key == ZERO_DOC_RIBBON {
        return vec![RibbonTab::GetStarted];
    }
    let mut tabs = Vec::new();
    if *key == PART_RIBBON {
        tabs.push(if sheet_metal {
            RibbonTab::SheetMetal
        } else {
            RibbonTab::Model
        });
    }
    if *key == ASSEMBLY_RIBBON {
        tabs.push(RibbonTab::Assemble);
    }
    if *key == DRAWING_RIBBON {
        tabs.push(RibbonTab::PlaceViews);
    }
    if *key == PRESENTATION_RIBBON {
        tabs.push(RibbonTab::Presentation);
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
