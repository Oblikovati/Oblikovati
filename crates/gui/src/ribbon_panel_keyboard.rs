// SPDX-License-Identifier: GPL-2.0-only
//! Route keyboard requests to a visible ribbon panel.
use super::*;
impl RibbonPanelLayout {
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
