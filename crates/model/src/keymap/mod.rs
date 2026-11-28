// SPDX-License-Identifier: GPL-2.0-only
//! Ordered command bindings and validated customization overlays.
mod chord;
mod default_aliases;
mod keyboard;
pub use chord::{format_chord, parse_chord};
pub use keyboard::PhysicalKey;
use oblikovati_api::{
    types::KeyChord,
    wire::{BindingInfo, KeymapExport},
};
use std::collections::{BTreeMap, BTreeSet};

/// Desktop keyboard preferences, including the separate default alias choice.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct KeymapPreferences {
    pub delta: KeymapExport,
    pub default_aliases: bool,
}

/// Storage for the user's keymap delta.
pub trait KeymapPreferenceAccess {
    /// Read the loaded delta; example: `store.keymap()`.
    fn keymap(&self) -> KeymapExport;
    /// Store a validated delta; example: `store.set_keymap(&map.export())?`.
    fn set_keymap(&mut self, delta: &KeymapExport) -> Result<(), String>;
    /// Read the desktop preference; for example, enable aliases when `store.default_aliases()`.
    fn default_aliases(&self) -> bool {
        false
    }
    /// Save the bindings and desktop preference together, as in `store.set_preferences(&map.preferences())?`.
    fn set_preferences(&mut self, preferences: &KeymapPreferences) -> Result<(), String> {
        if self.default_aliases() != preferences.default_aliases {
            return Err("keymap storage does not support changing default aliases".into());
        }
        self.set_keymap(&preferences.delta)
    }
}

/// A registered action and its defaults; example: `KeymapAction::builtin("edit.undo", "Undo", "Ctrl+Z", &[])?`.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct KeymapAction {
    id: String,
    name: String,
    kind: String,
    default: String,
    extra: Vec<String>,
}
impl KeymapAction {
    /// Define a built-in action with optional alternate default chords.
    /// Example: `KeymapAction::builtin("edit.redo", "Redo", "Ctrl+Y", &["Ctrl+Shift+Z"])?`.
    pub fn builtin(id: &str, name: &str, default: &str, extra: &[&str]) -> Result<Self, String> {
        Self::new(id, name, "builtin", default, extra)
    }
    /// Define a registered command; example: `KeymapAction::command("draw.line", "Line", "")?`.
    pub fn command(id: &str, name: &str, default: &str) -> Result<Self, String> {
        Self::new(id, name, "command", default, &[])
    }
    fn new(
        id: &str,
        name: &str,
        kind: &str,
        default: &str,
        extra: &[&str],
    ) -> Result<Self, String> {
        if id.is_empty() || name.is_empty() {
            return Err(format!(
                "keymap action id={id:?} name={name:?}: expected nonempty values"
            ));
        }
        let default = format_chord(&parse_chord(default)?);
        let extra = extra
            .iter()
            .map(|value| parse_chord(value).map(|chord| format_chord(&chord)))
            .collect::<Result<_, _>>()?;
        Ok(Self {
            id: id.into(),
            name: name.into(),
            kind: kind.into(),
            default,
            extra,
        })
    }
}

