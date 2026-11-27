// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl DocumentPackage {
    /// Read preserved body names and legacy style intent; example: `package.body_display_metadata(key)?`.
    pub fn body_display_metadata(
        &self,
        key: &str,
    ) -> Result<(Option<&str>, Option<&str>), PackageCodecError> {
        Ok((
            body_text(&self.root, "bodyNames", key)?,
            body_text(&self.root, "bodyColorStyles", key)?,
        ))
    }
}
pub(super) fn body_text<'a>(
    root: &'a Mapping,
    section: &str,
    key: &str,
) -> Result<Option<&'a str>, PackageCodecError> {
    let Some(value) = root.get(section).and_then(|section| section.get(key)) else {
        return Ok(None);
    };
    if value.is_null() {
        return Ok(None);
    }
    let text = value.as_str().ok_or_else(|| {
        PackageCodecError(format!(
            "{section} reference {key:?}, value {value:?}: expected body display text"
        ))
    })?;
    Ok((!text.is_empty()).then_some(text))
}
