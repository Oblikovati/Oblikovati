// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_head::{HeadCommand, HeadOptions, parse_arguments, run};
use std::ffi::OsString;
#[test]
fn head_arguments_match_default_and_bounded_smoke_modes() {
    assert_eq!(
        parse_arguments(&[]).unwrap(),
        HeadCommand::Window(HeadOptions::default())
    );
    for flag in ["--frames", "-frames"] {
        assert_eq!(
            parse_arguments(&[flag.into(), "3".into()]).unwrap(),
            HeadCommand::Window(HeadOptions { frames: Some(3) })
        );
        assert_eq!(
            parse_arguments(&[flag.into(), "0".into()]).unwrap(),
            HeadCommand::Window(HeadOptions::default())
        );
    }
}
#[test]
fn head_help_and_version_aliases_are_parsed_without_window_options() {
    assert_eq!(parse_arguments(&["-h".into()]).unwrap(), HeadCommand::Help);
    assert_eq!(
        parse_arguments(&["version".into()]).unwrap(),
        HeadCommand::Version
    );
}
#[test]
fn malformed_smoke_counts_and_unknown_operands_are_refused_before_window_creation() {
    for count in ["", "-1", "+1", "1.5", " 3", "18446744073709551616"] {
        assert!(
            parse_arguments(&["--frames".into(), count.into()])
                .unwrap_err()
                .contains("unsigned 64-bit")
        );
    }
    for operands in [vec!["--frames"], vec!["file.opd"], vec!["--help", "extra"]] {
        let arguments: Vec<OsString> = operands.into_iter().map(OsString::from).collect();
        assert!(
            parse_arguments(&arguments)
                .unwrap_err()
                .contains("expected")
        );
    }
}
#[test]
fn help_and_version_use_injected_output_and_common_build_identity() {
    let mut output = Vec::new();
    run(&["--help".into()], &mut output).unwrap();
    assert!(
        String::from_utf8(output.clone())
            .unwrap()
            .contains("--frames N")
    );
    output.clear();
    run(&["--version".into()], &mut output).unwrap();
    let identity = oblikovati_cli::BuildIdentity::parse(oblikovati_cli::BUILD_MANIFEST).unwrap();
    assert!(
        String::from_utf8(output)
            .unwrap()
            .contains(&identity.version)
    );
}
struct BrokenHeadOutput;
impl std::io::Write for BrokenHeadOutput {
    fn write(&mut self, _: &[u8]) -> std::io::Result<usize> {
        Err(std::io::Error::other("named output fault"))
    }
    fn flush(&mut self) -> std::io::Result<()> {
        Ok(())
    }
}
#[test]
fn output_failures_are_named() {
    assert!(
        run(&["--help".into()], &mut BrokenHeadOutput)
            .unwrap_err()
            .contains("writable stream")
    );
}
