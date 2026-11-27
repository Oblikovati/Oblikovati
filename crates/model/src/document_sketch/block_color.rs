// SPDX-License-Identifier: GPL-2.0-only
//! Independent block definition and placement presentation overrides.
use super::{DocumentSketchId, DocumentSketches, SketchError};
use crate::sketch::SketchEntityId;
use oblikovati_api::types::{Color, OVERRIDE_COLOR_SOURCE};

pub(super) fn validate_block_color(color: Option<&Color>) -> Result<(), SketchError> {
    if let Some(color) = color
        && (color.source != OVERRIDE_COLOR_SOURCE
            || !color.opacity.is_finite()
            || !(0.0..=1.0).contains(&color.opacity))
    {
        return Err(SketchError::Definition(format!(
            "block color {color:?}: expected an override color with finite opacity in [0, 1]"
        )));
    }
    Ok(())
}
impl DocumentSketches {
    /// Set or clear the definition color; example: `sketches.set_block_definition_color("Rib", None)?`.
    pub fn set_block_definition_color(
        &mut self,
        name: &str,
        color: Option<Color>,
    ) -> Result<(), SketchError> {
        validate_block_color(color.as_ref())?;
        let definition = self
            .block_definitions
            .iter_mut()
            .find(|definition| definition.name() == name)
            .ok_or_else(|| {
                SketchError::Definition(format!(
                    "block definition {name:?}: expected an existing name"
                ))
            })?;
        definition.color = color;
        Ok(())
    }
    /// Set or clear one placement color; example: `sketches.set_block_instance_color(sketch, instance, None)?`.
    pub fn set_block_instance_color(
        &mut self,
        sketch: DocumentSketchId,
        instance: SketchEntityId,
        color: Option<Color>,
    ) -> Result<(), SketchError> {
        validate_block_color(color.as_ref())?;
        let index = self.index(sketch)?;
        let item = self.sketches[index].block_instances.iter_mut().find(|item| item.id() == instance).ok_or_else(|| SketchError::Definition(format!("block instance {instance:?} in sketch {sketch:?}: expected an existing instance")))?;
        item.color = color;
        Ok(())
    }
}
