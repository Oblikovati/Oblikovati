// SPDX-License-Identifier: GPL-2.0-only
//! Go-compatible, atomically replaced marking-menu settings.
use crate::{
    atomic::{AtomicFiles, replace_file},
    load::PackageFileReader,
    preference_revision::PreferenceRevision,
};
use oblikovati_api::{
    types::{Environment, ScreenQuadrant},
    wire::{MarkingMenuItem, MarkingMenuView},
};
use oblikovati_model::marking_menu::{MarkingMenuPreferenceAccess, MarkingMenus};
use serde::{Deserialize, Serialize};
use std::{
    collections::BTreeMap,
    io,
    path::{Path, PathBuf},
};

#[derive(Clone, Default, Serialize, Deserialize)]
#[serde(default)]
struct MenuDocument {
    menus: Vec<StoredMenu>,
    classic: bool,
    #[serde(flatten)]
    retained: BTreeMap<String, serde_yaml_ng::Value>,
}
#[derive(Clone, Default, Serialize, Deserialize)]
#[serde(default)]
struct StoredMenu {
    environment: u8,
    quadrants: Vec<StoredSlot>,
    overflow: Vec<String>,
}
#[derive(Clone, Default, Serialize, Deserialize)]
#[serde(default)]
struct StoredSlot {
    quadrant: u8,
    command: String,
}

/// Own the Go-compatible marking menu file.
pub struct MarkingMenuPreferenceFile<Files> {
    path: PathBuf,
    files: Files,
    revision: PreferenceRevision,
    document: MenuDocument,
}
impl<Files: PackageFileReader + AtomicFiles> MarkingMenuPreferenceFile<Files> {
    /// Read existing settings; example: `MarkingMenuPreferenceFile::load(path, files)?`.
    pub fn load(path: &Path, mut files: Files) -> io::Result<Self> {
        let revision =
            PreferenceRevision::read(path, &mut files).map_err(|error| failure(path, error))?;
        let document = serde_yaml_ng::from_slice::<Option<MenuDocument>>(revision.contents())
            .map_err(|error| failure(path, error))?
            .unwrap_or_default();
        validate_document(path, &document)?;
        Ok(Self {
            path: path.into(),
            files,
            revision,
            document,
        })
    }
}
impl<Files: PackageFileReader + AtomicFiles> MarkingMenuPreferenceAccess
    for MarkingMenuPreferenceFile<Files>
{
    fn classic(&self) -> bool {
        self.document.classic
    }
    fn menus(&self) -> Vec<MarkingMenuView> {
        self.document.menus.iter().map(to_view).collect()
    }
    fn save(&mut self, menus: &MarkingMenus) -> Result<(), String> {
        let mut candidate = self.document.clone();
        candidate.menus = menus.all().iter().map(to_stored).collect();
        let bytes = crate::preference_yaml_guard::encode(&candidate)
            .map_err(|error| failure(&self.path, error).to_string())?;
        self.revision
            .require_current(&self.path, &mut self.files)
            .map_err(|error| failure(&self.path, error).to_string())?;
        replace_file(&self.path, bytes.as_bytes(), &mut self.files)
            .map_err(|error| failure(&self.path, error).to_string())?;
        self.revision = PreferenceRevision::Contents(bytes.into_bytes());
        self.document = candidate;
        Ok(())
    }
}
fn to_view(menu: &StoredMenu) -> MarkingMenuView {
    MarkingMenuView {
        environment: Environment(menu.environment),
        quadrants: Some(
            menu.quadrants
                .iter()
                .map(|slot| MarkingMenuItem {
                    quadrant: ScreenQuadrant(slot.quadrant),
                    command_id: slot.command.clone(),
                })
                .collect(),
        ),
        overflow: Some(menu.overflow.clone()),
    }
}
fn to_stored(menu: &MarkingMenuView) -> StoredMenu {
    StoredMenu {
        environment: menu.environment.0,
        quadrants: menu
            .quadrants
            .iter()
            .flatten()
            .map(|slot| StoredSlot {
                quadrant: slot.quadrant.0,
                command: slot.command_id.clone(),
            })
            .collect(),
        overflow: menu.overflow.clone().unwrap_or_default(),
    }
}
fn validate_document(path: &Path, document: &MenuDocument) -> io::Result<()> {
    let mut seen = std::collections::BTreeSet::new();
    let mut menus = MarkingMenus::default();
    for entry in &document.menus {
        if !seen.insert(entry.environment) {
            return Err(failure(
                path,
                format!("environment {}: expected one menu", entry.environment),
            ));
        }
        menus
            .set(to_view(entry))
            .map_err(|error| failure(path, error))?;
    }
    Ok(())
}
fn failure(path: &Path, cause: impl std::fmt::Display) -> io::Error {
    io::Error::other(format!(
        "marking menu preferences {path:?}: expected valid readable and writable YAML: {cause}"
    ))
}
