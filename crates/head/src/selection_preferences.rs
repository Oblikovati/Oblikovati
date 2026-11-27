// SPDX-License-Identifier: GPL-2.0-only
//! Atomic application-level selection-filter preferences.
use oblikovati_persistence::{
    atomic::{AtomicFiles, replace_file},
    load::PackageFileReader,
};
use oblikovati_session::selection::{AmbientSelectionFilter, SelectionKind};
use serde::{Deserialize, Serialize};
use std::{
    io,
    path::{Path, PathBuf},
};

/// Stores the user's selection filter independently of document history.
pub trait SelectionPreferenceAccess {
    /// Read the last published filter; example: `store.selection_filter().order()`.
    fn selection_filter(&self) -> &AmbientSelectionFilter;
    /// Read the saved preset; example: `store.selection_mode()`.
    fn selection_mode(&self) -> oblikovati_gui::GuiSelectionMode;
    /// Persist a complete filter before publishing it; example: `store.set_selection_filter(&filter, mode)?`.
    fn set_selection_filter(
        &mut self,
        filter: &AmbientSelectionFilter,
        mode: oblikovati_gui::GuiSelectionMode,
    ) -> Result<(), String>;
}

#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct SelectionRecipe {
    version: u32,
    mode: oblikovati_gui::GuiSelectionMode,
    entries: Vec<SelectionEntry>,
}
#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct SelectionEntry {
    kind: SelectionKind,
    enabled: bool,
}

pub(crate) struct SelectionPreferenceFile<F> {
    path: PathBuf,
    files: F,
    filter: AmbientSelectionFilter,
    mode: oblikovati_gui::GuiSelectionMode,
}
impl<F: PackageFileReader + AtomicFiles> SelectionPreferenceFile<F> {
    pub(crate) fn load(path: &Path, mut files: F) -> Result<Self, String> {
        let (filter, mode) = match files.read(path) {
            Ok(bytes) => decode(&bytes)
                .map_err(|cause| format!("selection preferences {path:?}: {cause}"))?,
            Err(error) if error.kind() == io::ErrorKind::NotFound => (
                AmbientSelectionFilter::default(),
                oblikovati_gui::GuiSelectionMode::All,
            ),
            Err(error) => {
                return Err(format!(
                    "selection preferences {path:?}: expected a readable file: {error}"
                ));
            }
        };
        Ok(Self {
            path: path.into(),
            files,
            filter,
            mode,
        })
    }
}
impl<F: PackageFileReader + AtomicFiles> SelectionPreferenceAccess for SelectionPreferenceFile<F> {
    fn selection_filter(&self) -> &AmbientSelectionFilter {
        &self.filter
    }
    fn selection_mode(&self) -> oblikovati_gui::GuiSelectionMode {
        self.mode
    }
    fn set_selection_filter(
        &mut self,
        filter: &AmbientSelectionFilter,
        mode: oblikovati_gui::GuiSelectionMode,
    ) -> Result<(), String> {
        validate_mode(filter, mode)?;
        let recipe = SelectionRecipe {
            version: 1,
            mode,
            entries: filter
                .order()
                .iter()
                .map(|kind| SelectionEntry {
                    kind: *kind,
                    enabled: filter.enabled(*kind),
                })
                .collect(),
        };
        let bytes = serde_json::to_vec(&recipe).map_err(|error| {
            format!(
                "selection preferences {:?}: expected serializable entries: {error}",
                self.path
            )
        })?;
        replace_file(&self.path, &bytes, &mut self.files).map_err(|error| error.to_string())?;
        self.filter = filter.clone();
        self.mode = mode;
        Ok(())
    }
}
fn decode(
    bytes: &[u8],
) -> Result<(AmbientSelectionFilter, oblikovati_gui::GuiSelectionMode), String> {
    let recipe: SelectionRecipe = serde_json::from_slice(bytes).map_err(|error| {
        format!("selection preferences: expected versioned JSON entries: {error}")
    })?;
    if recipe.version != 1 {
        return Err(format!(
            "selection preference version {}: expected 1",
            recipe.version
        ));
    }
    let entries: Vec<_> = recipe
        .entries
        .into_iter()
        .map(|row| (row.kind, row.enabled))
        .collect();
    let filter = AmbientSelectionFilter::from_entries(&entries)?;
    validate_mode(&filter, recipe.mode)?;
    Ok((filter, recipe.mode))
}
fn validate_mode(
    filter: &AmbientSelectionFilter,
    mode: oblikovati_gui::GuiSelectionMode,
) -> Result<(), String> {
    use oblikovati_gui::GuiSelectionMode;
    use oblikovati_session::selection::SelectionPriority;
    let preset = match mode {
        GuiSelectionMode::All => SelectionPriority::General,
        GuiSelectionMode::Faces => SelectionPriority::Face,
        GuiSelectionMode::Components => SelectionPriority::Part,
        GuiSelectionMode::Custom => return Ok(()),
    };
    let mut expected = AmbientSelectionFilter::default();
    expected.apply_priority(preset);
    if filter
        .order()
        .iter()
        .any(|kind| filter.enabled(*kind) != expected.enabled(*kind))
    {
        return Err(format!(
            "selection mode {mode:?}: expected enabled categories matching its preset"
        ));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_gui::GuiSelectionMode;

    fn recipe() -> SelectionRecipe {
        let filter = AmbientSelectionFilter::default();
        SelectionRecipe {
            version: 1,
            mode: GuiSelectionMode::All,
            entries: filter
                .order()
                .iter()
                .map(|kind| SelectionEntry {
                    kind: *kind,
                    enabled: true,
                })
                .collect(),
        }
    }
    #[test]
    fn decoding_rejects_invalid_versions_duplicate_categories_and_inconsistent_modes() {
        let mut value = recipe();
        assert_eq!(
            decode(&serde_json::to_vec(&value).unwrap()).unwrap().0,
            AmbientSelectionFilter::default()
        );
        value.version = 2;
        assert!(
            decode(&serde_json::to_vec(&value).unwrap())
                .unwrap_err()
                .contains("version")
        );
        value.version = 1;
        value.entries[1].kind = value.entries[0].kind;
        assert!(
            decode(&serde_json::to_vec(&value).unwrap())
                .unwrap_err()
                .contains("exactly once")
        );
        let mut value = recipe();
        value.entries[0].enabled = false;
        assert!(
            decode(&serde_json::to_vec(&value).unwrap())
                .unwrap_err()
                .contains("matching its preset")
        );
    }
}
