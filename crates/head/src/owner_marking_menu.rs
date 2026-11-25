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
        self.marking_menu_display = store.display();
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
        assignments: &[oblikovati_api::wire::MarkingMenuView],
        display: Option<oblikovati_model::marking_menu::MarkingMenuDisplay>,
    ) -> Result<bool, String> {
        let Some(display) = display else {
            let [menu] = assignments else {
                return Err(
                    "marking menu action: expected one menu without display changes".into(),
                );
            };
            oblikovati_api::client::Client::new(self.document_host())
                .ui()
                .set_marking_menu(&oblikovati_api::wire::SetMarkingMenuArgs { menu: menu.clone() })
                .map_err(|cause| cause.to_string())?;
            return Ok(true);
        };
        let mut menus = self.marking_menus.clone();
        for menu in assignments {
            menus.set(menu.clone())?;
        }
        if let Some(store) = &mut self.marking_store {
            store.save_with_display(&menus, display)?;
        }
        self.marking_menus = menus;
        self.marking_menu_display = display;
        Ok(true)
    }
}
