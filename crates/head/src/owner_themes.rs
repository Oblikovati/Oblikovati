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
    /// Share one library with API queries and the GUI; example: `owner.with_themes(library)`.
    pub fn with_themes(mut self, library: oblikovati_model::theme::ThemeLibrary) -> Self {
        self.themes = Some(library);
        self.theme_store = None;
        self
    }
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
    pub(super) fn gui_theme_snapshot(&self) -> Option<oblikovati_gui::GuiThemeState> {
        let library = self.themes.as_ref()?;
        let can_save = self
            .theme_store
            .as_ref()
            .is_some_and(|store| store.can_save_custom(library.active()));
        Some(
            oblikovati_gui::GuiThemeState::from_library(library)
                .with_custom_save_available(can_save)
                .with_lifecycle_available(
                    self.theme_store
                        .as_ref()
                        .is_none_or(|store| store.can_create_custom()),
                    self.theme_store.is_none() || can_save,
                ),
        )
    }
    pub(super) fn preview_custom_theme_color(
        &mut self,
        name: &str,
        token: &oblikovati_api::types::ThemeToken,
        value: oblikovati_api::types::Rgba,
    ) -> Result<bool, String> {
        active_custom_library(&mut self.themes, name)?.edit_active_color(token, value)
    }
    pub(super) fn persist_active_custom_theme(&mut self, name: &str) -> Result<bool, String> {
        let library = active_custom_library(&mut self.themes, name)?;
        let store = self
            .theme_store
            .as_mut()
            .filter(|store| store.can_save_custom(library.active()))
            .ok_or_else(|| format!("theme {name:?}: expected a writable custom XML source"))?;
        store.save_custom_theme(library.active())?;
        self.command_notice = Some(DesktopCommandNotice::ThemeSaved { name: name.into() });
        Ok(true)
    }
    pub(super) fn duplicate_application_theme(
        &mut self,
        base: &str,
        name: &str,
    ) -> Result<bool, String> {
        let library = self
            .themes
            .as_mut()
            .ok_or_else(|| format!("theme {base:?}: expected an application theme library"))?;
        let candidate = duplicate_candidate(library, base, name)?;
        if let Some(store) = &mut self.theme_store {
            store.create_custom_theme(candidate.active())?;
        }
        publish_duplicate_selection(library, &mut self.theme_store, candidate, base, name)
    }
    pub(super) fn delete_application_theme(&mut self, name: &str) -> Result<bool, String> {
        let library = active_custom_library(&mut self.themes, name)?;
        let mut candidate = library.clone();
        candidate.remove(name)?;
        if let Some(store) = &mut self.theme_store {
            store.remove_custom_theme(library.active())?;
        }
        *library = candidate;
        if let Some(store) = &mut self.theme_store {
            store.save_selection(library.active()).map_err(|cause| {
                format!("theme {name:?} was deleted but selection was not saved: {cause}")
            })?;
        }
        Ok(true)
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

fn active_custom_library<'a>(
    themes: &'a mut Option<ThemeLibrary>,
    name: &str,
) -> Result<&'a mut ThemeLibrary, String> {
    let library = themes
        .as_mut()
        .ok_or_else(|| format!("theme {name:?}: expected an application theme library"))?;
    if library.active().name() != name
        || library.active().kind() != &oblikovati_api::types::THEME_CUSTOM
    {
        return Err(format!(
            "theme {name:?}: expected the active editable custom theme {:?}",
            library.active().name()
        ));
    }
    Ok(library)
}

fn duplicate_candidate(
    library: &ThemeLibrary,
    base: &str,
    name: &str,
) -> Result<ThemeLibrary, String> {
    if library.active().name() != base {
        return Err(format!(
            "theme {base:?}: expected active theme {:?}",
            library.active().name()
        ));
    }
    let mut candidate = library.clone();
    candidate.duplicate(base, name)?;
    Ok(candidate)
}
fn publish_duplicate_selection(
    library: &mut ThemeLibrary,
    store: &mut Option<Box<dyn ThemeSelectionStore>>,
    mut candidate: ThemeLibrary,
    base: &str,
    name: &str,
) -> Result<bool, String> {
    let selection = store
        .as_mut()
        .map_or(Ok(()), |store| store.save_selection(candidate.active()));
    if let Err(cause) = selection {
        candidate.set_active(base);
        *library = candidate;
        return Err(format!(
            "theme {name:?} was created but selection was not saved: {cause}"
        ));
    }
    *library = candidate;
    Ok(true)
}
