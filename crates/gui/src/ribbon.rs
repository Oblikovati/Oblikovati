// SPDX-License-Identifier: GPL-2.0-only
use crate::application::ribbon_display::{RibbonDisplay, RibbonPlacement};
use crate::{GuiAuthoringContext, GuiCommandState};
use oblikovati_api::types::{
    ASSEMBLY_RIBBON, DOCUMENT_ASSEMBLY, DOCUMENT_DRAWING, DOCUMENT_PART, DOCUMENT_PRESENTATION,
    DRAWING_RIBBON, PART_RIBBON, PRESENTATION_RIBBON, RibbonKey, SUB_TYPE_SHEET_METAL_PART,
    UNKNOWN_DOCUMENT_RIBBON, ZERO_DOC_RIBBON,
};
#[path = "ribbon_reset.rs"]
mod reset;
#[path = "ribbon_team_web.rs"]
pub(crate) mod team_web;

impl DocumentRibbon {
    pub(crate) fn customization_panel(
        &mut self,
        ui: &imgui::Ui,
        commands: &GuiCommandState,
        actions: &mut Vec<crate::GuiShellAction>,
        (quick_access, workspace): (
            &crate::application::quick_access_toolbar::QuickAccessToolbar,
            &crate::application::workspace_layout::WorkspaceLayout,
        ),
    ) {
        use crate::application::ribbon_customization::interface_export::InterfaceExport;
        self.customization.panel(
            ui,
            commands,
            actions,
            InterfaceExport {
                ribbon: &self.display,
                quick_access,
                marking_menu_enabled: workspace.marking_menu_enabled,
                mini_toolbar_visible: workspace.selection_toolbar_visible,
            },
        );
    }

    pub(crate) fn apply_context_actions(&mut self, tab: RibbonTab) {
        if let Some((owner, panel)) = self.display.take_context_relocated() {
            self.panels.close_additional(owner, panel);
        }
        if self.display.take_context_expansion_close() {
            self.panels.close_unpinned_additional();
        }
        if self.display.take_customization_request() {
            self.panels.close_unpinned_additional();
            self.customization.open(tab);
        }
    }
}

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, PartialOrd, Ord)]
pub(crate) enum RibbonTab {
    #[default]
    GetStarted,
    Model,
    SheetMetal,
    Assemble,
    PlaceViews,
    Annotate,
    Presentation,
    Sketch,
    Tools,
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
            Self::Annotate => "Annotate",
            Self::Presentation => "Presentation",
            Self::Sketch => "Sketch",
            Self::Tools => "Tools",
            Self::Manage => "Manage",
            Self::View => "View",
        }
    }
}
#[derive(Default)]
pub(crate) struct DocumentRibbon {
    previous: Option<(RibbonKey, bool, bool)>,
    previous_document: Option<u64>,
    pre_sketch_tab: Option<(u64, RibbonTab)>,
    before_home: Option<(u64, RibbonTab, RibbonPlacement)>,
    pub active: RibbonTab,
    pub tips: crate::ribbon_key_tips::RibbonKeyTips,
    pub panels: crate::ribbon_panels::RibbonPanelLayout,
    pub display: crate::application::ribbon_display::RibbonDisplayState,
    pub customization: crate::application::ribbon_customization::RibbonCustomization,
    pub team_web: team_web::TeamWebWindow,
    settling_frames: u8,
    pending_tab: Option<RibbonTab>,
    pub double_click_tab: Option<RibbonTab>,
    pub visible_tabs: Vec<RibbonTab>,
    pub tab_popup_press: crate::popup_press::PopupPress<RibbonTab>,
    reset_error: Option<String>,
    reset_open: bool,
}
pub(crate) struct RibbonFrame {
    pub tabs: Vec<RibbonTab>,
    pub select: Option<RibbonTab>,
}
const TAB_SETTLING_FRAMES: u8 = 2;
impl DocumentRibbon {
    /// Settle a changed tab subset after ImGui lays out its new entries.
    pub(crate) fn tabs_changed(&mut self, visible: &[RibbonTab]) -> bool {
        if self.visible_tabs == visible {
            return false;
        }
        self.visible_tabs = visible.to_vec();
        self.settling_frames = TAB_SETTLING_FRAMES;
        true
    }
    pub(crate) fn tabs_settling(&self) -> bool {
        self.settling_frames > 0
    }
    pub(crate) fn select_tab(&mut self, tab: RibbonTab) {
        if self.active != tab
            && self.display.placement != crate::application::ribbon_display::RibbonPlacement::Top
        {
            self.display.temporary = None;
            self.display.side_anchor = None;
        }
        self.active = tab;
        self.pending_tab = Some(tab);
    }
    pub(crate) fn needs_frame(&self) -> bool {
        self.settling_frames > 0 || self.pending_tab.is_some()
    }
    /// Use the zero-document ribbon while Home covers an open document.
    /// For example, the Part's Tools tab is restored after visiting Home.
    pub(crate) fn prepare_workspace(
        &mut self,
        commands: &GuiCommandState,
        home_over_document: Option<u64>,
    ) -> RibbonFrame {
        let selected = self.switch_home_ribbon(commands.document, home_over_document);
        let mut frame = self.prepare(commands);
        if let Some(tab) = selected.filter(|tab| frame.tabs.contains(tab)) {
            self.active = tab;
            frame.select = Some(tab);
        }
        frame
    }

