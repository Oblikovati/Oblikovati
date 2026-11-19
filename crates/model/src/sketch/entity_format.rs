// SPDX-License-Identifier: GPL-2.0-only
//! Independent, inheritable presentation overrides for owned sketch geometry.
use super::{Sketch2D, SketchEntityId, SketchError, SketchRecipe};
use oblikovati_api::types::{OVERRIDE_COLOR_SOURCE, SketchEntityFormat};
use std::collections::BTreeSet;

/// A saved entity override; example: `SketchEntityFormatEntry { entity, format }`.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchEntityFormatEntry {
    pub entity: SketchEntityId,
    pub format: SketchEntityFormat,
}
impl Sketch2D {
    /// Inspect overrides by identity; example: `sketch.entity_format_overrides().get(&line)`.
    pub fn entity_format_overrides(
        &self,
    ) -> &std::collections::BTreeMap<SketchEntityId, SketchEntityFormat> {
        &self.entity_formats
    }
    /// Read the explicit override, or `None` when inherited; example: `sketch.entity_format(line)?`.
    pub fn entity_format(
        &self,
        entity: SketchEntityId,
    ) -> Result<Option<&SketchEntityFormat>, SketchError> {
        self.require_format_entity(entity)?;
        Ok(self.entity_formats.get(&entity))
    }
    /// Set or clear an override atomically; example: `sketch.set_entity_format(line, format)?`.
    pub fn set_entity_format(
        &mut self,
        entity: SketchEntityId,
        mut format: SketchEntityFormat,
    ) -> Result<(), SketchError> {
        self.require_format_entity(entity)?;
        require_format(&format)?;
        if format.override_color.source != OVERRIDE_COLOR_SOURCE {
            format.override_color = Default::default();
        } else if format.override_color.opacity == 0.0 {
            format.override_color.opacity = 0.0;
        }
        if format.line_weight == 0.0 {
            format.line_weight = 0.0;
        }
        if format.line_type.0.is_empty()
            && format.line_weight == 0.0
            && format.override_color.source != OVERRIDE_COLOR_SOURCE
        {
            self.entity_formats.remove(&entity);
        } else {
            self.entity_formats.insert(entity, format);
        }
        Ok(())
    }
    fn require_format_entity(&self, entity: SketchEntityId) -> Result<(), SketchError> {
        if !self.owned_entity_ids().contains(&entity) {
            return Err(SketchError::Definition(format!(
                "format entity {entity:?}: expected an owned sketch entity"
            )));
        }
        Ok(())
    }
    pub(super) fn capture_entity_formats(&self, recipe: &mut SketchRecipe) {
        if self.entity_formats.is_empty() {
            return;
        }
        self.capture_point_ownership_metadata(recipe);
        recipe.entity_formats = Some(
            self.owned_entity_ids()
                .into_iter()
                .filter_map(|entity| {
                    self.entity_formats
                        .get(&entity)
                        .cloned()
                        .map(|format| SketchEntityFormatEntry { entity, format })
                })
                .collect(),
        );
        recipe.version = 21;
    }
    pub(super) fn restore_entity_formats(
        &mut self,
        recipe: &SketchRecipe,
    ) -> Result<(), SketchError> {
        match (recipe.version, &recipe.entity_formats) {
            (1..=20, None) => Ok(()),
            (21, Some(entries)) if !entries.is_empty() => self.adopt_entity_formats(entries),
            shape => Err(SketchError::Definition(format!(
                "entity format recipe {shape:?}: expected no overrides before version 21 or a nonempty override list in version 21"
            ))),
        }
    }
    fn adopt_entity_formats(
        &mut self,
        entries: &[SketchEntityFormatEntry],
    ) -> Result<(), SketchError> {
        let mut seen = BTreeSet::new();
        for entry in entries {
            if !seen.insert(entry.entity) {
                return Err(SketchError::Definition(format!(
                    "entity format {:?}: expected a unique owned entity",
                    entry.entity
                )));
            }
            self.set_entity_format(entry.entity, entry.format.clone())?;
            if !self.entity_formats.contains_key(&entry.entity) {
                return Err(SketchError::Definition(format!(
                    "entity format {:?}: expected a non-default override",
                    entry.entity
                )));
            }
        }
        Ok(())
    }
}
fn require_format(format: &SketchEntityFormat) -> Result<(), SketchError> {
    let line = format.line_type.0.as_ref();
    let color = &format.override_color;
    if ![
        "",
        "continuous",
        "dashed",
        "hidden",
        "center",
        "phantom",
        "custom",
    ]
    .contains(&line)
        || !format.line_weight.is_finite()
        || format.line_weight < 0.0
        || (color.source == OVERRIDE_COLOR_SOURCE
            && (!color.opacity.is_finite() || !(0.0..=1.0).contains(&color.opacity)))
    {
        return Err(SketchError::Definition(format!(
            "entity format {format:?}: expected a known line type, nonnegative finite weight and override opacity in [0, 1]"
        )));
    }
    Ok(())
}
