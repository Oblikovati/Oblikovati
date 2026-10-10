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
