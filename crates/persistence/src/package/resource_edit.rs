// SPDX-License-Identifier: GPL-2.0-only

use super::{DocumentPackage, Mapping, PackageResource, ResourceContents, Value};
use base64::{Engine, engine::general_purpose::STANDARD};
use std::collections::BTreeMap;

impl DocumentPackage {
    /// Replace one resource with owned content while retaining extension metadata.
    /// Callers must update recipe references when changing resource identity.
    /// Example: `let updated = package.with_resource("font", &resource);`.
    pub fn with_resource(&self, id: &str, resource: &PackageResource) -> Self {
        let mut candidate = self.clone();
        let mut section = self.resource_section();
        capture_entry(&mut section, id, resource);
        candidate
            .root
            .insert("resources".into(), Value::Mapping(section));
        candidate.resources.insert(id.into(), resource.clone());
        candidate
    }

    /// Capture an owned snapshot table while preserving unchanged entries and extensions.
    /// Example: `let saved = package.with_resources(&snapshot.resources);`.
    pub fn with_resources(&self, resources: &BTreeMap<String, PackageResource>) -> Self {
        if &self.resources == resources {
            return self.clone();
        }
        let mut candidate = self.clone();
        let section = self.updated_resources(resources);
        if section.is_empty() {
            candidate.root.remove("resources");
        } else {
            candidate
                .root
                .insert("resources".into(), Value::Mapping(section));
        }
        candidate.resources.clone_from(resources);
        candidate
    }

    fn updated_resources(&self, resources: &BTreeMap<String, PackageResource>) -> Mapping {
        let original = self.resource_section();
        let mut section = Mapping::new();
        for (id, resource) in resources {
            if let Some(entry) = original.get(id) {
                section.insert(id.clone().into(), entry.clone());
            }
            if self.resources.get(id) != Some(resource) {
                capture_entry(&mut section, id, resource);
            }
        }
        section
    }

    fn resource_section(&self) -> Mapping {
        self.root
            .get("resources")
            .and_then(Value::as_mapping)
            .cloned()
            .unwrap_or_default()
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

fn capture_entry(section: &mut Mapping, id: &str, resource: &PackageResource) {
    let mut entry = section
        .get(id)
        .and_then(Value::as_mapping)
        .cloned()
        .unwrap_or_default();
    capture_resource(&mut entry, resource);
    section.insert(id.into(), Value::Mapping(entry));
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
