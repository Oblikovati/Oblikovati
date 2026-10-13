// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl DocumentPackage {
    /// Restore persistent body display names; empty and null names mean no override.
    /// Example: `let names = package.restore_body_names()?;`.
    pub fn restore_body_names(&self) -> Result<BTreeMap<String, String>, PackageCodecError> {
        let mut names = BTreeMap::new();
        let Some(section) = self.root.get("bodyNames").and_then(Value::as_mapping) else {
            return Ok(names);
        };
        for key in section.keys() {
            let reference = key.as_str().ok_or_else(|| {
                PackageCodecError(format!(
                    "bodyNames key {key:?}: expected a string reference key"
                ))
            })?;
            if let Some(name) = super::body_display::body_text(&self.root, "bodyNames", reference)?
            {
                names.insert(reference.into(), name.into());
            }
        }
        Ok(names)
    }
    /// Capture names without changing unrelated package content or untouched legacy entries.
    /// Example: `let renamed = package.with_body_names(&names)?;`.
    pub fn with_body_names(
        &self,
        names: &BTreeMap<String, String>,
    ) -> Result<Self, PackageCodecError> {
        let before = self.restore_body_names()?;
        if &before == names {
            return Ok(self.clone());
        }
        let mut candidate = self.clone();
        let section = capture_name_section(self.root.get("bodyNames"), &before, names)?;
        candidate
            .root
            .insert("bodyNames".into(), Value::Mapping(section));
        Ok(candidate)
    }
}
fn capture_name_section(
    original: Option<&Value>,
    before: &BTreeMap<String, String>,
    names: &BTreeMap<String, String>,
) -> Result<Mapping, PackageCodecError> {
    let mut section = original
        .and_then(Value::as_mapping)
        .cloned()
        .unwrap_or_default();
    for key in before.keys().filter(|key| !names.contains_key(*key)) {
        section.remove(key.as_str());
    }
    for (key, name) in names {
        capture_name(&mut section, key, name)?;
    }
    Ok(section)
}
fn capture_name(section: &mut Mapping, key: &str, name: &str) -> Result<(), PackageCodecError> {
    if name.is_empty() {
        return Err(PackageCodecError(format!(
            "body name {key:?}={name:?}: expected nonempty text or removal of the override"
        )));
    }
    section.insert(key.into(), name.into());
    Ok(())
}
