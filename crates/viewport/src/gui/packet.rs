// SPDX-License-Identifier: GPL-2.0-only
use std::sync::Arc;
#[derive(Clone, Debug, PartialEq)]
pub struct GuiVertex {
    pub position: [f32; 2],
    pub uv: [f32; 2],
    pub srgb: [u8; 4],
}
#[derive(Clone, Debug, PartialEq)]
pub struct GuiTexture {
    pub id: usize,
    pub extent: [u32; 2],
    pub rgba: Arc<Vec<[u8; 4]>>,
}
#[derive(Clone, Debug, PartialEq)]
pub struct GuiBatch {
    pub texture_id: usize,
    pub vertices: Vec<GuiVertex>,
    pub indices: Vec<u32>,
    pub clip_rect: [f32; 4],
}
#[derive(Clone, Debug, PartialEq)]
pub struct GuiFrame {
    pub extent: [u32; 2],
    pub textures: Vec<GuiTexture>,
    pub batches: Vec<GuiBatch>,
}
#[derive(Clone, Debug, PartialEq)]
pub struct GuiAtlas {
    pub extent: [u32; 2],
    pub rgba: Arc<Vec<[u8; 4]>>,
}
