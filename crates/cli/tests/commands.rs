// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_cli::{BUILD_MANIFEST, BuildIdentity, run};
use oblikovati_persistence::load::PackageFileReader;
use std::{
    ffi::OsString,
    io::{self, Write},
    path::{Path, PathBuf},
};

struct RecordedPackageReader {
    encoded: Vec<u8>,
    paths: Vec<PathBuf>,
    denied: bool,
}
impl PackageFileReader for RecordedPackageReader {
    fn read(&mut self, path: &Path) -> io::Result<Vec<u8>> {
        self.paths.push(path.into());
        if self.denied {
            return Err(io::Error::other("package read denied"));
        }
        Ok(self.encoded.clone())
    }
}
struct RefusingOutput;
impl Write for RefusingOutput {
    fn write(&mut self, _: &[u8]) -> io::Result<usize> {
        Err(io::Error::other("output closed"))
    }
    fn flush(&mut self) -> io::Result<()> {
        Ok(())
    }
}
fn reader(kind: u32) -> RecordedPackageReader {
    RecordedPackageReader {
        encoded: format!("schemaVersion: 2\ndocumentType: {kind}\ndisplayName: 'Part Ω'\n")
            .into_bytes(),
        paths: Vec::new(),
        denied: false,
    }
}
fn arguments(values: &[&str]) -> Vec<OsString> {
    values.iter().map(OsString::from).collect()
}

#[test]
fn info_preserves_the_go_text_report_for_all_document_kinds() {
    for (kind, name) in [
        (1, "part"),
        (2, "assembly"),
        (3, "drawing"),
        (4, "presentation"),
        (0, "unknown"),
        (99, "unknown"),
    ] {
        let mut input = reader(kind);
        let mut output = Vec::new();
        run(&arguments(&["info", "sample.opd"]), &mut input, &mut output).unwrap();
        assert_eq!(
            String::from_utf8(output).unwrap(),
            format!("schema version: 2\ntype: {name}\nname: Part Ω\n")
        );
        assert_eq!(input.paths, [PathBuf::from("sample.opd")]);
    }
}

#[test]
fn info_json_flag_can_precede_or_follow_the_path_and_escape_names() {
    for args in [
        ["info", "--json", "sample.opd"],
        ["info", "sample.opd", "--json"],
        ["info", "sample.opd", "-json=true"],
    ] {
        let mut output = Vec::new();
        run(&arguments(&args), &mut reader(1), &mut output).unwrap();
        let report: serde_json::Value = serde_json::from_slice(&output).unwrap();
        assert_eq!(
            report,
            serde_json::json!({"schemaVersion":2, "type":"part", "name":"Part Ω"})
        );
        assert!(output.ends_with(b"\n"));
    }
    let mut output = Vec::new();
    run(
        &arguments(&["info", "sample.opd", "--json", "--json=false"]),
        &mut reader(1),
        &mut output,
    )
    .unwrap();
    assert!(output.starts_with(b"schema version: 2"));
}

#[test]
fn invalid_or_unported_commands_do_not_read_files_or_write_success_output() {
    for args in [
        vec![],
        vec!["wrong"],
        vec!["new", "part", "sample.opd"],
        vec!["open", "sample.opd"],
        vec!["info"],
        vec!["info", "a", "b"],
        vec!["info", "sample.opd", "--bad"],
        vec!["version", "extra"],
    ] {
        let mut input = reader(1);
        let mut output = Vec::new();
        let error = run(&arguments(&args), &mut input, &mut output).unwrap_err();
        assert!(!error.is_empty());
        assert!(input.paths.is_empty());
        assert!(output.is_empty());
    }
}

#[test]
fn info_reports_file_codec_and_output_failures() {
    let mut input = reader(1);
    input.denied = true;
    assert!(
        run(
            &arguments(&["info", "sample.opd"]),
            &mut input,
            &mut Vec::new()
        )
        .unwrap_err()
        .contains("package read denied")
    );
    input.denied = false;
    input.encoded = b"not a package".to_vec();
    assert!(
        run(
            &arguments(&["info", "sample.opd"]),
            &mut input,
            &mut Vec::new()
        )
        .unwrap_err()
        .contains("info:")
    );
    assert!(
        run(
            &arguments(&["info", "sample.opd"]),
            &mut reader(1),
            &mut RefusingOutput
        )
        .unwrap_err()
        .contains("output closed")
    );
    assert!(
        run(&arguments(&["help"]), &mut reader(1), &mut RefusingOutput)
            .unwrap_err()
            .contains("output closed")
    );
}

#[test]
fn help_and_version_match_the_embedded_build_identity_without_reading_packages() {
    let identity = BuildIdentity::parse(BUILD_MANIFEST).unwrap();
    for command in ["version", "--version", "help", "-h", "--help"] {
        let mut input = reader(1);
        let mut output = Vec::new();
        run(&arguments(&[command]), &mut input, &mut output).unwrap();
        let output = String::from_utf8(output).unwrap();
        assert!(input.paths.is_empty());
        if command.contains("version") {
            assert_eq!(
                output,
                format!(
                    "oblikovati-cli {} (commit {})\n",
                    identity.version, identity.application_commit
                )
            );
        } else {
            assert!(
                output.contains("Kernel geometry replay and import/export are not yet available")
            );
        }
    }
}

#[test]
fn build_manifest_projection_validates_version_format_and_commit() {
    let valid = serde_json::json!({"format":1, "version":"0.26010.2.7", "application_commit":"a".repeat(40), "dependencies":{"api":"retained in full manifest"}}).to_string();
    assert_eq!(BuildIdentity::parse(&valid).unwrap().version, "0.26010.2.7");
    for invalid in [
        "{}",
        "not json",
        r#"{"format":2,"version":"v","application_commit":"unknown"}"#,
        r#"{"format":1,"version":"","application_commit":"unknown"}"#,
        r#"{"format":1,"version":"v","application_commit":"bad"}"#,
    ] {
        assert!(BuildIdentity::parse(invalid).is_err());
    }
}
