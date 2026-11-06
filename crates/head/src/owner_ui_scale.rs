// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::ui_scale::{UiScaleOptions, UiScalePreferenceAccess};
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// Attach loaded application scale storage; example: `owner.with_ui_scale_preferences(Box::new(preferences))`.
    pub fn with_ui_scale_preferences(mut self, store: Box<dyn UiScalePreferenceAccess>) -> Self {
        self.ui_scale_store = Some(store);
        self
    }
    pub(super) fn gui_ui_scale(&self) -> Option<UiScaleOptions> {
        self.ui_scale_store.as_ref().map(|store| store.ui_scale())
    }
    pub(super) fn persist_ui_scale(&mut self, options: UiScaleOptions) -> Result<bool, String> {
        let store = self.ui_scale_store.as_mut().ok_or_else(|| {
            format!("UI scale {options:?}: expected an application preference owner")
        })?;
        if store.ui_scale() == options {
            return Ok(false);
        }
        store.set_ui_scale(options)?;
        Ok(true)
    }
}
