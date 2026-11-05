// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentGuid, ReferenceEnvelope, ReferenceScheme};
mod decode;
/// Versioned semantic name: stable recipe sources, never a geometric anchor.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SemanticReferenceName {
    document: DocumentGuid,
    feature: u64,
    kind: u32,
    operation: String,
    role: String,
    sources: Vec<u64>,
}
impl SemanticReferenceName {
    /// Define one stable construction role; example: `SemanticReferenceName::new(doc, 1, 1, "sketch-prism", "side", &[1, 101])?`.
    pub fn new(
        document: DocumentGuid,
        feature: u64,
        kind: u32,
        operation: &str,
        role: &str,
        sources: &[u64],
    ) -> Result<Self, String> {
        require_semantic_tag(operation)?;
        require_semantic_tag(role)?;
        require_semantic_sources(feature, kind, sources)?;
        Ok(Self {
            document,
            feature,
            kind,
            operation: operation.into(),
            role: role.into(),
            sources: sources.into(),
        })
    }
    /// Serialize scheme 2 with fixed little-endian counts and IDs; example: `name.encode()?`.
    pub fn encode(&self) -> Result<Vec<u8>, String> {
        let mut payload = self.document.0.as_bytes().to_vec();
        append_tag(&mut payload, &self.operation);
        append_tag(&mut payload, &self.role);
        payload.extend_from_slice(&(self.sources.len() as u16).to_le_bytes());
        for source in &self.sources {
            payload.extend_from_slice(&source.to_le_bytes());
        }
        ReferenceEnvelope {
            scheme: ReferenceScheme::Semantic,
            context: self.feature,
            kind: self.kind,
            payload: &payload,
        }
        .encode_current()
        .map_err(|cause| cause.to_string())
    }
    /// Validate a persisted semantic name, refusing legacy meaning; example: `SemanticReferenceName::decode(&bytes)?`.
    pub fn decode(encoded: &[u8]) -> Result<Self, String> {
        let envelope = ReferenceEnvelope::decode(encoded).map_err(|cause| cause.to_string())?;
        if envelope.scheme != ReferenceScheme::Semantic {
            return Err(format!(
                "reference scheme {:?}: expected semantic scheme 2",
                envelope.scheme
            ));
        }
        decode::payload(envelope)
    }
    /// Read the persistent document namespace; example: `name.document() == active_guid`.
    pub fn document(&self) -> DocumentGuid {
        self.document
    }
}
fn require_semantic_tag(tag: &str) -> Result<(), String> {
    if tag.is_empty()
        || tag.len() > 64
        || !tag
            .bytes()
            .all(|byte| byte.is_ascii_lowercase() || byte.is_ascii_digit() || byte == b'-')
    {
        return Err(format!(
            "semantic tag {tag:?}: expected 1..64 lowercase ASCII letters, digits or hyphens"
        ));
    }
    Ok(())
}
fn require_semantic_sources(feature: u64, kind: u32, sources: &[u64]) -> Result<(), String> {
    if feature == 0
        || !matches!(kind, 1..=4 | 10 | 11 | 20 | 21)
        || sources.len() > 64
        || sources.contains(&0)
    {
        return Err(format!(
            "semantic feature {feature}, kind {kind}, sources {sources:?}: expected a nonzero feature, supported model kind and at most 64 nonzero source IDs"
        ));
    }
    Ok(())
}
fn append_tag(payload: &mut Vec<u8>, tag: &str) {
    payload.extend_from_slice(&(tag.len() as u16).to_le_bytes());
    payload.extend_from_slice(tag.as_bytes());
}
