// SPDX-License-Identifier: GPL-2.0-only

use super::{DocumentPackage, Mapping, PackageResource, ResourceContents, Value};
use base64::{Engine, engine::general_purpose::STANDARD};

impl DocumentPackage {
    /// Replace one resource with owned content while retaining extension metadata.
    /// Callers must update recipe references when changing resource identity.
    /// Example: `let updated = package.with_resource("font", &resource);`.
    pub fn with_resource(&self, id: &str, resource: &PackageResource) -> Self {
        let mut candidate = self.clone();
        let mut section = self
            .root
            .get("resources")
            .and_then(Value::as_mapping)
            .cloned()
            .unwrap_or_default();
        let mut entry = section
            .get(id)
            .and_then(Value::as_mapping)
            .cloned()
            .unwrap_or_default();
        capture_resource(&mut entry, resource);
        section.insert(id.into(), Value::Mapping(entry));
        candidate
            .root
            .insert("resources".into(), Value::Mapping(section));
        candidate.resources.insert(id.into(), resource.clone());
        candidate
    }

    /// Remove one resource; recipe reference validation remains the caller's responsibility.
    /// An absent ID leaves the package unchanged. Example: `package.without_resource("old")`.
    pub fn without_resource(&self, id: &str) -> Self {
        let mut candidate = self.clone();
        if candidate.resources.remove(id).is_none() {
            return candidate;
        }
        let section = candidate
            .root
            .get_mut("resources")
            .and_then(Value::as_mapping_mut)
            .expect("decoded resources have a mapping");
        section.remove(id);
        if section.is_empty() {
            candidate.root.remove("resources");
        }
        candidate
    }
}

fn capture_resource(entry: &mut Mapping, resource: &PackageResource) {
    entry.insert("type".into(), resource.kind.clone().into());
    entry.remove("origin");
    if !resource.origin.is_empty() {
        entry.insert("origin".into(), resource.origin.clone().into());
    }
    let (encoding, value) = match &resource.contents {
        ResourceContents::Utf8(text) => ("utf8", Some(text.clone())),
        ResourceContents::Binary(bytes) => ("base64", Some(STANDARD.encode(bytes))),
        ResourceContents::Embedded => ("embedded", None),
    };
    entry.insert("encoding".into(), encoding.into());
    entry.remove("value");
    if let Some(value) = value {
        entry.insert("value".into(), value.into());
    }
}
