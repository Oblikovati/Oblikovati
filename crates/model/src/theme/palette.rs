// SPDX-License-Identifier: GPL-2.0-only
use super::{
    ThemePalette,
    bindings::{Source, THEME_BINDINGS},
    color,
    xml_document::BlenderThemeDocument,
};
use oblikovati_api::types::{self, Rgba, ThemeToken};
pub(super) fn resolve_palette(document: &BlenderThemeDocument) -> Result<ThemePalette, String> {
    let (mut palette, failures) = resolve_direct_palette(document);
    if !failures.is_empty() {
        return Err(format!(
            "theme mapping: expected complete Blender attributes; {}",
            failures.join("; ")
        ));
    }
    resolve_derived(&mut palette)?;
    Ok(palette)
}
fn resolve_direct_palette(document: &BlenderThemeDocument) -> (ThemePalette, Vec<String>) {
    let mut palette = ThemePalette::new();
    let mut failures = Vec::new();
    for binding in THEME_BINDINGS {
        let Source::Direct(path, attribute, opaque) = binding.source else {
            continue;
        };
        match resolve_direct(document, path.nodes(), attribute, opaque) {
            Ok(value) => {
                palette.insert(binding.token.clone(), value);
            }
            Err(cause) => failures.push(format!("{}: {cause}", binding.token.0)),
        }
    }
    (palette, failures)
}
fn resolve_direct(
    document: &BlenderThemeDocument,
    path: Vec<&str>,
    attribute: &str,
    opaque: bool,
) -> Result<Rgba, String> {
    let value = document.attribute(&path, attribute).ok_or_else(|| {
        format!("element {path:?} attribute {attribute:?}: expected a mapped color")
    })?;
    let mut result = color::parse_hex(value)?;
    if opaque {
        result.a = 1.;
    }
    Ok(result)
}
fn resolve_derived(palette: &mut ThemePalette) -> Result<(), String> {
    for binding in THEME_BINDINGS {
        let value = match &binding.source {
            Source::Direct(..) => continue,
            Source::Mix(base, amount) => color::mix_rgb(
                required(palette, base)?,
                required(palette, &types::TOKEN_CHROME_TEXT)?,
                *amount,
            ),
            Source::Alpha(base, amount) => scaled_alpha(required(palette, base)?, *amount),
            Source::SketchGreen => {
                color::sketch_green(required(palette, &types::TOKEN_VIEWPORT_BG)?)
            }
        };
        palette.insert(binding.token.clone(), value);
    }
    Ok(())
}
fn required<'a>(palette: &'a ThemePalette, token: &ThemeToken) -> Result<&'a Rgba, String> {
    palette.get(token).ok_or_else(|| {
        format!(
            "theme base {:?}: expected a directly resolved color",
            token.0
        )
    })
}
fn scaled_alpha(base: &Rgba, amount: f32) -> Rgba {
    let mut result = base.clone();
    result.a = (result.a * amount).clamp(0., 1.);
    result
}
