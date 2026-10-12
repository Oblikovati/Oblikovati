// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::contract::{Appearance, Material};
use oblikovati_persistence::{assets::load_project_asset_library, load::PackageFileReader};
use std::{
    collections::BTreeMap,
    io,
    path::{Path, PathBuf},
};
#[derive(Default)]
struct MemoryProjectAssetFiles {
    contents: BTreeMap<PathBuf, Vec<u8>>,
    refused: Option<PathBuf>,
    reads: Vec<PathBuf>,
}
impl PackageFileReader for MemoryProjectAssetFiles {
    fn read(&mut self, path: &Path) -> io::Result<Vec<u8>> {
        self.reads.push(path.into());
        if self.refused.as_deref() == Some(path) {
            return Err(io::Error::new(
                io::ErrorKind::PermissionDenied,
                "named library permission refusal",
            ));
        }
        self.contents.get(path).cloned().ok_or_else(|| {
            io::Error::new(
                io::ErrorKind::NotFound,
                format!("library {path:?}: expected stored fixture or missing file"),
            )
        })
    }
}
fn libraries() -> MemoryProjectAssetFiles {
    let mut files = MemoryProjectAssetFiles::default();
    files.contents.insert("/design/appearances.yaml".into(),b"appearances: [{id: custom, name: Custom, albedo: '#804020', metallic: 0.3, roughness: 0.6, opacity: 1}]".to_vec());
    files.contents.insert("/design/materials.yaml".into(),b"materials: [{id: custom-steel, name: Custom steel, density: 7.87, appearance: custom, isotropy: isotropic}]".to_vec());
    files
}
#[test]
fn project_libraries_read_both_files_with_project_source_and_legacy_conversion() {
    let mut files = libraries();
    let assets = load_project_asset_library(Some(Path::new("/design")), &mut files).unwrap();
    assert_eq!(
        assets
            .materials()
            .get("custom-steel")
            .unwrap()
            .appearance_id(),
        "custom"
    );
    assert_eq!(
        assets.materials().get("custom-steel").unwrap().source().0,
        "project"
    );
    assert_eq!(
        assets.appearances().get("custom").unwrap().source().0,
        "project"
    );
    assert_eq!(
        assets.appearances().get("custom").unwrap().base().metalness,
        0.3
    );
    assert_eq!(
        files.reads,
        [
            PathBuf::from("/design/appearances.yaml"),
            PathBuf::from("/design/materials.yaml")
        ]
    );
}
#[test]
fn absent_libraries_and_absent_design_data_are_empty_without_path_fallback() {
    let mut files = MemoryProjectAssetFiles::default();
    assert!(
        load_project_asset_library(None, &mut files)
            .unwrap()
            .materials()
            .ordered()
            .next()
            .is_none()
    );
    assert!(files.reads.is_empty());
    assert!(
        load_project_asset_library(Some(Path::new("/design")), &mut files)
            .unwrap()
            .appearances()
            .ordered()
            .next()
            .is_none()
    );
    assert_eq!(files.reads.len(), 2);
}
#[test]
fn permission_parse_and_definition_errors_keep_named_file_context() {
    let mut files = libraries();
    files.refused = Some("/design/materials.yaml".into());
    assert!(
        load_project_asset_library(Some(Path::new("/design")), &mut files)
            .unwrap_err()
            .contains("materials.yaml")
    );
    files.refused = None;
    files
        .contents
        .insert("/design/materials.yaml".into(), b"materials: [".to_vec());
    assert!(
        load_project_asset_library(Some(Path::new("/design")), &mut files)
            .unwrap_err()
            .contains("materials.yaml")
    );
    files.contents.insert(
        "/design/materials.yaml".into(),
        b"materials: [{id: bad, density: .nan}]".to_vec(),
    );
    assert!(
        load_project_asset_library(Some(Path::new("/design")), &mut files)
            .unwrap_err()
            .contains("materials.yaml")
    );
}
#[test]
fn duplicate_definitions_and_wrong_library_shape_are_refused() {
    let mut files = libraries();
    files.contents.insert(
        "/design/materials.yaml".into(),
        b"materials: [{id: duplicate},{id: duplicate}]".to_vec(),
    );
    assert!(
        load_project_asset_library(Some(Path::new("/design")), &mut files)
            .unwrap_err()
            .contains("unique")
    );
    files.contents.insert(
        "/design/appearances.yaml".into(),
        b"unsupported: []".to_vec(),
    );
    assert!(
        load_project_asset_library(Some(Path::new("/design")), &mut files)
            .unwrap_err()
            .contains("appearances.yaml")
    );
}

#[test]
fn library_files_cannot_silently_ignore_the_other_asset_family() {
    let mut files = libraries();
    files.contents.insert(
        "/design/appearances.yaml".into(),
        b"materials: [{id: hidden}]".to_vec(),
    );
    assert!(
        load_project_asset_library(Some(Path::new("/design")), &mut files)
            .unwrap_err()
            .contains("appearances only")
    );
}
