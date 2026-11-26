// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiAtlas;
use std::{collections::BTreeMap, sync::Arc};

/// A portable owned texture identity; example: `GuiTextureKey::from_raw(2)?`.
#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub struct GuiTextureKey(u32);
impl GuiTextureKey {
    pub const FONT: Self = Self(1);
    pub const TAB_PREVIEW: Self = Self(u32::MAX);
    /// Refuse null and pointer-width-dependent values; example: `GuiTextureKey::from_raw(id)?`.
    pub fn from_raw(raw: usize) -> Result<Self, String> {
        let value = u32::try_from(raw)
            .ok()
            .filter(|value| *value != 0)
            .ok_or_else(|| format!("GUI texture {raw}: expected a portable ID in 1..=u32::MAX"))?;
        Ok(Self(value))
    }
    /// Produce an ImGui ID without a native pointer; example: `TextureId::new(key.raw())`.
    pub const fn raw(self) -> usize {
        self.0 as usize
    }
}
/// Font masks and straight-alpha sRGB images have different color validation.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiTextureSource {
    Font(GuiAtlas),
    Color(GuiAtlas),
}
impl GuiTextureSource {
    /// Read immutable encoded pixels; example: `source.pixels().extent`.
    pub fn pixels(&self) -> &GuiAtlas {
        match self {
            Self::Font(pixels) | Self::Color(pixels) => pixels,
        }
    }
}
/// A frame retains its exact resource generation after owner changes or destruction.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiTextureResources(Arc<BTreeMap<GuiTextureKey, GuiTextureSource>>);
impl GuiTextureResources {
    /// Add a color image to an owned frame generation, for example a tab preview.
    pub fn with_color(&self, key: GuiTextureKey, pixels: GuiAtlas) -> Result<Self, String> {
        validate_texture_pixels(key, &pixels, false)?;
        if self.0.contains_key(&key) {
            return Err(format!(
                "GUI texture {}: expected a new color binding",
                key.raw()
            ));
        }
        let mut sources = self.0.as_ref().clone();
        sources.insert(key, GuiTextureSource::Color(pixels));
        Ok(Self(Arc::new(sources)))
    }
    /// Own one font and unique color images; example: `GuiTextureResources::new(font, [(key, icon)])?`.
    pub fn new(
        font: GuiAtlas,
        images: impl IntoIterator<Item = (GuiTextureKey, GuiAtlas)>,
    ) -> Result<Self, String> {
        validate_texture_pixels(GuiTextureKey::FONT, &font, true)?;
        let mut sources = BTreeMap::from([(GuiTextureKey::FONT, GuiTextureSource::Font(font))]);
        for (key, pixels) in images {
            validate_texture_pixels(key, &pixels, false)?;
            if sources
                .insert(key, GuiTextureSource::Color(pixels))
                .is_some()
            {
                return Err(format!(
                    "GUI texture {}: expected one unique binding; font ID 1 is reserved",
                    key.raw()
                ));
            }
        }
        Ok(Self(Arc::new(sources)))
    }
    /// Replace only the font for a new generation; example: `resources.with_font(new_atlas)?`.
    pub fn with_font(&self, font: GuiAtlas) -> Result<Self, String> {
        let images = self.iter().filter_map(|(key, source)| match source {
            GuiTextureSource::Color(pixels) => Some((key, pixels.clone())),
            GuiTextureSource::Font(_) => None,
        });
        Self::new(font, images)
    }
    /// Resolve every draw, including empty batches; example: `resources.binding(raw_id)?`.
    pub fn binding(&self, raw: usize) -> Result<GuiTextureKey, String> {
        let key = GuiTextureKey::from_raw(raw)?;
        self.source(key)?;
        Ok(key)
    }
    /// Read one exact source; example: `resources.source(GuiTextureKey::FONT)?`.
    pub fn source(&self, key: GuiTextureKey) -> Result<&GuiTextureSource, String> {
        self.0.get(&key).ok_or_else(|| {
            format!(
                "GUI texture {}: expected a registered immutable resource",
                key.raw()
            )
        })
    }
    /// Iterate in portable ID order; example: `for (key, source) in resources.iter() { ... }`.
    pub fn iter(&self) -> impl Iterator<Item = (GuiTextureKey, &GuiTextureSource)> {
        self.0.iter().map(|(key, source)| (*key, source))
    }
}
fn validate_texture_pixels(
    key: GuiTextureKey,
    pixels: &GuiAtlas,
    font: bool,
) -> Result<(), String> {
    let [width, height] = pixels.extent;
    if width == 0 || height == 0 || u64::from(width) * u64::from(height) != pixels.rgba.len() as u64
    {
        return Err(format!(
            "GUI texture {} extent {:?}, {} pixels: expected positive dimensions and exactly width*height RGBA pixels",
            key.raw(),
            pixels.extent,
            pixels.rgba.len()
        ));
    }
    if font {
        validate_font_channels(pixels)?;
    }
    Ok(())
}
fn validate_font_channels(pixels: &GuiAtlas) -> Result<(), String> {
    if let Some((index, pixel)) = pixels
        .rgba
        .iter()
        .enumerate()
        .find(|(_, pixel)| pixel[..3] != [255; 3])
    {
        return Err(format!(
            "GUI font atlas pixel {index} {pixel:?}: expected white RGB with alpha coverage"
        ));
    }
    Ok(())
}

#[cfg(test)]
#[path = "../../tests/gui/resources.rs"]
mod tests;
