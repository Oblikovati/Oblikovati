// SPDX-License-Identifier: GPL-2.0-only
use super::IconRoleMasks;
use crate::GuiAtlas;
use std::sync::Arc;
impl IconRoleMasks {
    /// Compose Go's background-to-primary roles as straight-alpha sRGB; example: `masks.compose(colors)?`.
    pub(crate) fn compose(&self, colors: [[f32; 4]; 4]) -> Result<GuiAtlas, String> {
        for (role, color) in colors.iter().enumerate() {
            if color
                .iter()
                .any(|value| !value.is_finite() || !(0. ..=1.).contains(value))
            {
                return Err(format!(
                    "icon role {role} color {color:?}: expected finite RGBA components in 0..=1"
                ));
            }
        }
        let rgba = (0..(self.edge * self.edge) as usize)
            .map(|index| compose_pixel(&self.cover, index, &colors))
            .collect();
        Ok(GuiAtlas {
            extent: [self.edge; 2],
            rgba: Arc::new(rgba),
        })
    }
}
fn compose_pixel(masks: &[Vec<u8>; 4], index: usize, colors: &[[f32; 4]; 4]) -> [u8; 4] {
    let mut accumulated = [0.; 4];
    for (mask, color) in masks.iter().zip(colors) {
        let alpha = f32::from(mask[index]) / 255. * color[3];
        for channel in 0..3 {
            accumulated[channel] = color[channel] * alpha + accumulated[channel] * (1. - alpha);
        }
        accumulated[3] = alpha + accumulated[3] * (1. - alpha);
    }
    let alpha = accumulated[3];
    if alpha == 0. {
        return [0; 4];
    }
    std::array::from_fn(|channel| {
        let value = if channel == 3 {
            alpha
        } else {
            accumulated[channel] / alpha
        };
        (value.clamp(0., 1.) * 255.).round() as u8
    })
}
