// SPDX-License-Identifier: GPL-2.0-only

use super::{DocumentPackage, PackageCodecError, Value};
use base64::{Engine, engine::general_purpose::STANDARD};

impl DocumentPackage {
    /// Store owned bytes in one named stream, retaining every unrelated section.
    /// Example: `let updated = package.with_stream("addin/state", &[1, 2]);`.
    pub fn with_stream(&self, name: &str, bytes: &[u8]) -> Self {
        let mut candidate = self.clone();
        let mut section = candidate.stream_section();
        section.insert(name.into(), STANDARD.encode(bytes).into());
        candidate.streams.insert(name.into(), bytes.to_vec());
        candidate
            .root
            .insert("data".into(), Value::Mapping(section));
        candidate
    }

    /// Remove one stream; an absent name leaves the package representation unchanged.
    /// Example: `let updated = package.without_stream("addin/obsolete");`.
    pub fn without_stream(&self, name: &str) -> Self {
        let mut candidate = self.clone();
        candidate.remove_stream(name);
        candidate
    }

    fn remove_stream(&mut self, name: &str) {
        if self.streams.remove(name).is_none() {
            return;
        }
        let section = self
            .root
            .get_mut("data")
            .and_then(Value::as_mapping_mut)
            .expect("decoded streams have a mapping");
        section.remove(name);
        if section.is_empty() {
            self.root.remove("data");
        }
    }

    /// Drop only regenerable `cache/` streams and report uncompressed bytes reclaimed.
    /// Example: `let (compacted, reclaimed) = package.without_caches()?;`.
    pub fn without_caches(&self) -> Result<(Self, usize), PackageCodecError> {
        let mut candidate = self.clone();
        let mut reclaimed = 0usize;
        for (name, bytes) in self
            .streams
            .iter()
            .filter(|(name, _)| name.starts_with("cache/"))
        {
            reclaimed = reclaimed.checked_add(bytes.len()).ok_or_else(|| {
                PackageCodecError(format!(
                    "cache byte count {reclaimed} plus {}: expected a representable usize",
                    bytes.len()
                ))
            })?;
            candidate.remove_stream(name);
        }
        Ok((candidate, reclaimed))
    }

    fn stream_section(&self) -> serde_yaml_ng::Mapping {
        self.root
            .get("data")
            .and_then(Value::as_mapping)
            .cloned()
            .unwrap_or_default()
    }
}
