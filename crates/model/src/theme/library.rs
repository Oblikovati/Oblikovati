// SPDX-License-Identifier: GPL-2.0-only
use super::{Rgba, ThemeDefinition, ThemeToken, validate_name};
use std::collections::BTreeSet;
#[derive(Clone, Debug)]
pub struct ThemeLibrary {
    themes: Vec<ThemeDefinition>,
    active: usize,
    revision: u64,
}
impl ThemeLibrary {
    /// Combine built-ins and loaded customs without ambiguous names; example: `ThemeLibrary::new(Vec::new(), "Dark")?`.
    pub fn new(customs: Vec<ThemeDefinition>, active: &str) -> Result<Self, String> {
        let mut themes = vec![ThemeDefinition::dark()?, ThemeDefinition::light()?];
        validate_customs(&customs, &themes)?;
        themes.extend(customs);
        let active = themes
            .iter()
            .position(|theme| theme.name() == active)
            .unwrap_or(0);
        Ok(Self {
            themes,
            active,
            revision: 1,
        })
    }
    /// Read themes in gallery order; example: `library.themes()[0].name()`.
    pub fn themes(&self) -> &[ThemeDefinition] {
        &self.themes
    }
    /// Read the selected complete snapshot; example: `library.active().color(token)`.
    pub fn active(&self) -> &ThemeDefinition {
        &self.themes[self.active]
    }
    /// Read the restyle revision; example: `let before = library.revision();`.
    pub fn revision(&self) -> u64 {
        self.revision
    }
    /// Iterate the snapshots that preference storage persists; example: `library.customs().count()`.
    pub fn customs(&self) -> impl Iterator<Item = &ThemeDefinition> {
        self.themes
            .iter()
            .filter(|theme| theme.kind() == &oblikovati_api::types::THEME_CUSTOM)
    }
    /// Select a known theme; unknown and already-active names are no-ops, as in Go. Example: `library.set_active("Light")`.
    pub fn set_active(&mut self, name: &str) -> bool {
        let Some(index) = self.themes.iter().position(|theme| theme.name() == name) else {
            return false;
        };
        if index == self.active {
            return false;
        }
        self.active = index;
        self.bump_revision();
        true
    }
    /// Add and select an independent custom copy; example: `library.duplicate("Dark", "Workshop")?;`.
    pub fn duplicate(&mut self, base: &str, name: &str) -> Result<(), String> {
        let source = self.duplicate_source(base, name)?;
        let duplicate = source.duplicate(name)?;
        self.themes.push(duplicate);
        self.active = self.themes.len() - 1;
        self.bump_revision();
        Ok(())
    }
    /// Remove a custom theme, returning to Dark when active; example: `library.remove("Workshop")?;`.
    pub fn remove(&mut self, name: &str) -> Result<(), String> {
        let index = self.custom_index(name)?;
        self.themes.remove(index);
        self.active = if index == self.active {
            0
        } else {
            self.active - usize::from(index < self.active)
        };
        self.bump_revision();
        Ok(())
    }
    /// Apply a custom edit and advance the live-preview revision; example: `library.edit_active_color(token, rgba)?;`.
    pub fn edit_active_color(&mut self, token: &ThemeToken, value: Rgba) -> Result<bool, String> {
        if !self.themes[self.active].set_color(token, value)? {
            return Ok(false);
        }
        self.bump_revision();
        Ok(true)
    }
    fn duplicate_source(&self, base: &str, name: &str) -> Result<&ThemeDefinition, String> {
        validate_name(name)?;
        if self.themes.iter().any(|theme| theme.name() == name) {
            return Err(format!(
                "theme name {name:?}: expected a unique library name"
            ));
        }
        self.themes
            .iter()
            .find(|theme| theme.name() == base)
            .ok_or_else(|| format!("theme base {base:?}: expected an existing library theme"))
    }
    fn custom_index(&self, name: &str) -> Result<usize, String> {
        let index = self
            .themes
            .iter()
            .position(|theme| theme.name() == name)
            .ok_or_else(|| format!("theme name {name:?}: expected an existing custom theme"))?;
        if self.themes[index].kind() != &oblikovati_api::types::THEME_CUSTOM {
            return Err(format!(
                "theme name {name:?}: expected a removable custom theme"
            ));
        }
        Ok(index)
    }
    fn bump_revision(&mut self) {
        self.revision = self.revision.saturating_add(1);
    }
}
fn validate_customs(
    customs: &[ThemeDefinition],
    builtins: &[ThemeDefinition],
) -> Result<(), String> {
    let mut names: BTreeSet<&str> = builtins.iter().map(ThemeDefinition::name).collect();
    for custom in customs {
        if custom.kind() != &oblikovati_api::types::THEME_CUSTOM || !names.insert(custom.name()) {
            return Err(format!(
                "loaded theme {:?} ({:?}): expected a uniquely named custom snapshot",
                custom.name(),
                custom.kind()
            ));
        }
    }
    Ok(())
}
