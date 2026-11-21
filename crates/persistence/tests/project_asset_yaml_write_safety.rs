// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_persistence::{
    assets::{ProjectAssetFamily, load_project_asset_library, persist_project_asset_family},
    atomic::{AtomicFiles, StagedFile},
    load::PackageFileReader,
};
use std::{io, path::Path};

#[derive(Default)]
struct ProjectLibraryFiles {
    stages: usize,
}
struct UnexpectedProjectStage;
impl PackageFileReader for ProjectLibraryFiles {
    fn read(&mut self, path: &Path) -> io::Result<Vec<u8>> {
        if path.ends_with("appearances.yaml") {
            return Ok(b"appearances: [{id: custom, name: Custom, albedo: '#804020', metallic: 0.3, roughness: 0.6, opacity: 1, extension: {'<<': {x: 1}}}]".to_vec());
        }
        Err(io::Error::from(io::ErrorKind::NotFound))
    }
}
impl AtomicFiles for ProjectLibraryFiles {
    type Stage = UnexpectedProjectStage;
    fn stage(&mut self, _: &Path) -> io::Result<Self::Stage> {
        self.stages += 1;
        Err(io::Error::other("unexpected staging"))
    }
}
impl StagedFile for UnexpectedProjectStage {
    fn write_contents(&mut self, _: &[u8]) -> io::Result<()> {
        unreachable!()
    }
    fn synchronize(&mut self) -> io::Result<()> {
        unreachable!()
    }
    fn commit(self, _: &Path) -> io::Result<()> {
        unreachable!()
    }
}

#[test]
fn project_library_retained_ambiguous_key_refuses_before_staging() {
    let mut files = ProjectLibraryFiles::default();
    let directory = Path::new("/design");
    let before = load_project_asset_library(Some(directory), &mut files).unwrap();
    let error = persist_project_asset_family(
        directory,
        ProjectAssetFamily::Appearance,
        &before,
        &before,
        &mut files,
    )
    .unwrap_err();
    assert!(error.contains("<<"), "{error}");
    assert_eq!(files.stages, 0);
}
