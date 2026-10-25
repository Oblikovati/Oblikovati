// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiTexture;
use std::{collections::BTreeMap, sync::Arc};

const ICON_EDGE: u32 = 40;
const FIRST_ICON_TEXTURE_ID: usize = 100;

struct IconRaster {
    name: &'static str,
    rgba: &'static [u8],
}

include!(concat!(env!("OUT_DIR"), "/icon_rasters.rs"));

#[derive(Default)]
pub(crate) struct GuiIconCache {
    ids: BTreeMap<String, usize>,
    textures: Vec<GuiTexture>,
}

impl GuiIconCache {
    pub(crate) fn texture_id(&mut self, name: &str) -> Option<usize> {
        if let Some(id) = self.ids.get(name) {
            return Some(*id);
        }
        let raster = ICON_RASTERS.iter().find(|raster| raster.name == name)?;
        let id = FIRST_ICON_TEXTURE_ID.checked_add(self.textures.len())?;
        let rgba = icon_pixels(raster)?;
        self.ids.insert(name.into(), id);
        self.textures.push(GuiTexture {
            id,
            extent: [ICON_EDGE; 2],
            rgba: Arc::new(rgba),
        });
        Some(id)
    }

    pub(crate) fn textures(&self) -> &[GuiTexture] {
        &self.textures
    }
}

fn icon_pixels(raster: &IconRaster) -> Option<Vec<[u8; 4]>> {
    let expected = usize::try_from(ICON_EDGE)
        .ok()?
        .checked_pow(2)?
        .checked_mul(4)?;
    if raster.rgba.len() != expected {
        return None;
    }
    Some(raster.rgba.as_chunks::<4>().0.to_vec())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn bundled_icons_are_complete_square_rgba_textures() {
        assert!(ICON_RASTERS.len() >= 200);
        assert!(
            ICON_RASTERS
                .iter()
                .all(|raster| icon_pixels(raster).is_some())
        );
    }

    #[test]
    fn cache_reuses_a_texture_for_the_same_icon() {
        let mut cache = GuiIconCache::default();
        assert_eq!(cache.texture_id("new-part"), cache.texture_id("new-part"));
        assert_eq!(cache.textures().len(), 1);
        assert_eq!(cache.texture_id("missing"), None);
    }

    #[test]
    fn new_part_raster_uses_the_go_dark_icon_roles() {
        let raster = ICON_RASTERS
            .iter()
            .find(|raster| raster.name == "new-part")
            .unwrap();
        let pixels = icon_pixels(raster).unwrap();
        assert!(pixels.iter().any(|pixel| pixel[0] > 200 && pixel[3] > 0));
        assert!(
            pixels
                .iter()
                .any(|pixel| pixel[1] > pixel[0] && pixel[1] > pixel[2] && pixel[3] > 0)
        );
        assert!(pixels.contains(&[0, 0, 0, 31]));
    }
}
