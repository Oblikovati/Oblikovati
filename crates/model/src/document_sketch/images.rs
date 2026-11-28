// SPDX-License-Identifier: GPL-2.0-only
//! Placed raster images owned by a 2D sketch (sketch recipe version 18).
use super::*;
use crate::sketch::SketchEntityId;

/// An inserted raster image anchored in sketch coordinates (cm) with finite positive dimensions.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchImage {
    pub id: SketchEntityId,
    pub reference: String,
    pub anchor: [f64; 2],
    pub width: f64,
    pub height: f64,
    #[serde(default)]
    pub rotation: f64,
    #[serde(default = "default_opacity")]
    pub opacity: f64,
}

fn default_opacity() -> f64 {
    1.0
}

impl SketchImage {
    /// Validate image reference, coordinates, dimensions, rotation, and opacity.
    pub fn validate(&self) -> Result<(), SketchError> {
        if self.id.0 == 0 {
            return Err(SketchError::Definition(
                "sketch image ID 0: expected a nonzero entity identity".into(),
            ));
        }
        if self.reference.trim().is_empty() || self.reference.contains('\0') {
            return Err(SketchError::Definition(
                "sketch image reference: expected a non-empty path without NUL".into(),
            ));
        }
        if !self.anchor[0].is_finite() || !self.anchor[1].is_finite() {
            return Err(SketchError::Definition(
                "sketch image anchor: expected finite coordinates".into(),
            ));
        }
        if !self.width.is_finite()
            || self.width <= 0.0
            || !self.height.is_finite()
            || self.height <= 0.0
        {
            return Err(SketchError::Definition(
                "sketch image size: expected finite positive width and height".into(),
            ));
        }
        if !self.rotation.is_finite() {
            return Err(SketchError::Definition(
                "sketch image rotation: expected a finite angle".into(),
            ));
        }
        if !self.opacity.is_finite() || self.opacity <= 0.0 || self.opacity > 1.0 {
            return Err(SketchError::Definition(
                "sketch image opacity: expected a finite value in (0, 1]".into(),
            ));
        }
        Ok(())
    }
}

impl PlacedSketch {
    /// Inspect inserted raster images on this sketch in authored order.
    pub fn images(&self) -> &[SketchImage] {
        &self.images
    }
}

impl DocumentSketches {
    /// Insert a validated raster image into an existing sketch with a globally unique entity ID.
    pub fn add_image(
        &mut self,
        sketch: DocumentSketchId,
        image: SketchImage,
    ) -> Result<(), SketchError> {
        image.validate()?;
        if self.entity_ids().contains(&image.id.0) {
            return Err(SketchError::Definition(format!(
                "sketch image entity ID {}: expected a unique entity identity",
                image.id.0
            )));
        }
        let index = self.index(sketch)?;
        self.sketches[index].images.push(image);
        Ok(())
    }

    pub(super) fn restore_images(
        &mut self,
        sketch: &PlacedSketchRecipe,
        version: u32,
    ) -> Result<(), SketchError> {
        if version < 18 && !sketch.images.is_empty() {
            return Err(SketchError::Definition(format!(
                "sketch images in recipe version {version}: expected version 18"
            )));
        }
        for image in &sketch.images {
            self.add_image(sketch.id, image.clone())?;
        }
        Ok(())
    }
}
