// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    packaging::copy_file,
    process::{BuildResult, Runner},
    tree::all_files as source_files,
};
use std::{
    collections::BTreeSet,
    path::{Path, PathBuf},
};

fn host_library(name: &str) -> bool {
    // A package must use the host glibc and ELF interpreter as one coherent ABI.
    [
        "libc.so.6",
        "libm.so.6",
        "libpthread.so.0",
        "libdl.so.2",
        "librt.so.1",
        "libresolv.so.2",
        "libutil.so.1",
    ]
    .contains(&name)
}

fn linked_libraries(output: &str) -> BuildResult<Vec<PathBuf>> {
    let mut libraries = Vec::new();
    for line in output.lines() {
        let Some((name, location)) = line.trim().split_once(" => ") else {
            continue;
        };
        if location.contains("not found") {
            return Err(format!(
                "Linux dependency {name:?}: expected a resolved shared library"
            ));
        }
        if host_library(name) {
            continue;
        }
        let path = location.split_once(" (").map_or(location, |(path, _)| path);
        if !Path::new(path).is_absolute() {
            return Err(format!(
                "Linux library path {path:?}: expected an absolute ldd path"
            ));
        }
        libraries.push(path.into());
    }
    Ok(libraries)
}

pub(crate) fn bundle(root: &Path, runner: &dyn Runner, stage: &Path) -> BuildResult<()> {
    let mut names = BTreeSet::new();
    for binary in source_files(&stage.join("bin"))? {
        let output = runner.output(root, "ldd", &[&binary.to_string_lossy()])?;
        for library in linked_libraries(&output)? {
            let name = library
                .file_name()
                .ok_or_else(|| format!("{}: expected a library filename", library.display()))?;
            if names.insert(name.to_owned()) {
                copy_file(&library, &stage.join("lib").join(name))?;
            }
        }
    }
    Ok(())
}

#[cfg(test)]
#[path = "../tests/linux.rs"]
mod tests;
