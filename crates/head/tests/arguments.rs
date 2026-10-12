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
    for operands in [vec!["--frames"], vec!["--unknown"], vec!["--help", "extra"]] {
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

#[test]
fn startup_paths_and_frame_options_preserve_order_and_spaces() {
    let arguments = ["first part.opd", "--frames=3", "second.opd"].map(OsString::from);
    let expected = HeadCommand::OpenDocuments {
        options: HeadOptions { frames: Some(3) },
        documents: ["first part.opd", "second.opd"]
            .map(std::path::PathBuf::from)
            .into(),
    };
    assert_eq!(parse_arguments(&arguments).unwrap(), expected);
    assert_eq!(
        parse_arguments(&["-frames=3".into()]).unwrap(),
        HeadCommand::Window(HeadOptions { frames: Some(3) })
    );
}
#[test]
fn literal_separator_accepts_leading_dash_names_without_interpreting_flags() {
    let arguments = ["--frames", "1", "--", "--help", "-part.opd"].map(OsString::from);
    let command = parse_arguments(&arguments).unwrap();
    let HeadCommand::OpenDocuments { options, documents } = command else {
        panic!("expected startup documents")
    };
    assert_eq!(options.frames, Some(1));
    assert_eq!(
        documents,
        [
            std::path::PathBuf::from("--help"),
            std::path::PathBuf::from("-part.opd")
        ]
    );
    assert_eq!(
        parse_arguments(&["--".into()]).unwrap(),
        HeadCommand::Window(HeadOptions::default())
    );
}
#[cfg(unix)]
#[test]
fn duplicate_options_and_invalid_document_operands_are_refused_without_io() {
    use std::os::unix::ffi::OsStringExt;
    let duplicate = ["--frames", "1", "--frames=2"].map(OsString::from);
    assert!(
        parse_arguments(&duplicate)
            .unwrap_err()
            .contains("only one --frames")
    );
    for operand in [OsString::new(), OsString::from_vec(vec![0xff])] {
        assert!(
            parse_arguments(&[operand])
                .unwrap_err()
                .contains("nonempty UTF-8")
        );
    }
}
