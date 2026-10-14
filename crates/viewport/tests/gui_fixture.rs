// SPDX-License-Identifier: GPL-2.0-only
use super::{GuiAtlas, GuiBatch, GuiFrame, GuiVertex};
use std::sync::Arc;
/// Use a solid font texel to isolate composition; example: `atlas()` for a compiled quad.
pub fn atlas() -> GuiAtlas {
    GuiAtlas {
        extent: [1, 1],
        rgba: Arc::new(vec![[255; 4]]),
    }
}
/// Cover the target with a clipped colored quad; example: `frame([64,48], [255;4], [0.,0.,64.,48.])`.
pub fn frame(extent: [u32; 2], color: [u8; 4], clip_rect: [f32; 4]) -> GuiFrame {
    let [width, height] = extent.map(|value| value as f32);
    let vertices = [[0., 0.], [width, 0.], [width, height], [0., height]]
        .map(|position| GuiVertex {
            position,
            uv: [0.5; 2],
            srgb: color,
        })
        .into();
    GuiFrame {
        extent,
        batches: vec![GuiBatch {
            vertices,
            indices: vec![0, 1, 2, 0, 2, 3],
            clip_rect,
        }],
    }
}
