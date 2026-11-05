// SPDX-License-Identifier: GPL-2.0-only
use std::path::PathBuf;

/// Build a native absolute project fixture path without reading or creating files.
/// Example: `native_project_fixture_root("workspace")` for an injected lookup root.
pub fn native_project_fixture_root(name: &str) -> PathBuf {
    // #3795: Unix-rooted literals are drive-relative on Windows and fail before the probe runs.
    std::env::temp_dir()
        .join("oblikovati-project-fixtures")
        .join(name)
}
