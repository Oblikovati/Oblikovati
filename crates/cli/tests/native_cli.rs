// SPDX-License-Identifier: GPL-2.0-only

use std::process::Command;

#[test]
fn executable_inspects_a_native_package_and_reports_real_exit_codes() {
    let directory = tempfile::tempdir().unwrap();
    let file = directory.path().join("part Ω.opd");
    let bytes = b"schemaVersion: 2\ndocumentType: 1\ndisplayName: Bracket\n";
    std::fs::write(&file, bytes).unwrap();
    let output = Command::new(env!("CARGO_BIN_EXE_oblikovati-cli"))
        .arg("info")
        .arg(&file)
        .arg("--json")
        .output()
        .unwrap();
    assert!(output.status.success());
    assert!(output.stderr.is_empty());
    assert_eq!(
        serde_json::from_slice::<serde_json::Value>(&output.stdout).unwrap()["name"],
        "Bracket"
    );
    assert_eq!(std::fs::read(&file).unwrap(), bytes);
    let missing = Command::new(env!("CARGO_BIN_EXE_oblikovati-cli"))
        .arg("info")
        .arg(directory.path().join("missing.opd"))
        .output()
        .unwrap();
    assert_eq!(missing.status.code(), Some(1));
    assert!(missing.stdout.is_empty());
    assert!(
        String::from_utf8(missing.stderr)
            .unwrap()
            .contains("missing.opd")
    );
    let version = Command::new(env!("CARGO_BIN_EXE_oblikovati-cli"))
        .arg("--version")
        .output()
        .unwrap();
    assert!(version.status.success());
    assert!(
        String::from_utf8(version.stdout)
            .unwrap()
            .starts_with("oblikovati-cli ")
    );
}
