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
