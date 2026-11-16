// SPDX-License-Identifier: GPL-2.0-only
//! Canonical, bounded identities derived from complete parent names (#4045).
use super::{DocumentGuid, ReferenceEnvelope, ReferenceScheme, SemanticReferenceName};

const MAX_DEPTH: usize = 8;
const MAX_PARENTS: usize = 16;
const MAX_BYTES: usize = 16384;

/// A versioned topology identity retaining its complete ordered parent set.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct DerivedReferenceName {
    document: DocumentGuid,
    feature: u64,
    kind: u32,
    role: String,
    parents: Vec<Vec<u8>>,
}

impl DerivedReferenceName {
    /// Create a canonical parent-derived name; example: `DerivedReferenceName::new(doc, 2, 2, "edge", &[face])?`.
    pub fn new(
        document: DocumentGuid,
        feature: u64,
        kind: u32,
        role: &str,
        parents: &[Vec<u8>],
    ) -> Result<Self, String> {
        let mut ordered = parents.to_vec();
        ordered.sort();
        validate_parts(document, feature, kind, role, &ordered, 0)?;
        let result = Self {
            document,
            feature,
            kind,
            role: role.into(),
            parents: ordered,
        };
        result.encode()?;
        Ok(result)
    }

    /// Encode a canonical versioned name; example: `name.encode()?`.
    pub fn encode(&self) -> Result<Vec<u8>, String> {
        let mut payload = self.document.0.as_bytes().to_vec();
        payload.push(self.role.len() as u8);
        payload.extend_from_slice(self.role.as_bytes());
        payload.push(self.parents.len() as u8);
        for parent in &self.parents {
            payload.extend_from_slice(&(parent.len() as u16).to_le_bytes());
            payload.extend_from_slice(parent);
        }
        if payload.len() > MAX_BYTES {
            return Err(format!(
                "derived payload {}: expected at most {MAX_BYTES} bytes",
                payload.len()
            ));
        }
        ReferenceEnvelope {
            scheme: ReferenceScheme::Derived,
            context: self.feature,
            kind: self.kind,
            payload: &payload,
        }
        .encode_current()
        .map_err(|cause| cause.to_string())
    }

    /// Decode and validate a canonical derived name; example: `DerivedReferenceName::decode(&key)?`.
    pub fn decode(bytes: &[u8]) -> Result<Self, String> {
        decode_at(bytes, 0)
    }
    /// Read the owning document; example: `name.document()`.
    pub fn document(&self) -> DocumentGuid {
        self.document
    }
}

/// Validate a topology name and read its document; example: `topology_reference_document(&key)?`.
pub fn topology_reference_document(bytes: &[u8]) -> Result<DocumentGuid, String> {
    parent_document(bytes, 0)
}

fn parent_document(bytes: &[u8], depth: usize) -> Result<DocumentGuid, String> {
    let envelope = ReferenceEnvelope::decode(bytes).map_err(|cause| cause.to_string())?;
    if !matches!(envelope.kind, 1..=4) {
        return Err(format!(
            "topology parent kind {}: expected 1..=4",
            envelope.kind
        ));
    }
    match envelope.scheme {
        ReferenceScheme::Semantic => {
            SemanticReferenceName::decode(bytes).map(|name| name.document())
        }
        ReferenceScheme::Derived => decode_at(bytes, depth).map(|name| name.document),
        other => Err(format!(
            "topology reference scheme {other:?}: expected semantic or derived"
        )),
    }
}

fn validate_parts(
    document: DocumentGuid,
    feature: u64,
    kind: u32,
    role: &str,
    parents: &[Vec<u8>],
    depth: usize,
) -> Result<(), String> {
    if depth >= MAX_DEPTH
        || feature == 0
        || !matches!(kind, 1..=4)
        || role.is_empty()
        || role.len() > 64
        || !role
            .bytes()
            .all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || c == b'-')
    {
        return Err(format!(
            "derived feature {feature}, kind {kind}, role {role:?}, depth {depth}: expected valid topology identity below depth {MAX_DEPTH}"
        ));
    }
    if parents.is_empty()
        || parents.len() > MAX_PARENTS
        || parents.windows(2).any(|pair| pair[0] >= pair[1])
    {
        return Err(format!(
            "derived parents {}: expected 1..={MAX_PARENTS} distinct canonical parents",
            parents.len()
        ));
    }
    for parent in parents {
        if parent.len() > u16::MAX as usize || parent_document(parent, depth + 1)? != document {
            return Err(format!(
                "derived parent {parent:?}: expected a bounded name in document {document}"
            ));
        }
    }
    Ok(())
}

fn decode_at(bytes: &[u8], depth: usize) -> Result<DerivedReferenceName, String> {
    if depth >= MAX_DEPTH || bytes.len() > MAX_BYTES + 21 {
        return Err(format!(
            "derived reference depth {depth}, bytes {}: expected depth below {MAX_DEPTH} and bounded bytes",
            bytes.len()
        ));
    }
    let envelope = ReferenceEnvelope::decode(bytes).map_err(|cause| cause.to_string())?;
    if envelope.scheme != ReferenceScheme::Derived {
        return Err(format!(
            "reference scheme {:?}: expected derived",
            envelope.scheme
        ));
    }
    let mut reader = Reader(envelope.payload);
    let guid: [u8; 16] = reader.take(16)?.try_into().expect("validated GUID length");
    let document = DocumentGuid(uuid::Uuid::from_bytes(guid));
    let role_len = reader.byte()? as usize;
    let role = std::str::from_utf8(reader.take(role_len)?).map_err(|cause| cause.to_string())?;
    let parents = reader.parents()?;
    if !reader.0.is_empty() {
        return Err(format!(
            "derived payload trailing bytes {}: expected none",
            reader.0.len()
        ));
    }
    validate_parts(
        document,
        envelope.context,
        envelope.kind,
        role,
        &parents,
        depth,
    )?;
    Ok(DerivedReferenceName {
        document,
        feature: envelope.context,
        kind: envelope.kind,
        role: role.into(),
        parents,
    })
}

struct Reader<'a>(&'a [u8]);
impl<'a> Reader<'a> {
    fn take(&mut self, count: usize) -> Result<&'a [u8], String> {
        if count > self.0.len() {
            return Err(format!(
                "derived payload remaining {}: expected {count} bytes",
                self.0.len()
            ));
        }
        let (value, rest) = self.0.split_at(count);
        self.0 = rest;
        Ok(value)
    }
    fn byte(&mut self) -> Result<u8, String> {
        Ok(self.take(1)?[0])
    }
    fn parents(&mut self) -> Result<Vec<Vec<u8>>, String> {
        let count = self.byte()? as usize;
        if count == 0 || count > MAX_PARENTS {
            return Err(format!(
                "derived parent count {count}: expected 1..={MAX_PARENTS}"
            ));
        }
        let mut parents = Vec::with_capacity(count);
        for _ in 0..count {
            let len =
                u16::from_le_bytes(self.take(2)?.try_into().expect("validated length")) as usize;
            parents.push(self.take(len)?.to_vec());
        }
        Ok(parents)
    }
}
