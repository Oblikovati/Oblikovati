// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
#![allow(unsafe_code)]
#[path = "support/compiler.rs"]
mod compiler;
use compiler::FixtureCompiler;
use oblikovati_addin_host::{AddInProbe, probe_library};
use oblikovati_api::{API_MAJOR, API_MINOR};
use std::path::{Path, PathBuf};
fn probe(path: &Path) -> Result<AddInProbe, oblikovati_api::Error> {
    // SAFETY: these fixtures are built here from trusted source with the pinned ABI prototypes.
    unsafe { probe_library(path) }
}
#[test]
fn rust_library_reports_copied_metadata_and_optional_automation() {
    let compiler = FixtureCompiler::new();
    for (name, flags, automation) in [("plain", vec![], false), ("auto", vec!["automation"], true)]
    {
        let library = compiler.rust_library(name, &flags, API_MAJOR, API_MINOR);
        let metadata = probe(&library).unwrap();
        assert_eq!(metadata.id, "rust-fixture");
        assert_eq!(metadata.manifest, r#"{"id":"rust-fixture"}"#);
        assert_eq!((metadata.major, metadata.minor), (API_MAJOR, API_MINOR));
        assert_eq!(metadata.has_automation, automation);
    }
}
#[test]
fn go_reference_library_passes_the_existing_older_minor_handshake() {
    let compiler = FixtureCompiler::new();
    let reference = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("../../../Oblikovati/head/internal/addinhost/testdata/echoaddin/main.go");
    let library = compiler.go_library(&reference);
    let metadata = probe(&library).unwrap();
    assert_eq!(metadata.id, "com.oblikovati.echo-fixture");
    assert!(metadata.has_automation);
    assert_eq!((metadata.major, metadata.minor), (0, 0));
    assert!(metadata.manifest.contains("Echo Fixture"));
}
#[test]
fn missing_exports_and_invalid_versions_are_named_before_metadata_is_accepted() {
    let compiler = FixtureCompiler::new();
    for (name, flags, major, minor, expected) in REJECTION_CASES {
        let library = compiler.rust_library(name, flags, major, minor);
        let error = probe(&library).unwrap_err().to_string();
        assert!(error.contains(expected), "{name}: {error}");
    }
}
const REJECTION_CASES: [(&str, &[&str], i32, i32, &str); 5] = [
    (
        "missing",
        &["missing_major"],
        API_MAJOR,
        API_MINOR,
        "ObkAddInApiMajor",
    ),
    (
        "required",
        &["missing_notify"],
        API_MAJOR,
        API_MINOR,
        "ObkAddInNotify",
    ),
    (
        "major",
        &[],
        API_MAJOR + 1,
        0,
        "expected nonnegative version",
    ),
    ("newer", &[], API_MAJOR, API_MINOR + 1, "minor <="),
    (
        "negative",
        &[],
        API_MAJOR,
        -1,
        "expected nonnegative version",
    ),
];
#[test]
fn null_and_non_utf8_identity_are_refused_with_the_offending_export() {
    let compiler = FixtureCompiler::new();
    for (name, flag, expected) in [
        ("null", "null_id", "returned null"),
        ("utf8", "invalid_utf8", "expected UTF-8"),
    ] {
        let library = compiler.rust_library(name, &[flag], API_MAJOR, API_MINOR);
        let error = probe(&library).unwrap_err().to_string();
        assert!(error.contains("ObkAddInId"));
        assert!(error.contains(expected));
    }
}
#[test]
fn nonexistent_library_and_embedded_nul_paths_are_named_loader_declines() {
    for path in [
        Path::new("/nonexistent/oblikovati.so"),
        Path::new("bad\0path"),
    ] {
        let error = probe(path).unwrap_err().to_string();
        assert!(error.contains("add-in path"));
        assert!(error.contains("expected"));
    }
}