/// Application binding state; it owns no I/O and can be staged before persistence.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Keymap {
    actions: Vec<KeymapAction>,
    chords: BTreeMap<String, String>,
    aliases: BTreeMap<String, String>,
    default_aliases: bool,
}
impl Keymap {
    /// Create the desktop shortcuts; example: `Keymap::desktop_defaults()`.
    pub fn desktop_defaults() -> Self {
        let sketch_commands = [
            ("sketch-line", "Line"),
            ("sketch-rectangle", "Rectangle"),
            ("sketch-circle", "Circle"),
            ("sketch-point", "Point"),
            ("sketch-center-arc", "Center Point Arc"),
            ("sketch-ellipse", "Ellipse"),
            ("sketch-center-rectangle", "Center Rectangle"),
            ("sketch-three-point-rectangle", "Three Point Rectangle"),
            (
                "sketch-three-point-center-rectangle",
                "Three Point Center Rectangle",
            ),
            ("sketch-three-point-circle", "Three Point Circle"),
            ("sketch-three-point-arc", "Three Point Arc"),
            ("constraint-horizontal", "Horizontal"),
            ("constraint-vertical", "Vertical"),
            ("constraint-coincident", "Coincident"),
            ("constraint-fix", "Fix"),
            ("constraint-parallel", "Parallel"),
            ("constraint-perpendicular", "Perpendicular"),
            ("constraint-collinear", "Collinear"),
            ("constraint-equal", "Equal"),
            ("constraint-concentric", "Concentric"),
            ("constraint-tangent", "Tangent"),
            ("dimension", "Dimension"),
        ];
        let actions = [
            ("Create.Extrude", "Extrude", "E", &[][..]),
            ("Create.Revolve", "Revolve", "R", &[][..]),
            ("edit.undo", "Undo", "Ctrl+Z", &[][..]),
            ("edit.redo", "Redo", "Ctrl+Y", &["Ctrl+Shift+Z"][..]),
            ("file.save", "Save", "Ctrl+S", &[][..]),
            ("tool.cancel", "Cancel", "Escape", &[][..]),
            ("view.pan", "Pan", "F2", &[][..]),
            ("view.zoom", "Zoom", "F3", &[][..]),
            ("view.orbit", "Orbit", "F4", &[][..]),
            ("view.previous", "Previous View", "F5", &[][..]),
            ("view.home", "Home View", "F6", &[][..]),
        ];
        Self {
            actions: sketch_commands
                .into_iter()
                .map(|(id, name)| KeymapAction {
                    id: id.into(),
                    name: name.into(),
                    kind: "command".into(),
                    default: String::new(),
                    extra: Vec::new(),
                })
                .chain(
                    actions
                        .into_iter()
                        .map(|(id, name, default, extra)| KeymapAction {
                            id: id.into(),
                            name: name.into(),
                            kind: "builtin".into(),
                            default: default.into(),
                            extra: extra.iter().map(|value| (*value).into()).collect(),
                        }),
                )
                .collect(),
            chords: BTreeMap::new(),
            aliases: BTreeMap::new(),
            default_aliases: false,
        }
    }
    /// Construct a catalog with unique actions and default chords.
    /// Example: `Keymap::new(vec![KeymapAction::builtin("edit.undo", "Undo", "Ctrl+Z", &[])?])?`.
    pub fn new(actions: Vec<KeymapAction>) -> Result<Self, String> {
        let mut ids = BTreeSet::new();
        for action in &actions {
            if !ids.insert(&action.id) {
                return Err(format!(
                    "keymap action {:?}: expected a unique id",
                    action.id
                ));
            }
        }
        let keymap = Self {
            actions,
            chords: BTreeMap::new(),
            aliases: BTreeMap::new(),
            default_aliases: false,
        };
        keymap.check_conflicts()?;
        Ok(keymap)
    }
    /// List effective bindings in registration order; example: `keymap.catalog()`.
    pub fn catalog(&self) -> Vec<BindingInfo> {
        self.actions
            .iter()
            .map(|action| BindingInfo {
                action_id: action.id.clone(),
                display_name: action.name.clone(),
                kind: action.kind.clone(),
                chord: self.effective(action).to_owned(),
                default_chord: action.default.clone(),
                alias: self.aliases.get(&action.id).cloned().unwrap_or_default(),
                customized: self.chords.contains_key(&action.id)
                    || self.aliases.contains_key(&action.id),
            })
            .collect()
    }
    fn effective<'a>(&'a self, action: &'a KeymapAction) -> &'a str {
        self.chords
            .get(&action.id)
            .map_or(action.default.as_str(), String::as_str)
    }
    /// Resolve a canonical key chord to its action; example: `keymap.resolve_chord(&parse_chord("Ctrl+Z")?)`.
    pub fn resolve_chord(&self, chord: &KeyChord) -> Option<&str> {
        let chord = chord::comparison(&format_chord(chord));
        if chord.is_empty() {
            return None;
        }
        self.actions
            .iter()
            .find(|action| chord::comparison(self.effective(action)) == chord)
            .or_else(|| {
                self.actions.iter().find(|action| {
                    !self.chords.contains_key(&action.id)
                        && action
                            .extra
                            .iter()
                            .any(|extra| chord::comparison(extra) == chord)
                })
            })
            .map(|action| action.id.as_str())
    }
    /// Resolve a user alias without case sensitivity; example: `keymap.resolve_alias("LINE")`.
    pub fn resolve_alias(&self, alias: &str) -> Option<&str> {
        if alias.is_empty() {
            return None;
        }
        self.aliases
            .iter()
            .find(|(id, value)| self.has_action(id) && value.eq_ignore_ascii_case(alias))
            .map(|(id, _)| id.as_str())
            .or_else(|| {
                if self.default_aliases {
                    default_aliases::resolve(alias).filter(|id| self.has_action(id))
                } else {
                    None
                }
            })
    }
    /// Check whether an alias can run at sketch Ready; example: `keymap.supports_sketch_alias("edit.undo")`.
    pub fn supports_sketch_alias(&self, id: &str) -> bool {
        self.actions.iter().any(|action| {
            action.id == id
                && (action.kind == "command"
                    || matches!(
                        id,
                        "edit.undo" | "edit.redo" | "file.save" | "view.home" | "view.previous"
                    ))
        })
    }
    /// Replace one shortcut if it is valid and unoccupied; example: `keymap.set_chord("edit.undo", "Ctrl+U")?`.
    pub fn set_chord(&mut self, id: &str, chord: &str) -> Result<(), String> {
        self.require_action(id)?;
        let parsed = parse_chord(chord)?;
        chord::require_supported(&parsed)?;
        if chord::reserved_bare(&parsed) {
            return Err(format!(
                "key chord {chord:?}: expected a modifier on an alphanumeric key"
            ));
        }
        let mut candidate = self.clone();
        candidate.chords.insert(id.into(), format_chord(&parsed));
        candidate.check_conflicts()?;
        *self = candidate;
        Ok(())
    }
    /// Set a user alias; example: `keymap.set_alias("draw.line", "LINE")?`.
    pub fn set_alias(&mut self, id: &str, alias: &str) -> Result<(), String> {
        self.require_action(id)?;
        if alias.chars().count() == 1 {
            return Err(format!(
                "keymap alias {alias:?}: expected empty or at least two characters"
            ));
        }
        if let Some((other, _)) = self.aliases.iter().find(|(other, value)| {
            other.as_str() != id && !alias.is_empty() && value.eq_ignore_ascii_case(alias)
        }) {
            return Err(format!(
                "keymap alias {alias:?}: expected an alias not owned by {other:?}"
            ));
        }
        if alias.is_empty() {
            self.aliases.remove(id);
        } else {
            self.aliases.insert(id.into(), alias.into());
        }
        Ok(())
    }
    /// Restore one action to its defaults; example: `keymap.reset("edit.undo")?`.
    pub fn reset(&mut self, id: &str) -> Result<(), String> {
        self.require_action(id)?;
        let mut candidate = self.clone();
        candidate.chords.remove(id);
        candidate.aliases.remove(id);
        candidate.check_conflicts()?;
        *self = candidate;
        Ok(())
    }
    /// Restore all defaults; example: `keymap.reset_all()`.
    pub fn reset_all(&mut self) {
        self.chords.clear();
        self.aliases.clear();
    }
    /// Restore shortcuts without changing aliases; example: `keymap.reset_all_chords()`.
    pub fn reset_all_chords(&mut self) {
        self.chords.clear();
    }
    /// Copy the public customization delta; example: `let saved = keymap.export()`.
    pub fn export(&self) -> KeymapExport {
        KeymapExport {
            chords: (!self.chords.is_empty()).then(|| self.chords.clone()),
            aliases: (!self.aliases.is_empty()).then(|| self.aliases.clone()),
        }
    }
    /// Copy bindings with desktop settings, as in `store.set_preferences(&keymap.preferences())?`.
    pub fn preferences(&self) -> KeymapPreferences {
        KeymapPreferences {
            delta: self.export(),
            default_aliases: self.default_aliases,
        }
    }
    /// Read the default alias choice, as in `let enabled = keymap.default_aliases()`.
    pub fn default_aliases(&self) -> bool {
        self.default_aliases
    }
    /// Enable default aliases without replacing user bindings, as in `keymap.enable_default_aliases(true)`.
    pub fn enable_default_aliases(&mut self, enabled: bool) {
        self.default_aliases = enabled;
    }
    /// Replace the complete delta atomically; example: `keymap.import(&saved)?`.
    pub fn import(&mut self, delta: &KeymapExport) -> Result<(), String> {
        let mut candidate = Self::new(self.actions.clone())?;
        candidate.default_aliases = self.default_aliases;
        for (id, chord) in delta.chords.iter().flatten() {
            candidate.require_action(id)?;
            let chord = parse_chord(chord)?;
            chord::require_supported(&chord)?;
            if chord::reserved_bare(&chord) {
                return Err(format!(
                    "key chord {:?}: expected a modifier on an alphanumeric key",
                    format_chord(&chord)
                ));
            }
            candidate.chords.insert(id.clone(), format_chord(&chord));
        }
        candidate.check_conflicts()?;
        for (id, alias) in delta.aliases.iter().flatten() {
            candidate.set_alias(id, alias)?;
        }
        *self = candidate;
        Ok(())
    }
    /// Restore known bindings while retaining other versions' entries.
    /// Example: `keymap.restore_saved(&store.keymap())?`.
    pub fn restore_saved(&mut self, delta: &KeymapExport) -> Result<(), String> {
        let known = KeymapExport {
            chords: delta.chords.as_ref().map(|entries| {
                entries
                    .iter()
                    .filter(|(id, _)| self.has_action(id))
                    .map(|(id, value)| (id.clone(), value.clone()))
                    .collect()
            }),
            aliases: delta.aliases.as_ref().map(|entries| {
                entries
                    .iter()
                    .filter(|(id, _)| self.has_action(id))
                    .map(|(id, value)| (id.clone(), value.clone()))
                    .collect()
            }),
        };
        let mut candidate = Self::new(self.actions.clone())?;
        candidate.restore_known(&known)?;
        for (id, value) in delta
            .chords
            .iter()
            .flatten()
            .filter(|(id, _)| !self.has_action(id))
        {
            candidate.chords.insert(id.clone(), value.clone());
        }
        for (id, value) in delta
            .aliases
            .iter()
            .flatten()
            .filter(|(id, _)| !self.has_action(id))
        {
            candidate.aliases.insert(id.clone(), value.clone());
        }
        *self = candidate;
        Ok(())
    }
    fn restore_known(&mut self, known: &KeymapExport) -> Result<(), String> {
        for (id, value) in known.chords.iter().flatten() {
            let chord = parse_chord(value)?;
            chord::require_supported(&chord)?;
            self.chords.insert(id.clone(), format_chord(&chord));
        }
        self.disable_defaults_owned_by_saved_chords();
        self.check_conflicts()?;
        for (id, alias) in known.aliases.iter().flatten() {
            self.set_alias(id, alias)?;
        }
        Ok(())
    }
    fn disable_defaults_owned_by_saved_chords(&mut self) {
        let owned: BTreeSet<_> = self
            .chords
            .values()
            .filter(|chord| !chord.is_empty())
            .map(|chord| chord::comparison(chord))
            .collect();
        for action in &self.actions {
            if self.chords.contains_key(&action.id) {
                continue;
            }
            let defaults = std::iter::once(&action.default).chain(&action.extra);
            if defaults
                .filter(|chord| !chord.is_empty())
                .any(|chord| owned.contains(&chord::comparison(chord)))
            {
                self.chords.insert(action.id.clone(), String::new());
            }
        }
    }
    fn has_action(&self, id: &str) -> bool {
        self.actions.iter().any(|action| action.id == id)
    }
    fn require_action(&self, id: &str) -> Result<(), String> {
        if self.has_action(id) {
            Ok(())
        } else {
            Err(format!(
                "keymap action {id:?}: expected a registered action"
            ))
        }
    }
    fn check_conflicts(&self) -> Result<(), String> {
        let mut seen = BTreeMap::new();
        for action in &self.actions {
            let chords = std::iter::once(self.effective(action)).chain(
                action
                    .extra
                    .iter()
                    .filter(|_| !self.chords.contains_key(&action.id))
                    .map(String::as_str),
            );
            for chord in chords.filter(|chord| !chord.is_empty()) {
                if let Some(previous) = seen.insert(chord::comparison(chord), action.id.as_str())
                    && previous != action.id
                {
                    return Err(format!(
                        "key chord {chord:?}: expected no conflict with action {previous:?}"
                    ));
                }
            }
        }
        Ok(())
    }
}

#[cfg(test)]
#[path = "tests.rs"]
mod tests;
