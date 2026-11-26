// SPDX-License-Identifier: GPL-2.0-only
//! Keep a filtered thumbnail for each window and bind one to the GUI frame.
use super::PreviewKey;
use crate::{GuiAtlas, GuiTextureKey, GuiTextureResources};
use oblikovati_viewport::ViewportPreviewPixels;
use std::{collections::BTreeMap, sync::Arc};

const MAX_EXTENT: [usize; 2] = [384, 216];

#[derive(Default)]
pub(super) struct PreviewTexture {
    pub(super) pictures: BTreeMap<PreviewKey, GuiAtlas>,
    bound: Option<(PreviewKey, GuiAtlas)>,
    resources: Option<TextureGeneration>,
}

struct TextureGeneration {
    base: GuiTextureResources,
    atlas: GuiAtlas,
    combined: GuiTextureResources,
}

impl PreviewTexture {
    pub(super) fn note(
        &mut self,
        key: PreviewKey,
        extent: [u32; 2],
        pixels: ViewportPreviewPixels<'_>,
        minimum: [f32; 2],
        maximum: [f32; 2],
    ) {
        self.pictures
            .insert(key, filtered_picture(extent, pixels, minimum, maximum));
    }

    pub(super) fn retain(&mut self, open: impl Fn(PreviewKey) -> bool) {
        self.pictures.retain(|&key, _| open(key));
        if self.bound.as_ref().is_some_and(|(key, _)| !open(*key)) {
            self.bound = None;
            self.resources = None;
        }
    }

    pub(super) fn bind(&mut self, key: PreviewKey) -> bool {
        let Some(picture) = self.pictures.get(&key) else {
            return false;
        };
        self.bound = Some((key, picture.clone()));
        true
    }

    pub(super) fn resources(
        &mut self,
        base: &GuiTextureResources,
    ) -> Result<GuiTextureResources, String> {
        let Some((_, atlas)) = &self.bound else {
            return Ok(base.clone());
        };
        if let Some(generation) = &self.resources
            && Arc::ptr_eq(&generation.atlas.rgba, &atlas.rgba)
            && &generation.base == base
        {
            return Ok(generation.combined.clone());
        }
        let combined = base.with_color(GuiTextureKey::TAB_PREVIEW, atlas.clone())?;
        self.resources = Some(TextureGeneration {
            base: base.clone(),
            atlas: atlas.clone(),
            combined: combined.clone(),
        });
        Ok(combined)
    }
}

// The caller has already checked that the crop is nonempty and backed by a complete frame.
fn filtered_picture(
    extent: [u32; 2],
    pixels: ViewportPreviewPixels<'_>,
    minimum: [f32; 2],
    maximum: [f32; 2],
) -> GuiAtlas {
    let [width, height] = extent.map(|side| side as usize);
    let [left, top] = minimum.map(|edge| edge as usize);
    let [right, bottom] = [
        (maximum[0] as usize).min(width),
        (maximum[1] as usize).min(height),
    ];
    let [columns, rows] = [
        (right - left).min(MAX_EXTENT[0]),
        (bottom - top).min(MAX_EXTENT[1]),
    ];
    let x_taps = source_taps(left, right - left, columns);
    let y_taps = source_taps(top, bottom - top, rows);
    let mut rgba = Vec::with_capacity(columns * rows);
    for [top, bottom] in y_taps {
        for &[left, right] in &x_taps {
            let indices = [
                top * width + left,
                top * width + right,
                bottom * width + left,
                bottom * width + right,
            ];
            rgba.push(average_pixel(pixels, indices));
        }
    }
    GuiAtlas {
        extent: [columns as u32, rows as u32],
        rgba: Arc::new(rgba),
    }
}

fn source_taps(start: usize, span: usize, count: usize) -> Vec<[usize; 2]> {
    (0..count)
        .map(|index| {
            // Quarter and three-quarter samples preserve coverage along downsampled edges.
            [1, 3].map(|quarter| start + (index * 4 + quarter) * span / (count * 4))
        })
        .collect()
}

fn average_pixel(pixels: ViewportPreviewPixels<'_>, indices: [usize; 4]) -> [u8; 4] {
    let colors = match pixels {
        ViewportPreviewPixels::Rgb(rgb) => indices.map(|index| rgb[index]),
        ViewportPreviewPixels::Rgba(rgba) => indices.map(|index| {
            let [r, g, b, _] = rgba[index];
            [r, g, b]
        }),
    };
    let [r, g, b] = std::array::from_fn(|channel| {
        let total: u16 = colors.iter().map(|color| u16::from(color[channel])).sum();
        ((total + 2) / 4) as u8
    });
    [r, g, b, 255]
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn captures_two_backing_pixels_per_logical_pixel_and_filters_edges() {
        let input: Vec<_> = (0..768 * 432)
            .map(|i| [if i % 2 == 0 { 0 } else { 255 }, 0, 0])
            .collect();
        let atlas = filtered_picture(
            [768, 432],
            ViewportPreviewPixels::Rgb(&input),
            [0.; 2],
            [768., 432.],
        );
        assert_eq!(atlas.extent, [384, 216]);
        assert!(atlas.rgba.iter().all(|&pixel| pixel == [128, 0, 0, 255]));
    }

    #[test]
    fn switching_and_closing_windows_preserves_earlier_frame_texture_generations() {
        let base = GuiTextureResources::new(
            GuiAtlas {
                extent: [1, 1],
                rgba: Arc::new(vec![[255; 4]]),
            },
            [],
        )
        .unwrap();
        let mut previews = PreviewTexture::default();
        let mut frames = Vec::new();
        for (key, pixel) in [
            (PreviewKey::Window(1), [1, 2, 3]),
            (PreviewKey::Home, [4, 5, 6]),
        ] {
            previews.note(
                key,
                [1, 1],
                ViewportPreviewPixels::Rgb(&[pixel]),
                [0.; 2],
                [1.; 2],
            );
            assert!(previews.bind(key));
            frames.push(previews.resources(&base).unwrap());
        }
        previews.retain(|_| false);
        assert!(!previews.bind(PreviewKey::Window(1)));
        assert_eq!(previews.resources(&base).unwrap(), base);
        let pixel = |index: usize| {
            frames[index]
                .source(GuiTextureKey::TAB_PREVIEW)
                .unwrap()
                .pixels()
                .rgba[0]
        };
        assert_eq!(pixel(0), [1, 2, 3, 255]);
        assert_eq!(pixel(1), [4, 5, 6, 255]);
    }
}
