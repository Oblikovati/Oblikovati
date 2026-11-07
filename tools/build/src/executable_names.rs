// SPDX-License-Identifier: GPL-2.0-only
//! Public executable names remain independent of the composition-root crate names.
pub(crate) const CLI_PACKAGE: &str = "oblikovati-cli";
pub(crate) const GUI_PACKAGE: &str = "oblikovati-head";
pub(crate) const CLI_BINARY: &str = CLI_PACKAGE;
pub(crate) const GUI_BINARY: &str = "oblikovati";
pub(crate) const APPLICATION_TARGETS: [(&str, &str); 2] =
    [(CLI_PACKAGE, CLI_BINARY), (GUI_PACKAGE, GUI_BINARY)];

pub(crate) fn binary_filename(binary: &str, target: &str) -> String {
    let suffix = if target.contains("windows") {
        ".exe"
    } else {
        ""
    };
    format!("{binary}{suffix}")
}

#[test]
fn public_filenames_follow_the_target_instead_of_the_build_host() {
    for target in ["x86_64-unknown-linux-gnu", "aarch64-apple-darwin"] {
        assert_eq!(binary_filename(GUI_BINARY, target), "oblikovati");
        assert_eq!(binary_filename(CLI_BINARY, target), "oblikovati-cli");
    }
    assert_eq!(
        binary_filename(GUI_BINARY, "aarch64-pc-windows-msvc"),
        "oblikovati.exe"
    );
    assert_eq!(
        binary_filename(CLI_BINARY, "x86_64-pc-windows-msvc"),
        "oblikovati-cli.exe"
    );
}
