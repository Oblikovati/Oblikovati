// SPDX-License-Identifier: GPL-2.0-only
//! The small picture a saved file carries, so a dialog can show it without opening the document.
use super::{DocumentPackage, PackageCodecError, Value, binary_text::decode_base64};
use base64::{Engine, engine::general_purpose::STANDARD};

const SECTION: &str = "previewPicture";
const WIDTH: &str = "width";
const HEIGHT: &str = "height";
const RGB: &str = "rgb";
const BYTES_PER_CELL: usize = 3;

/// Cells in reading order, three bytes each.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SavedPicture {
    pub extent: [u32; 2],
    pub rgb: Vec<[u8; 3]>,
}
impl DocumentPackage {
    /// Read the stored picture; a file saved without one has none.
    /// Example: `let picture = package.restore_preview_picture()?;`.
    pub fn restore_preview_picture(&self) -> Result<Option<SavedPicture>, PackageCodecError> {
        self.root.get(SECTION).map(picture_of_section).transpose()
    }
    /// Store this picture, or leave the file without one.
    /// Example: `let saved = package.with_preview_picture(Some(&picture));`.
    pub fn with_preview_picture(&self, picture: Option<&SavedPicture>) -> Self {
        let mut candidate = self.clone();
        match picture {
            Some(picture) => {
                candidate
                    .root
                    .insert(SECTION.into(), section_of_picture(picture));
            }
            None => {
                candidate.root.remove(SECTION);
            }
        }
        candidate
    }
}
/// Read a closed file's picture for a dialog; a file that cannot be read has none.
/// Example: `preview_picture_of_file(Path::new("/work/part.opd"))`.
pub fn preview_picture_of_file(path: &std::path::Path) -> Option<SavedPicture> {
    let encoded = std::fs::read(path).ok()?;
    DocumentPackage::decode(&encoded)
        .and_then(|package| package.restore_preview_picture())
        .ok()
        .flatten()
}
fn section_of_picture(picture: &SavedPicture) -> Value {
    let bytes: Vec<u8> = picture.rgb.iter().flatten().copied().collect();
    let mut section = serde_yaml_ng::Mapping::new();
    section.insert(WIDTH.into(), picture.extent[0].into());
    section.insert(HEIGHT.into(), picture.extent[1].into());
    section.insert(RGB.into(), STANDARD.encode(bytes).into());
    Value::Mapping(section)
}
fn picture_of_section(section: &Value) -> Result<SavedPicture, PackageCodecError> {
    let refused = || {
        PackageCodecError(format!(
            "{SECTION} {section:?}: expected {WIDTH}, {HEIGHT} and base64 {RGB}"
        ))
    };
    let side = |name: &str| {
        section
            .get(name)
            .and_then(Value::as_u64)
            .and_then(|side| u32::try_from(side).ok())
    };
    let extent = [
        side(WIDTH).ok_or_else(refused)?,
        side(HEIGHT).ok_or_else(refused)?,
    ];
    let text = section
        .get(RGB)
        .and_then(Value::as_str)
        .ok_or_else(refused)?;
    let bytes = decode_base64(SECTION, text)?;
    let cells = extent[0] as usize * extent[1] as usize;
    if bytes.len() != cells * BYTES_PER_CELL {
        return Err(PackageCodecError(format!(
            "{SECTION} {extent:?} with {} bytes: expected {} bytes",
            bytes.len(),
            cells * BYTES_PER_CELL
        )));
    }
    let rgb = bytes.as_chunks::<BYTES_PER_CELL>().0.to_vec();
    Ok(SavedPicture { extent, rgb })
}

#[cfg(test)]
mod tests {
    use super::*;
    fn package() -> DocumentPackage {
        DocumentPackage::empty(oblikovati_api::types::DOCUMENT_PART, "Part1")
    }
    fn picture() -> SavedPicture {
        SavedPicture {
            extent: [2, 1],
            rgb: vec![[1, 2, 3], [250, 251, 252]],
        }
    }
    #[test]
    fn a_stored_picture_survives_the_file_encoding() {
        let package = package().with_preview_picture(Some(&picture()));
        let decoded = DocumentPackage::decode(&package.encode().unwrap()).unwrap();
        assert_eq!(decoded.restore_preview_picture().unwrap(), Some(picture()));
    }
    #[test]
    fn saving_without_a_picture_takes_the_old_one_away() {
        let package = package().with_preview_picture(Some(&picture()));
        let without = package.with_preview_picture(None);
        assert_eq!(without.restore_preview_picture().unwrap(), None);
    }
    #[test]
    fn a_picture_shorter_than_its_extent_is_refused_with_both_lengths() {
        let mut package = package().with_preview_picture(Some(&picture()));
        let mut section = section_of_picture(&picture());
        section[WIDTH] = 3.into();
        package.root.insert(SECTION.into(), section);
        let refusal = package.restore_preview_picture().unwrap_err();
        assert!(
            refusal.0.contains("6 bytes") && refusal.0.contains("9 bytes"),
            "{refusal:?}"
        );
    }
    #[test]
    fn a_missing_file_has_no_picture() {
        assert_eq!(
            preview_picture_of_file(std::path::Path::new("/nonexistent/part.opd")),
            None
        );
    }
}
