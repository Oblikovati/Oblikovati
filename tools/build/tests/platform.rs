// SPDX-License-Identifier: GPL-2.0-only

use super::*;

#[test]
fn legacy_matrix_is_complete_and_rust_artifacts_are_isolated() {
    let names = artifact_names("0.000200.1.0-nightly.26101012").unwrap();
    assert_eq!(names.len(), 9);
    assert_eq!(
        names.iter().filter(|name| name.contains("-cli-")).count(),
        6
    );
    assert!(
        names
            .iter()
            .all(|name| name.starts_with("oblikovati-rust-"))
    );
    assert!(
        names
            .iter()
            .any(|name| name.ends_with("macos-universal.zip"))
    );
    for entry in PLATFORMS {
        assert_eq!(platform(entry.target).unwrap().suffix, entry.suffix);
    }
    assert!(platform("unknown").is_err());
    for version in ["../../escape", "$(id)", "1.2.3", "1.2.3.4\n", "1.2.3.4-rc"] {
        assert!(artifact_names(version).is_err());
    }
}

#[test]
fn macos_platforms_decline_the_retired_sonoma_runner() {
    for entry in PLATFORMS
        .iter()
        .filter(|entry| entry.target.ends_with("apple-darwin"))
    {
        assert_ne!(
            entry.runner, "macos-14",
            "macOS platform {:?}: expected a supported GitHub runner after 2026-11-02",
            entry
        );
    }
}
#[test]
fn universal_package_declines_the_retired_sonoma_runner() {
    let workflow = include_str!("../../../.github/workflows/build.yml");
    assert!(
        !workflow.contains("runs-on: macos-14"),
        "universal package workflow: expected a supported GitHub runner after 2026-11-02"
    );
}
