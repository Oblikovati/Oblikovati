// SPDX-License-Identifier: GPL-2.0-only

use crate::process::BuildResult;
use std::{
    fs,
    path::{Path, PathBuf},
};
use walkdir::WalkDir;

pub(crate) fn source_files(root: &Path) -> BuildResult<Vec<PathBuf>> {
    walk_files(root, true)
}

pub(crate) fn all_files(root: &Path) -> BuildResult<Vec<PathBuf>> {
    walk_files(root, false)
}

fn walk_files(root: &Path, source_only: bool) -> BuildResult<Vec<PathBuf>> {
    let entries = WalkDir::new(root).into_iter().filter_entry(|entry| {
        !source_only
            || entry.depth() == 0
            || ![
                ".git",
                "target",
                "node_modules",
                "dist",
                "artifacts",
                "stage",
                ".tools",
            ]
            .contains(&entry.file_name().to_string_lossy().as_ref())
    });
    let mut files = Vec::new();
    for entry in entries {
        let entry = entry.map_err(|error| format!("source tree {}: {error}", root.display()))?;
        if entry.file_type().is_file() {
            files.push(entry.path().to_path_buf());
        }
        if entry.file_type().is_symlink() {
            return Err(format!(
                "source path {}: expected a regular file or directory",
                entry.path().display()
            ));
        }
    }
    files.sort();
    Ok(files)
}

pub(crate) fn read_text(path: &Path) -> BuildResult<String> {
    fs::read_to_string(path).map_err(|error| format!("read {}: {error}", path.display()))
}

pub(crate) fn write_text(path: &Path, text: &str) -> BuildResult<()> {
    let parent = path
        .parent()
        .ok_or_else(|| format!("path {}: expected a parent", path.display()))?;
    fs::create_dir_all(parent).map_err(|error| format!("create {}: {error}", parent.display()))?;
    fs::write(path, text).map_err(|error| format!("write {}: {error}", path.display()))
}

pub(crate) fn relative_name(root: &Path, path: &Path) -> BuildResult<String> {
    let relative = path.strip_prefix(root).map_err(|error| {
        format!(
            "path {}: expected a child of {}: {error}",
            path.display(),
            root.display()
        )
    })?;
    Ok(relative.to_string_lossy().replace('\\', "/"))
}

#[cfg(test)]
#[path = "../tests/tree.rs"]
pub(crate) mod tests;
