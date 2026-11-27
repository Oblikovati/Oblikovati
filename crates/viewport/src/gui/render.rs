// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiAtlas, GuiBatch, GuiFrame, GuiTextureKey, GuiTextureResources, GuiTextureSource};
use martensite_render_core::{
    PrimitiveDraw, RasterImage, RenderClipping, RenderFrame, RenderLayer, RenderVertex,
    TexturedStyle, identity_clip,
};
use std::collections::BTreeMap;
use std::sync::{LazyLock, Mutex};

static SRGB_CHANNELS: LazyLock<[f32; 256]> =
    LazyLock::new(|| std::array::from_fn(|byte| srgb_channel(byte as u8)));
/// Resource pixels are converted once and shared by immutable kernel submissions.
pub struct GuiFrameCompiler {
    resources: GuiTextureResources,
    images: BTreeMap<GuiTextureKey, RasterImage>,
    prepared: Mutex<PreparedGuiLayer>,
}

#[derive(Default)]
struct PreparedGuiLayer {
    extent: [u32; 2],
    batches: Vec<PreparedGuiBatch>,
}

struct PreparedGuiBatch {
    source: GuiBatch,
    draw: PrimitiveDraw,
}
impl GuiFrameCompiler {
    /// Prepare the font resource independently of the UI context; example: `GuiFrameCompiler::new(&atlas)?`.
    pub fn new(atlas: &GuiAtlas) -> Result<Self, String> {
        Self::with_resources(&GuiTextureResources::new(atlas.clone(), [])?)
    }
    /// Prepare the exact frame resources once; example: `GuiFrameCompiler::with_resources(&frame.resources)?`.
    pub fn with_resources(resources: &GuiTextureResources) -> Result<Self, String> {
        let images = resources
            .iter()
            .map(|(key, source)| source_image(source).map(|image| (key, image)))
            .collect::<Result<BTreeMap<_, _>, _>>()?;
        Ok(Self {
            resources: resources.clone(),
            images,
            prepared: Mutex::default(),
        })
    }
    /// Compile a standalone review frame; example: `compiler.compile(&snapshot)?`.
    pub fn compile(&self, frame: &GuiFrame) -> Result<RenderFrame, String> {
        let layer = self.layer(frame)?;
        RenderFrame::with_layers(
            frame.extent[0],
            frame.extent[1],
            [0., 0., 0., 1.],
            vec![layer],
        )
        .map_err(|cause| cause.to_string())
    }
    /// Prepare a depth-independent layer for shared composition; example: `compiler.layer(&snapshot)?`.
    pub fn layer(&self, frame: &GuiFrame) -> Result<RenderLayer, String> {
        if self.resources != frame.resources {
            return Err(
                "GUI frame resources: expected a compiler for this immutable resource generation"
                    .into(),
            );
        }
        let projection = pixel_projection(frame.extent)?;
        let mut prepared = self
            .prepared
            .lock()
            .map_err(|_| "GUI frame preparation: expected a live draw cache")?;
        if prepared.extent != frame.extent {
            prepared.batches.clear();
            prepared.extent = frame.extent;
        }
        let mut draws = Vec::with_capacity(frame.batches.len());
        for batch in &frame.batches {
            if batch.indices.is_empty() {
                self.image(batch.texture)?;
                continue;
            }
            draws.push(self.prepare_draw(batch, projection, &mut prepared.batches, draws.len())?);
        }
        prepared.batches.truncate(draws.len());
        Ok(RenderLayer::new(draws, false, false, false))
    }
    fn prepare_draw(
        &self,
        batch: &GuiBatch,
        projection: [f32; 16],
        prepared: &mut Vec<PreparedGuiBatch>,
        index: usize,
    ) -> Result<PrimitiveDraw, String> {
        let previous = prepared.get_mut(index);
        if let Some(slot) = &previous
            && slot.source == *batch
        {
            return Ok(slot.draw.clone());
        }
        let draw = compile_batch(batch, self.image(batch.texture)?, projection)?;
        let next = PreparedGuiBatch {
            source: batch.clone(),
            draw: draw.clone(),
        };
        if let Some(slot) = previous {
            *slot = next;
        } else {
            prepared.push(next);
        }
        Ok(draw)
    }
    fn image(&self, key: GuiTextureKey) -> Result<RasterImage, String> {
        self.images
            .get(&key)
            .cloned()
            .ok_or_else(|| format!("GUI texture {}: expected a compiled resource", key.raw()))
    }
}
/// Lower a complete owned GUI frame before any GPU work.
/// Example: `compile_gui_frame(&snapshot, &atlas)?`; retain a compiler for repeated frames.
pub fn compile_gui_frame(frame: &GuiFrame, atlas: &GuiAtlas) -> Result<RenderFrame, String> {
    let supplied = GuiTextureResources::new(atlas.clone(), [])?;
    if supplied.source(GuiTextureKey::FONT)? != frame.resources.source(GuiTextureKey::FONT)? {
        return Err("GUI supplied font: expected the immutable font retained by this frame".into());
    }
    GuiFrameCompiler::with_resources(&frame.resources)?.compile(frame)
}
fn source_image(source: &GuiTextureSource) -> Result<RasterImage, String> {
    let atlas = source.pixels();
    let pixels = atlas
        .rgba
        .iter()
        .map(|pixel| vertex_color(*pixel))
        .collect();
    RasterImage::new(atlas.extent, pixels).map_err(|cause| cause.to_string())
}
fn compile_batch(
    batch: &GuiBatch,
    image: RasterImage,
    projection: [f32; 16],
) -> Result<PrimitiveDraw, String> {
    let vertices = kernel_vertices(batch);
    let style = batch_texture(batch, image)?;
    let clipping = clip_rectangle(batch.clip_rect)?;
    PrimitiveDraw::textured(vertices, batch.indices.clone(), projection, style)
        .map(|draw| draw.with_clipping(clipping))
        .map_err(|cause| cause.to_string())
}
fn kernel_vertices(batch: &GuiBatch) -> Vec<RenderVertex> {
    batch
        .vertices
        .iter()
        .map(|vertex| {
            RenderVertex::new(
                [vertex.position[0], vertex.position[1], 0.],
                vertex_color(vertex.srgb),
            )
        })
        .collect()
}
fn batch_texture(batch: &GuiBatch, image: RasterImage) -> Result<TexturedStyle, String> {
    let uv = batch.vertices.iter().map(|vertex| vertex.uv).collect();
    TexturedStyle::new(image, uv, 0.).map_err(|cause| cause.to_string())
}
fn vertex_color([red, green, blue, alpha]: [u8; 4]) -> [f32; 4] {
    let channels = &*SRGB_CHANNELS;
    [
        channels[usize::from(red)],
        channels[usize::from(green)],
        channels[usize::from(blue)],
        f32::from(alpha) / 255.,
    ]
}
fn srgb_channel(byte: u8) -> f32 {
    let encoded = f32::from(byte) / 255.;
    if encoded <= 0.04045 {
        return encoded / 12.92;
    }
    ((encoded + 0.055) / 1.055).powf(2.4)
}
fn pixel_projection([width, height]: [u32; 2]) -> Result<[f32; 16], String> {
    if width == 0 || height == 0 {
        return Err(format!(
            "GUI extent {width}x{height}: expected positive drawable dimensions"
        ));
    }
    let x = 2. / width as f32;
    let y = -2. / height as f32;
    Ok([
        x, 0., 0., 0., 0., y, 0., 0., 0., 0., 1., 0., -1., 1., 0., 1.,
    ])
}
fn clip_rectangle(rect: [f32; 4]) -> Result<RenderClipping, String> {
    validate_clip_rectangle(rect)?;
    let [left, top, right, bottom] = rect;
    let planes = [
        [1., 0., 0., -left],
        [-1., 0., 0., right],
        [0., 1., 0., -top],
        [0., -1., 0., bottom],
    ];
    // Kernel clip chains are intersected unions: each rectangle side is its own chain.
    RenderClipping::new(
        planes.into_iter().map(|plane| vec![plane]).collect(),
        identity_clip(),
    )
    .map_err(|cause| cause.to_string())
}
fn validate_clip_rectangle(rect: [f32; 4]) -> Result<(), String> {
    let [left, top, right, bottom] = rect;
    if !rect.iter().all(|value| value.is_finite()) || left > right || top > bottom {
        return Err(format!(
            "GUI clip {rect:?}: expected finite ordered rectangle"
        ));
    }
    Ok(())
}
