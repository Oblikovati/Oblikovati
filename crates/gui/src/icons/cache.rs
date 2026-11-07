// SPDX-License-Identifier: GPL-2.0-only
use super::{
    IconRoleMasks,
    catalog::{GLYPHS, IconGlyph},
    rasterize_roles,
};
use crate::{GuiAtlas, GuiFrameRequest, GuiTextureKey, GuiTextureResources};
use oblikovati_api::types::{
    TOKEN_ICON_BACKGROUND, TOKEN_ICON_PRIMARY, TOKEN_ICON_SECONDARY, TOKEN_ICON_TERTIARY,
};
use std::{collections::BTreeMap, sync::Arc};
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum IconSize {
    Small,
    Large,
}
impl IconSize {
    pub(crate) fn index(self) -> usize {
        match self {
            Self::Small => 0,
            Self::Large => 1,
        }
    }
}
#[derive(Clone)]
pub(crate) struct GuiRibbonIcons {
    keys: Vec<[GuiTextureKey; 2]>,
    pub logical: [[f32; 2]; 2],
}
impl GuiRibbonIcons {
    pub(crate) fn key(&self, glyph: IconGlyph, size: IconSize) -> GuiTextureKey {
        self.keys[glyph.index()][size.index()]
    }
    fn new() -> Result<Self, String> {
        let keys = GLYPHS
            .iter()
            .map(|glyph| {
                Ok([
                    GuiTextureKey::from_raw(2 + glyph.index() * 2)?,
                    GuiTextureKey::from_raw(3 + glyph.index() * 2)?,
                ])
            })
            .collect::<Result<Vec<_>, String>>()?;
        Ok(Self {
            keys,
            logical: [[18.; 2], [40.; 2]],
        })
    }
}
#[derive(Clone, Debug, PartialEq)]
struct IconPreparation {
    edges: [u32; 2],
    colors: [[f32; 4]; 4],
}
pub(crate) struct GuiIconCache {
    masks: BTreeMap<(IconGlyph, u32), Arc<IconRoleMasks>>,
    prepared: Option<IconPreparation>,
    defaults: [[f32; 4]; 4],
    pub resources: GuiTextureResources,
    pub presentation: GuiRibbonIcons,
}
impl GuiIconCache {
    pub(crate) fn new(font: GuiAtlas) -> Result<Self, String> {
        Ok(Self {
            masks: BTreeMap::new(),
            prepared: None,
            defaults: role_colors(oblikovati_model::theme::ThemeDefinition::dark()?.palette()),
            resources: GuiTextureResources::new(font, [])?,
            presentation: GuiRibbonIcons::new()?,
        })
    }
    pub(crate) fn prepare(
        &mut self,
        request: &GuiFrameRequest,
        font: &GuiAtlas,
    ) -> Result<bool, String> {
        let preparation = IconPreparation {
            edges: physical_edges(request)?,
            colors: request
                .commands
                .theme
                .as_ref()
                .map_or(self.defaults, |theme| role_colors(theme.palette())),
        };
        self.presentation.logical = preparation
            .edges
            .map(|edge| std::array::from_fn(|axis| edge as f32 / request.layout_scale()[axis]));
        if self.prepared.as_ref() == Some(&preparation) {
            if self.resources.source(GuiTextureKey::FONT)?.pixels() != font {
                self.resources = self.resources.with_font(font.clone())?;
            }
            return Ok(previous_logical != self.presentation.logical);
        }
        let images = self.compose_images(&preparation)?;
        let resources = GuiTextureResources::new(font.clone(), images)?;
        self.resources = resources;
        self.prepared = Some(preparation);
        Ok(true)
    }
    fn compose_images(
        &mut self,
        preparation: &IconPreparation,
    ) -> Result<Vec<(GuiTextureKey, GuiAtlas)>, String> {
        let mut images = Vec::new();
        for glyph in GLYPHS {
            for size in [IconSize::Small, IconSize::Large] {
                let edge = preparation.edges[size.index()];
                let masks = self.masks_for(*glyph, edge)?;
                images.push((
                    self.presentation.key(*glyph, size),
                    masks.compose(preparation.colors)?,
                ));
            }
        }
        Ok(images)
    }
    fn masks_for(&mut self, glyph: IconGlyph, edge: u32) -> Result<Arc<IconRoleMasks>, String> {
        if let Some(masks) = self.masks.get(&(glyph, edge)) {
            return Ok(masks.clone());
        }
        let masks = Arc::new(
            rasterize_roles(glyph.source(), edge)
                .map_err(|cause| format!("icon {:?}: {cause}", glyph.name()))?,
        );
        self.masks.insert((glyph, edge), masks.clone());
        Ok(masks)
    }
}
fn physical_edges(request: &GuiFrameRequest) -> Result<[u32; 2], String> {
    let backing = request.framebuffer_scale.into_iter().fold(0., f32::max);
    let factor = request.commands.ui_scale.unwrap_or_default().icon();
    let edges = [18., 40.].map(|base| (base * factor * backing).round());
    if edges
        .iter()
        .any(|edge| !edge.is_finite() || !(1. ..=512.).contains(edge))
    {
        return Err(format!(
            "icon pixel edges {edges:?}: expected finite pixel sizes in 1..=512"
        ));
    }
    Ok(edges.map(|edge| edge as u32))
}
fn role_colors(palette: &oblikovati_model::theme::ThemePalette) -> [[f32; 4]; 4] {
    let tokens = [
        TOKEN_ICON_BACKGROUND,
        TOKEN_ICON_TERTIARY,
        TOKEN_ICON_SECONDARY,
        TOKEN_ICON_PRIMARY,
    ];
    tokens.map(|token| {
        let color = &palette[&token];
        [color.r, color.g, color.b, color.a]
    })
}

#[cfg(test)]
#[path = "../../tests/icons/cache.rs"]
mod tests;
