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
        menu: Option<&oblikovati_api::wire::MarkingMenuView>,
        classic: Option<bool>,
    ) -> Result<bool, String> {
        let Some(classic) = classic else {
            let Some(menu) = menu else { return Ok(false) };
            oblikovati_api::client::Client::new(self.document_host())
                .ui()
                .set_marking_menu(&oblikovati_api::wire::SetMarkingMenuArgs { menu: menu.clone() })
                .map_err(|cause| cause.to_string())?;
            return Ok(true);
        };
        let mut menus = self.marking_menus.clone();
        if let Some(menu) = menu {
            menus.set(menu.clone())?;
        }
        if let Some(store) = &mut self.marking_store {
            store.save_with_classic(&menus, classic)?;
        }
        self.marking_menus = menus;
        self.classic_marking_menu = classic;
        Ok(true)
    }
}
