// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiAtlas, GuiBatch, GuiFrame};
use martensite_vulkan::{
    PrimitiveDraw, RasterImage, RenderClipping, RenderFrame, RenderLayer, RenderVertex,
    TexturedStyle, identity_clip,
};
/// The atlas is converted once and shared by all immutable kernel submissions.
pub struct GuiFrameCompiler {
    image: RasterImage,
}
impl GuiFrameCompiler {
    /// Prepare the font resource independently of the UI context; example: `GuiFrameCompiler::new(&atlas)?`.
    pub fn new(atlas: &GuiAtlas) -> Result<Self, String> {
        Ok(Self {
            image: font_image(atlas)?,
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
        let projection = pixel_projection(frame.extent)?;
        let draws = frame
            .batches
            .iter()
            .filter(|batch| !batch.indices.is_empty())
            .map(|batch| {
                compile_batch(
                    batch,
                    texture_image(batch.texture_id, frame, &self.image)?,
                    projection,
                )
            })
            .collect::<Result<Vec<_>, _>>()?;
        Ok(RenderLayer::new(draws, false, false, false))
    }
}
/// Lower a complete owned GUI frame before any GPU work.
/// Example: `compile_gui_frame(&snapshot, &atlas)?`; retain a compiler for repeated frames.
pub fn compile_gui_frame(frame: &GuiFrame, atlas: &GuiAtlas) -> Result<RenderFrame, String> {
    GuiFrameCompiler::new(atlas)?.compile(frame)
}
fn font_image(atlas: &GuiAtlas) -> Result<RasterImage, String> {
    validate_font_pixels(atlas)?;
    let pixels = atlas
        .rgba
        .iter()
        .map(|pixel| pixel.map(|byte| f32::from(byte) / 255.))
        .collect();
    RasterImage::new(atlas.extent, pixels).map_err(|cause| cause.to_string())
}
fn validate_font_pixels(atlas: &GuiAtlas) -> Result<(), String> {
    if let Some((index, pixel)) = atlas
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
fn texture_image(id: usize, frame: &GuiFrame, font: &RasterImage) -> Result<RasterImage, String> {
    if id == 1 {
        return Ok(font.clone());
    }
    let texture = frame
        .textures
        .iter()
        .find(|texture| texture.id == id)
        .ok_or_else(|| format!("GUI texture {id}: expected an owned frame texture"))?;
    texture_image_pixels(texture)
}

fn texture_image_pixels(texture: &crate::GuiTexture) -> Result<RasterImage, String> {
    let expected = usize::try_from(texture.extent[0])
        .ok()
        .and_then(|width| {
            usize::try_from(texture.extent[1])
                .ok()
                .and_then(|height| width.checked_mul(height))
        })
        .ok_or_else(|| {
            format!(
                "GUI texture {} extent {:?}: expected addressable pixels",
                texture.id, texture.extent
            )
        })?;
    if texture.id == 1 || expected != texture.rgba.len() {
        return Err(format!(
            "GUI texture {} extent {:?} pixels {}: expected non-font texture with {} RGBA pixels",
            texture.id,
            texture.extent,
            texture.rgba.len(),
            expected
        ));
    }
    let pixels = texture
        .rgba
        .iter()
        .map(|pixel| pixel.map(|byte| f32::from(byte) / 255.))
        .collect();
    RasterImage::new(texture.extent, pixels).map_err(|cause| cause.to_string())
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
    [
        srgb_channel(red),
        srgb_channel(green),
        srgb_channel(blue),
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