    fn switch_home_ribbon(
        &mut self,
        document: Option<u64>,
        home_over_document: Option<u64>,
    ) -> Option<RibbonTab> {
        let selected = match (home_over_document, self.before_home) {
            (Some(document), None) => {
                self.before_home = Some((document, self.active, self.display.placement));
                self.display.placement = RibbonPlacement::Top;
                (self.active == RibbonTab::Tools).then_some(RibbonTab::Tools)
            }
            (None, Some((saved_document, tab, placement))) => {
                self.before_home = None;
                // Selecting a document can dismiss the flyout before the workspace changes.
                let opened_tools =
                    self.display.mode == RibbonDisplay::Tabs && self.active == RibbonTab::Tools;
                self.display.placement = placement;
                (document == Some(saved_document)).then_some(if opened_tools {
                    RibbonTab::Tools
                } else {
                    tab
                })
            }
            _ => return None,
        };
        self.display.clear_workspace_temporary();
        selected
    }
    pub(crate) fn prepare(&mut self, state: &GuiCommandState) -> RibbonFrame {
        let key = document_ribbon_key(state);
        let sketch = state.authoring == GuiAuthoringContext::Sketch;
        let sheet_metal = active_sheet_metal(state);
        let tabs = document_tabs(&key, sketch, sheet_metal);
        let identity = (key, sketch, sheet_metal);
        let changed = self.previous.as_ref() != Some(&identity);
        self.settling_frames = self.settling_frames.saturating_sub(1);
        if changed {
            // Dear ImGui lays a new tab out in its second frame, and GUI frames wait for an event.
            self.settling_frames = TAB_SETTLING_FRAMES;
            self.update_contextual_tab(state.document, &identity, &tabs);
            self.previous = Some(identity);
            self.pending_tab = None;
        }
        self.previous_document = state.document;
        RibbonFrame {
            tabs,
            select: if changed {
                Some(self.active)
            } else {
                self.pending_tab.take()
            },
        }
    }
    fn update_contextual_tab(
        &mut self,
        document: Option<u64>,
        identity: &(RibbonKey, bool, bool),
        tabs: &[RibbonTab],
    ) {
        let was_sketch = self.previous.as_ref().is_some_and(|(_, sketch, _)| *sketch);
        if identity.1 && !was_sketch && document == self.previous_document {
            self.pre_sketch_tab = document.map(|id| (id, self.active));
        }
        if self.keep_tab_after_sketch(identity, tabs) {
            return;
        }
        self.active = self
            .restored_part_tab(document, identity, tabs)
            .unwrap_or_else(|| contextual_tab(tabs, identity.1));
    }
    fn restored_part_tab(
        &self,
        document: Option<u64>,
        identity: &(RibbonKey, bool, bool),
        tabs: &[RibbonTab],
    ) -> Option<RibbonTab> {
        if identity.1 || !self.previous.as_ref().is_some_and(|(_, sketch, _)| *sketch) {
            return None;
        }
        let (saved_document, tab) = self.pre_sketch_tab?;
        (document == Some(saved_document) && tabs.contains(&tab)).then_some(tab)
    }
    fn keep_tab_after_sketch(
        &self,
        identity: &(RibbonKey, bool, bool),
        tabs: &[RibbonTab],
    ) -> bool {
        self.previous
            .as_ref()
            .is_some_and(|(old_key, old_sketch, old_sheet)| {
                old_key == &identity.0 && *old_sketch && !identity.1 && *old_sheet == identity.2
            })
            && self.active != RibbonTab::Sketch
            && tabs.contains(&self.active)
    }
    pub(crate) fn scroll_tabs(&mut self, tabs: &[RibbonTab], wheel: f32) {
        let Some(index) = tabs.iter().position(|tab| *tab == self.active) else {
            return;
        };
        let next = if wheel > 0. {
            index.saturating_sub(1)
        } else if wheel < 0. {
            (index + 1).min(tabs.len() - 1)
        } else {
            return;
        };
        if next == index {
            return;
        }
        self.select_tab(tabs[next]);
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
        return vec![RibbonTab::GetStarted, RibbonTab::Tools];
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
        tabs.push(RibbonTab::Annotate);
    }
    if *key == PRESENTATION_RIBBON {
        tabs.push(RibbonTab::Presentation);
    }
    if *key == PART_RIBBON || *key == ASSEMBLY_RIBBON {
        if *key == PART_RIBBON || sketch {
            tabs.push(RibbonTab::Sketch);
        }
        tabs.push(RibbonTab::Tools);
        tabs.push(RibbonTab::Manage);
    } else {
        tabs.push(RibbonTab::Tools);
    }
    tabs.push(RibbonTab::View);
    if *key == PART_RIBBON {
        tabs.push(RibbonTab::Annotate);
    }
    tabs
}
fn contextual_tab(tabs: &[RibbonTab], sketch: bool) -> RibbonTab {
    // Enter sketch on its contextual tab; otherwise use the document's first tab.
    if sketch && tabs.contains(&RibbonTab::Sketch) {
        return RibbonTab::Sketch;
    }
    tabs[0]
}

#[cfg(test)]
#[path = "../tests/ribbon/unit.rs"]
mod tests;
