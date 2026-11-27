// SPDX-License-Identifier: GPL-2.0-only
#[path = "../../../../test-utilities/rust/fixture_paths.rs"]
mod native_paths;
pub(super) use native_paths::{FixturePathMap, fixture_file_name, fixture_native_path};

/// Bind a declared recipe source; example: `fixture_source_package(template, "/source.opd")`.
pub(super) fn fixture_source_package(
    template: &str,
    source: &str,
) -> oblikovati_persistence::package::DocumentPackage {
    assert!(
        template.contains("$SOURCE_DOCUMENT"),
        "fixture template {template:?}: expected $SOURCE_DOCUMENT"
    );
    let recipe = template.replace("$SOURCE_DOCUMENT", &fixture_file_name(source));
    oblikovati_persistence::package::DocumentPackage::decode(recipe.as_bytes()).unwrap()
}
