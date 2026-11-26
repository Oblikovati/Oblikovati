// SPDX-License-Identifier: GPL-2.0-only
//! Named Sweep settings in the desktop preference profile.
use super::{GuiSweepEditSeed, GuiSweepOrientation};
use imgui::Ui;
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

const NO_PRESET: &str = "No Preset";
const MAX_SETTINGS_BYTES: usize = 1024 * 1024;

/// Stores the Sweep settings for one desktop preference profile.
pub trait GuiSweepPresetStore {
    /// Read the current preset file. Example: `store.load()?`.
    fn load(&mut self) -> Result<Option<Vec<u8>>, String>;
    /// Atomically replace the preset file. Example: `store.save(&bytes)?`.
    fn save(&mut self, bytes: &[u8]) -> Result<(), String>;
}

#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct Recipe {
    version: u32,
    entries: BTreeMap<String, GuiSweepEditSeed>,
}

#[derive(Default)]
pub(super) struct SweepPresets {
    entries: BTreeMap<String, GuiSweepEditSeed>,
    store: Option<Box<dyn GuiSweepPresetStore>>,
    selected: Option<String>,
    error: Option<String>,
    pub(super) needs_frame: bool,
}

impl SweepPresets {
    pub(super) fn set_store(
        &mut self,
        mut store: Box<dyn GuiSweepPresetStore>,
    ) -> Result<(), String> {
        let entries = store
            .load()?
            .map(|bytes| decode(&bytes))
            .transpose()?
            .unwrap_or_default();
        self.entries = entries;
        self.store = Some(store);
        self.reset();
        Ok(())
    }

    pub(super) fn reset(&mut self) {
        self.selected = None;
        self.error = None;
        self.needs_frame = false;
    }

    pub(super) fn draw(
        &mut self,
        ui: &Ui,
        certified: Option<GuiSweepEditSeed>,
    ) -> Option<GuiSweepEditSeed> {
        self.needs_frame = false;
        ui.set_next_item_width(-35.);
        let change = self.choose(ui);
        ui.same_line();
        let disabled = ui.begin_disabled(certified.is_none());
        let save = crate::widget_abi::button(ui, "+##save-sweep-preset", [28., 0.]);
        if ui.is_item_hovered() {
            ui.tooltip_text("Save Sweep preset");
        }
        drop(disabled);
        if save && let Some(fields) = certified {
            self.error = self.save(fields).err();
            self.needs_frame = true;
        }
        if let Some(error) = &self.error {
            ui.text_wrapped(error);
        }
        change
    }

    fn choose(&mut self, ui: &Ui) -> Option<GuiSweepEditSeed> {
        let label = self.selected.as_deref().unwrap_or(NO_PRESET);
        let _menu = ui.begin_combo("##sweep-preset", label)?;
        if ui
            .selectable_config(NO_PRESET)
            .selected(self.selected.is_none())
            .build()
        {
            self.selected = None;
            self.needs_frame = true;
        }
        for (name, fields) in &self.entries {
            if ui
                .selectable_config(name)
                .selected(self.selected.as_ref() == Some(name))
                .build()
            {
                self.selected = Some(name.clone());
                self.needs_frame = true;
                return Some(fields.clone());
            }
        }
        None
    }

    fn save(&mut self, mut fields: GuiSweepEditSeed) -> Result<(), String> {
        fields.taper = fields.taper.trim().to_owned();
        fields.twist = fields.twist.trim().to_owned();
        let name = name_of(&fields).ok_or("Sweep preset requires valid orientation settings")?;
        if let Some(saved) = self.entries.get(&name) {
            if saved != &fields {
                return Err(format!(
                    "Sweep preset {name:?} already uses different expressions"
                ));
            }
        } else {
            let mut entries = self.entries.clone();
            entries.insert(name.clone(), fields);
            if let Some(store) = &mut self.store {
                let recipe = Recipe {
                    version: 1,
                    entries,
                };
                let bytes = serde_json::to_vec(&recipe).map_err(|cause| cause.to_string())?;
                require_size(&bytes)?;
                store.save(&bytes)?;
                entries = recipe.entries;
            }
            self.entries = entries;
        }
        self.selected = Some(name);
        Ok(())
    }
}

fn decode(bytes: &[u8]) -> Result<BTreeMap<String, GuiSweepEditSeed>, String> {
    require_size(bytes)?;
    let recipe: Recipe = serde_json::from_slice(bytes)
        .map_err(|cause| format!("Sweep presets: expected versioned settings: {cause}"))?;
    if recipe.version != 1
        || recipe
            .entries
            .iter()
            .any(|(name, fields)| name_of(fields).as_ref() != Some(name))
    {
        return Err("Sweep presets: expected version 1 with matching names and settings".into());
    }
    Ok(recipe.entries)
}

fn require_size(bytes: &[u8]) -> Result<(), String> {
    if bytes.len() > MAX_SETTINGS_BYTES {
        return Err(format!(
            "Sweep preset settings: expected at most {MAX_SETTINGS_BYTES} bytes"
        ));
    }
    Ok(())
}

fn name_of(fields: &GuiSweepEditSeed) -> Option<String> {
    match fields.orientation {
        GuiSweepOrientation::Fixed if fields.taper.is_empty() && fields.twist.is_empty() => {
            Some("Fixed".into())
        }
        GuiSweepOrientation::FollowPath
            if !fields.taper.trim().is_empty() && !fields.twist.trim().is_empty() =>
        {
            Some(format!(
                "Follow Path_{}_{}",
                fields.taper.trim(),
                fields.twist.trim()
            ))
        }
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::{
        cell::{Cell, RefCell},
        rc::Rc,
    };

    #[derive(Clone, Default)]
    struct MemoryStore {
        bytes: Rc<RefCell<Option<Vec<u8>>>>,
        reject_save: Rc<Cell<bool>>,
    }
    impl GuiSweepPresetStore for MemoryStore {
        fn load(&mut self) -> Result<Option<Vec<u8>>, String> {
            Ok(self.bytes.borrow().clone())
        }
        fn save(&mut self, bytes: &[u8]) -> Result<(), String> {
            if self.reject_save.get() {
                return Err("expected rejected test write".into());
            }
            self.bytes.replace(Some(bytes.to_vec()));
            Ok(())
        }
    }

    #[test]
    fn sweep_preset_reloads_and_failed_save_keeps_the_original() {
        let store = MemoryStore::default();
        let fields = GuiSweepEditSeed {
            orientation: GuiSweepOrientation::FollowPath,
            taper: "0 deg".into(),
            twist: "90 deg".into(),
            join: false,
        };
        let name = "Follow Path_0 deg_90 deg";
        let mut first = SweepPresets::default();
        first.set_store(Box::new(store.clone())).unwrap();
        first.save(fields.clone()).unwrap();
        assert_eq!(first.selected.as_deref(), Some(name));

        let mut reloaded = SweepPresets::default();
        reloaded.set_store(Box::new(store.clone())).unwrap();
        assert_eq!(reloaded.entries.get(name), Some(&fields));
        assert!(reloaded.selected.is_none());
        store.reject_save.set(true);
        let refused = GuiSweepEditSeed {
            twist: "45 deg".into(),
            ..fields
        };
        assert!(reloaded.save(refused).is_err());
        assert_eq!(reloaded.entries, first.entries);
        assert!(reloaded.selected.is_none());
    }
}
