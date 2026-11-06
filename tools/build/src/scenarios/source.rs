// SPDX-License-Identifier: GPL-2.0-only
use crate::process::BuildResult;
use std::path::{Path, PathBuf};
pub(super) trait ScenarioSource {
    fn feature_paths(&mut self, root: &Path) -> BuildResult<Vec<PathBuf>>;
    fn read(&mut self, path: &Path) -> BuildResult<Vec<u8>>;
}
pub(super) struct NativeScenarioSource;
impl ScenarioSource for NativeScenarioSource {
    fn feature_paths(&mut self, root: &Path) -> BuildResult<Vec<PathBuf>> {
        let mut paths = Vec::new();
        for entry in walkdir::WalkDir::new(root.join("features")).follow_links(false) {
            let entry = entry.map_err(|cause| {
                format!("scenario root {root:?}: expected a readable feature tree: {cause}")
            })?;
            if entry.file_type().is_file()
                && entry
                    .path()
                    .extension()
                    .is_some_and(|extension| extension == "feature")
            {
                paths.push(entry.into_path());
            }
        }
        Ok(paths)
    }
    fn read(&mut self, path: &Path) -> BuildResult<Vec<u8>> {
        std::fs::read(path).map_err(|cause| {
            format!("scenario source {path:?}: expected readable feature bytes: {cause}")
        })
    }
}
