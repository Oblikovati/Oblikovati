// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::theme::{ThemeDefinition, ThemeLibrary, ThemeSelectionStore};
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// Inject selection storage and its loaded library; example: `owner.with_theme_store(library, Box::new(preferences))`.
    pub fn with_theme_store(
        mut self,
        library: ThemeLibrary,
        store: Box<dyn ThemeSelectionStore>,
    ) -> Self {
        self.themes = Some(library);
        self.theme_store = Some(store);
        self
    }
    /// Surface custom-file declines without hiding valid themes; example: `owner.with_theme_file_declines(loaded.declines)`.
    pub fn with_theme_file_declines(
        mut self,
        declines: Vec<oblikovati_persistence::theme_files::ThemeFileDecline>,
    ) -> Self {
        if !declines.is_empty() {
            let causes = declines
                .into_iter()
                .map(|decline| format!("{:?}: {}", decline.path, decline.cause))
                .collect();
            self.command_notice = Some(DesktopCommandNotice::ThemeFilesDeclined { causes });
        }
        self
    }
    pub(super) fn select_application_theme(&mut self, name: &str) -> Result<bool, String> {
        let library = self.themes.as_mut().ok_or_else(|| {
            format!("theme {name:?}: expected an injected application theme library")
        })?;
        let selected = selected_theme(library, name)?;
        if selected.name() == library.active().name() {
            return Ok(false);
        }
        if let Some(store) = &mut self.theme_store {
            store.save_selection(selected)?;
        }
        Ok(library.set_active(name))
    }
}

fn selected_theme<'a>(
    library: &'a ThemeLibrary,
    name: &str,
) -> Result<&'a ThemeDefinition, String> {
    library
        .themes()
        .iter()
        .find(|theme| theme.name() == name)
        .ok_or_else(|| format!("theme {name:?}: expected a name from the application gallery"))
}
