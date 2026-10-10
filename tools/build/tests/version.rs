// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::process::tests::FakeRunner;

#[test]
fn conventional_commits_match_the_legacy_policy() {
    for message in ["feat(ui): title", "FEAT: X"] {
        assert_eq!(classify(message), Scope::Feature);
    }
    for message in [
        "fix: X",
        "perf: X",
        "revert: X",
        "wire: X",
        "untyped subject",
    ] {
        assert_eq!(classify(message), Scope::Patch);
    }
    for kind in ["docs", "chore", "ci", "test", "style", "build", "refactor"] {
        assert_eq!(classify(&format!("{kind}: X")), Scope::None);
    }
    for message in [
        "feat!: X",
        "fix(app)!: X",
        "fix: X\n\nBREAKING CHANGE: Y",
        "fix: X\nBREAKING-CHANGE: Y",
    ] {
        assert_eq!(classify(message), Scope::Breaking);
    }
    assert_eq!(bump(1, 4, "docs: X\0fix: Y\0feat: Z"), Ok((2, 0)));
    assert_eq!(bump(1, 4, "fix: X"), Ok((1, 5)));
    assert_eq!(bump(1, 4, ""), Ok((1, 4)));
    assert!(bump(1, u64::MAX, "fix: X").is_err());
}

#[test]
fn api_fields_and_tag_validation_do_not_assume_two_digit_limits() {
    for (api, want) in [
        ("v0.2.0", "000200"),
        (" 0.12.3 ", "001203"),
        ("0.154.0", "0015400"),
    ] {
        assert_eq!(api_field(api).unwrap(), want);
    }
    for api in [
        "0.2",
        "0.x.0",
        "0.-2.0",
        "0.2.0-beta",
        "0..0",
        "0.+2.0",
        "999999999999999999999.0.0",
    ] {
        assert!(api_field(api).is_err(), "{api}");
    }
    for tag in [
        "v0.000200.1.0",
        "rust-v0.000200.1.0-rc",
        "rust-v0.2.1",
        "rust-v0.2.1.0.1",
    ] {
        assert_eq!(tag_fields(tag), None);
    }
    assert_eq!(tag_fields("rust-v0.000200.1.4"), Some([0, 200, 1, 4]));
}

#[test]
fn versions_reset_on_api_or_major_changes_and_ignore_go_tags() {
    let cases = [
        ("", "feat: X", 0, "0.000200.1.0"),
        ("rust-v0.000200.1.4", "feat: X", 0, "0.000200.2.0"),
        (
            "rust-v0.000200.1.4\nrust-v0.000200.1.3",
            "fix: X",
            0,
            "0.000200.1.5",
        ),
        ("rust-v0.000100.5.2", "fix: X", 0, "0.000200.0.1"),
        ("rust-v0.000200.5.2", "fix: X", 1, "1.000200.0.1"),
        (
            "v9.999999.99.99\nrust-v0.000200.1.9-beta",
            "docs: X",
            0,
            "0.000200.0.0",
        ),
    ];
    for (tags, messages, major, want) in cases {
        let runner = FakeRunner::default();
        runner
            .replies
            .borrow_mut()
            .extend([Ok(tags.into()), Ok(messages.into())]);
        let now = "2026-06-15T03:47:51Z".parse().unwrap();
        assert_eq!(
            compute(Path::new("."), &runner, major, "0.2.0", "stable", now).unwrap(),
            want
        );
    }
}

#[test]
fn nightly_stamp_and_git_failures_are_explicit() {
    let runner = FakeRunner::default();
    let now = "2026-06-15T03:47:51Z".parse().unwrap();
    assert_eq!(
        compute(Path::new("."), &runner, 0, "0.2.0", "nightly", now).unwrap(),
        "0.000200.0.0-nightly.26061503"
    );
    assert!(compute(Path::new("."), &runner, 0, "0.2.0", "beta", now).is_err());
    runner
        .replies
        .borrow_mut()
        .push_back(Err("git unavailable".into()));
    assert!(
        compute(Path::new("."), &runner, 0, "0.2.0", "stable", now)
            .unwrap_err()
            .contains("git unavailable")
    );
}
