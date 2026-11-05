// SPDX-License-Identifier: GPL-2.0-only

/// Resource classification follows its encoding, never its extensible type tag.
#[derive(Clone, Debug, Eq, PartialEq)]
pub enum ResourceContents {
    Utf8(String),
    Binary(Vec<u8>),
    Embedded,
}

/// A self-contained imported resource or a reference to an application resource.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct DocumentResource {
    pub kind: String,
    pub origin: String,
    pub contents: ResourceContents,
}

impl DocumentResource {
    /// Capture imported bytes without reading the display-only origin path.
    /// Valid NUL-free UTF-8 remains text; all other input stays binary.
    /// Example: `DocumentResource::imported("ObjFile", "mesh.obj", b"v 0 0 0\n")`.
    pub fn imported(kind: &str, origin: &str, bytes: &[u8]) -> Self {
        let contents = match std::str::from_utf8(bytes) {
            Ok(text) if !bytes.contains(&0) => ResourceContents::Utf8(text.into()),
            _ => ResourceContents::Binary(bytes.to_vec()),
        };
        Self {
            kind: kind.into(),
            origin: origin.into(),
            contents,
        }
    }
}
