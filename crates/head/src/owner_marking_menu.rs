// SPDX-License-Identifier: GPL-2.0-only
impl<O, W, F, G, R, S: super::DispatchWakeup> super::DesktopOwner<O, W, F, G, R, S> {
    /// Load and attach marking-menu storage; example: `owner.with_marking_menu_preferences(store)?`.
    pub fn with_marking_menu_preferences(
        mut self,
        store: Box<dyn oblikovati_model::marking_menu::MarkingMenuPreferenceAccess>,
    ) -> Result<Self, String> {
        for menu in store.menus() {
            self.marking_menus.set(menu)?;
        }
        self.classic_marking_menu = store.classic();
        self.marking_store = Some(store);
        Ok(self)
    }
}

impl<
    O: super::TransactionObserver,
    W: super::WorkspaceObserver,
    F: super::DocumentFiles,
    G: super::GuidSource,
    R: super::DocumentObserverFactory<Observer = O> + super::HeadInvalidation,
    S: super::DispatchWakeup,
> super::DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn set_marking_menu_action(
        &mut self,
        args: &oblikovati_api::wire::SetMarkingMenuArgs,
    ) -> Result<bool, String> {
        oblikovati_api::client::Client::new(self.document_host())
            .ui()
            .set_marking_menu(args)
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
}
