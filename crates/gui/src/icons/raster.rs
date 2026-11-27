// SPDX-License-Identifier: GPL-2.0-only
//! The SVG renderer is contained here; output is coverage, not premultiplied color.
use super::svg_roles::role_document;
use image::{
    RgbaImage,
    imageops::{self, FilterType},
};
use resvg::{tiny_skia, usvg};
use std::{fmt::Write, sync::Arc};

pub(super) fn rasterize_identity(
    source: &str,
    edge: u32,
    mode: usize,
) -> Result<crate::GuiAtlas, String> {
    rasterize_identity_at_size(source, [edge; 2], mode)
}

pub(super) fn rasterize_identity_at_size(
    source: &str,
    extent: [u32; 2],
    mode: usize,
) -> Result<crate::GuiAtlas, String> {
    let mut resolved = resolve_identity_paints(source, mode)?;
    apply_identity_classes(&mut resolved, mode)?;
    let pixels = render_at_size(resolved.as_bytes(), extent)?;
    let rgba = pixels
        .pixels()
        .iter()
        .map(|pixel| {
            let pixel = pixel.demultiply();
            [pixel.red(), pixel.green(), pixel.blue(), pixel.alpha()]
        })
        .collect();
    Ok(crate::GuiAtlas {
        extent,
        rgba: Arc::new(rgba),
    })
}

fn resolve_identity_paints(mut source: &str, mode: usize) -> Result<String, String> {
    use super::identity_paints::PAINTS;
    let mut resolved = String::with_capacity(source.len());
    while let Some((before, paint)) = source.split_once("var(--") {
        let (name, rest) = paint
            .split_once(',')
            .ok_or("icon paint: expected a fallback")?;
        let end = rest
            .find(['"', '\''])
            .and_then(|quote| rest[..quote].rfind(')'))
            .ok_or("icon paint: expected a closing delimiter")?;
        let color = PAINTS
            .iter()
            .find(|(key, _)| *key == name)
            .ok_or_else(|| format!("identity icon paint {name:?}: expected a design token"))?;
        resolved.push_str(before);
        resolved.push_str(color.1[mode]);
        source = &rest[end + 1..];
    }
    resolved.push_str(source);
    Ok(resolved)
}
fn apply_identity_classes(svg: &mut String, mode: usize) -> Result<(), String> {
    if !svg.contains("class=") {
        return Ok(());
    }
    let mut rules = String::from("<style>");
    for (token, colors) in super::identity_paints::PAINTS {
        for (prefix, property) in [('f', "fill"), ('s', "stroke")] {
            let class = format!("{prefix}-{token}");
            if svg.contains(&class) {
                write!(rules, ".{class}{{{property}:{}}}", colors[mode])
                    .map_err(|cause| cause.to_string())?;
            }
        }
    }
    rules.push_str("</style>");
    let root = svg
        .find("<svg")
        .and_then(|start| svg[start..].find('>').map(|end| start + end + 1))
        .ok_or("identity icon classes: expected an SVG root")?;
    svg.insert_str(root, &rules);
    Ok(())
}
#[derive(Clone, Debug)]
pub(crate) struct IconRoleMasks {
    pub edge: u32,
    pub cover: [Vec<u8>; 4],
}
pub(crate) fn rasterize_roles(source: &[u8], edge: u32) -> Result<IconRoleMasks, String> {
    if !(1..=512).contains(&edge) {
        return Err(format!(
            "icon edge {edge}: expected a pixel size in 1..=512"
        ));
    }
    let full = render(&role_document(source, None)?, edge * 4)?;
    let bounds = content_bounds(&full)?;
    let mut cover: [Vec<u8>; 4] = Default::default();
    for (role, mask) in cover.iter_mut().enumerate() {
        let pixels = render(&role_document(source, Some(role))?, edge * 4)?;
        *mask = normalized_alpha(&pixels, bounds, edge);
    }
    Ok(IconRoleMasks { edge, cover })
}
fn render(source: &[u8], edge: u32) -> Result<tiny_skia::Pixmap, String> {
    render_at_size(source, [edge; 2])
}

fn render_at_size(source: &[u8], [width, height]: [u32; 2]) -> Result<tiny_skia::Pixmap, String> {
    let options = usvg::Options {
        image_href_resolver: usvg::ImageHrefResolver {
            resolve_data: Box::new(|_, _, _| None),
            resolve_string: Box::new(|_, _| None),
        },
        ..Default::default()
    };
    let tree = usvg::Tree::from_data(source, &options)
        .map_err(|cause| format!("icon SVG: expected a renderable role document: {cause}"))?;
    let mut pixmap = tiny_skia::Pixmap::new(width, height)
        .ok_or_else(|| format!("icon raster {width} × {height}: expected allocatable pixels"))?;
    let transform = tiny_skia::Transform::from_scale(
        width as f32 / tree.size().width(),
        height as f32 / tree.size().height(),
    );
    resvg::render(&tree, transform, &mut pixmap.as_mut());
    Ok(pixmap)
}
fn content_bounds(pixels: &tiny_skia::Pixmap) -> Result<[u32; 4], String> {
    let mut bounds = [pixels.width(), pixels.height(), 0, 0];
    for (index, _pixel) in pixels
        .pixels()
        .iter()
        .enumerate()
        .filter(|(_, pixel)| pixel.alpha() > 16)
    {
        let x = index as u32 % pixels.width();
        let y = index as u32 / pixels.width();
        bounds = [
            bounds[0].min(x),
            bounds[1].min(y),
            bounds[2].max(x + 1),
            bounds[3].max(y + 1),
        ];
    }
    if bounds[0] >= bounds[2] || bounds[1] >= bounds[3] {
        return Err(format!(
            "icon content bounds {bounds:?}: expected nonempty alpha coverage above 16"
        ));
    }
    Ok(bounds)
}
fn normalized_alpha(
    pixels: &tiny_skia::Pixmap,
    [left, top, right, bottom]: [u32; 4],
    edge: u32,
) -> Vec<u8> {
    let mut cropped = RgbaImage::new(right - left, bottom - top);
    for (x, y, pixel) in cropped.enumerate_pixels_mut() {
        let alpha = pixels.pixels()[((y + top) * pixels.width() + x + left) as usize].alpha();
        *pixel = image::Rgba([255, 255, 255, alpha]);
    }
    let scale = 0.92 * f64::from(edge) / f64::from((right - left).max(bottom - top));
    let width = (f64::from(right - left) * scale).round().max(1.) as u32;
    let height = (f64::from(bottom - top) * scale).round().max(1.) as u32;
    let scaled = imageops::resize(&cropped, width, height, FilterType::CatmullRom);
    let mut out = RgbaImage::new(edge, edge);
    imageops::replace(
        &mut out,
        &scaled,
        i64::from((edge - width) / 2),
        i64::from((edge - height) / 2),
    );
    out.pixels().map(|pixel| pixel[3]).collect()
}
