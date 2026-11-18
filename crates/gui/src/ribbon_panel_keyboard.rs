// SPDX-License-Identifier: GPL-2.0-only
//! Route keyboard requests to a visible ribbon panel.
use super::*;
impl RibbonPanelLayout {
    pub(crate) fn has_pending_model_pick(&self) -> bool {
        self.pending_model_pick
    }
    pub(crate) fn begin_model_pick(&mut self) {
        self.pending_model_pick = true;
    }
    pub(crate) fn has_additional(&self) -> bool {
        self.additional.has_open()
    }
    pub(crate) fn resolve_model_pick(&mut self, face: bool) -> bool {
        if !std::mem::take(&mut self.pending_model_pick) {
            return false;
        }
        self.retained_model_face = face;
        if !face {
            self.close_unpinned_additional();
        }
        true
    }
    pub(crate) fn forget_retained_face(&mut self) {
        self.retained_model_face = false;
    }
    pub(crate) fn cancel_retained_face(&mut self) -> bool {
        if !std::mem::take(&mut self.retained_model_face) {
            return false;
        }
        self.close_unpinned_additional();
        true
    }
    pub(crate) fn temporary_tab_press(&mut self) -> Option<bool> {
        self.additional.temporary_tab_press()
    }
    pub(crate) fn close_unpinned_additional(&mut self) {
        self.additional.close_unpinned();
    }
    pub(crate) fn close_additional(&mut self, tab: RibbonTab, panel: RibbonPanel) {
        self.additional.close_one(tab, panel);
    }
    pub(crate) fn request_additional(&mut self, tab: RibbonTab, panel: RibbonPanel) {
        self.pending_additional = Some((tab, panel));
    }
    pub(super) fn take_additional(&mut self, tab: RibbonTab, panel: RibbonPanel) -> bool {
        if self.pending_additional != Some((tab, panel)) {
            return false;
        }
        self.pending_additional = None;
        true
    }
}
