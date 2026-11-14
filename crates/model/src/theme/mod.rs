// SPDX-License-Identifier: GPL-2.0-only
//! Semantic UI themes retain their source XML; geometry and body appearance have separate owners.
mod bindings;
mod color;
mod library;
mod palette;
mod preferences;
pub use preferences::ThemeSelectionStore;
mod xml_document;
mod xml_events;
use bindings::{Source, THEME_BINDINGS};
pub use library::ThemeLibrary;
pub use oblikovati_api::types::{Rgba, ThemeKind, ThemeToken};
use std::collections::BTreeMap;
use xml_document::BlenderThemeDocument;
pub type ThemePalette = BTreeMap<ThemeToken, Rgba>;

#[derive(Clone, Debug)]
pub struct ThemeDefinition {
    name: String,
    kind: ThemeKind,
    palette: ThemePalette,
    document: BlenderThemeDocument,
}
impl ThemeDefinition {
    /// Resolve a complete Blender theme and token snapshot; example: `ThemeDefinition::from_xml(xml, "Workshop", kind)?`.
    pub fn from_xml(bytes: &[u8], fallback_name: &str, kind: ThemeKind) -> Result<Self, String> {
        validate_kind(&kind)?;
        let document = BlenderThemeDocument::parse(bytes)?;
        let mut palette = palette::resolve_palette(&document)?;
        let snapshot = document.snapshot();
        apply_snapshot(&mut palette, &snapshot);
        let name = snapshot_name(&snapshot, fallback_name)?;
        Ok(Self {
            name,
            kind,
            palette,
            document,
        })
    }
    /// Load a fresh embedded dark theme; example: `let dark = ThemeDefinition::dark()?;`.
    pub fn dark() -> Result<Self, String> {
        Self::from_xml(
            include_bytes!("../../assets/themes/dark.xml"),
            "Dark",
            oblikovati_api::types::THEME_DARK,
        )
    }
    /// Load a fresh embedded light theme; example: `let light = ThemeDefinition::light()?;`.
    pub fn light() -> Result<Self, String> {
        Self::from_xml(
            include_bytes!("../../assets/themes/light.xml"),
            "Light",
            oblikovati_api::types::THEME_LIGHT,
        )
    }
    /// Read the display identity; example: `assert_eq!(ThemeDefinition::dark()?.name(), "Dark");`.
    pub fn name(&self) -> &str {
        &self.name
    }
    /// Read the canonical API classification; example: `theme.kind() == &types::THEME_CUSTOM`.
    pub fn kind(&self) -> &ThemeKind {
        &self.kind
    }
    /// Read the complete immutable color snapshot; example: `theme.palette().len()`.
    pub fn palette(&self) -> &ThemePalette {
        &self.palette
    }
    /// Copy a color, using Go's magenta fallback for an unknown token; example: `theme.color(&types::TOKEN_CHROME_TEXT)`.
    pub fn color(&self, token: &ThemeToken) -> Rgba {
        self.palette.get(token).cloned().unwrap_or(Rgba {
            r: 1.,
            g: 0.,
            b: 1.,
            a: 1.,
        })
    }
    /// Make an independent custom snapshot; example: `let custom = dark.duplicate("Workshop")?;`.
    pub fn duplicate(&self, name: &str) -> Result<Self, String> {
        validate_name(name)?;
        let mut copy = self.clone();
        copy.name = name.into();
        copy.kind = oblikovati_api::types::THEME_CUSTOM;
        Ok(copy)
    }
    /// Recolor an editable theme and its authored XML attribute; example: `custom.set_color(token, rgba)?;`.
    pub fn set_color(&mut self, token: &ThemeToken, value: Rgba) -> Result<bool, String> {
        if self.kind != oblikovati_api::types::THEME_CUSTOM {
            return Ok(false);
        }
        color::validate_color(&value)?;
        let binding = THEME_BINDINGS
            .iter()
            .find(|binding| &binding.token == token)
            .ok_or_else(|| format!("theme token {:?}: expected a known semantic slot", token.0))?;
        if let Source::Direct(path, attribute, _) = binding.source {
            self.document
                .set_attribute(&path.nodes(), attribute, color::format_hex(&value))?;
        }
        self.palette.insert(token.clone(), value);
        Ok(true)
    }
    /// Read canonical hexadecimal colors for the API view; example: `theme.hex_colors()`.
    pub fn hex_colors(&self) -> BTreeMap<String, String> {
        self.encoded_colors().collect()
    }
    fn encoded_colors(&self) -> impl Iterator<Item = (String, String)> + '_ {
        THEME_BINDINGS.iter().map(|binding| {
            (
                binding.token.0.to_string(),
                color::format_hex(&self.color(&binding.token)),
            )
        })
    }
    /// Serialize a self-contained snapshot without dropping unrelated XML; example: `let bytes = custom.to_xml()?;`.
    pub fn to_xml(&self) -> Result<Vec<u8>, String> {
        self.document
            .encode(&self.name, self.encoded_colors().collect())
    }
}
impl oblikovati_api::contract::Theme for ThemeDefinition {
    fn color(&self, token: ThemeToken) -> Rgba {
        self.color(&token)
    }
    fn kind(&self) -> ThemeKind {
        self.kind.clone()
    }
    fn name(&self) -> String {
        self.name.clone()
    }
}
const _: fn(&ThemeDefinition) -> &dyn oblikovati_api::contract::Theme = |theme| theme;
/// Iterate semantic slots in canonical display order; example: `theme_tokens().count()`.
pub fn theme_tokens() -> impl ExactSizeIterator<Item = &'static ThemeToken> {
    THEME_BINDINGS.iter().map(|binding| &binding.token)
}
fn apply_snapshot(palette: &mut ThemePalette, snapshot: &BTreeMap<String, String>) {
    for binding in THEME_BINDINGS {
        if let Some(value) = snapshot.get(binding.token.0.as_ref())
            && let Ok(value) = color::parse_hex(value)
        {
            palette.insert(binding.token.clone(), value);
        }
    }
}
fn validate_kind(kind: &ThemeKind) -> Result<(), String> {
    use oblikovati_api::types::{THEME_CUSTOM, THEME_DARK, THEME_LIGHT};
    if [THEME_DARK, THEME_LIGHT, THEME_CUSTOM].contains(kind) {
        return Ok(());
    }
    Err(format!(
        "theme kind {:?}: expected dark, light, or custom",
        kind.0
    ))
}
fn validate_name(name: &str) -> Result<(), String> {
    if !name.is_empty() && !name.contains('\0') {
        return Ok(());
    }
    Err(format!(
        "theme name {name:?}: expected nonempty text without NUL"
    ))
}

#[cfg(test)]
mod xml_tests;

fn snapshot_name(snapshot: &BTreeMap<String, String>, fallback: &str) -> Result<String, String> {
    let name = snapshot
        .get("name")
        .filter(|name| !name.is_empty())
        .map(String::as_str)
        .unwrap_or(fallback);
    validate_name(name)?;
    Ok(name.into())
}
