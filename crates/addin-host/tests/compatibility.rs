// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_addin_host::check_api_version;
#[test]
fn matching_and_older_minor_versions_are_admitted_including_zero_major() {
    for (reported, host) in [((1, 4), (1, 4)), ((1, 2), (1, 4)), ((0, 0), (0, 1))] {
        assert_eq!(check_api_version(Some(reported), host).unwrap(), reported);
    }
}
#[test]
fn missing_newer_major_newer_minor_and_negative_versions_are_named_declines() {
    for reported in [
        None,
        Some((2, 0)),
        Some((1, 5)),
        Some((-1, 0)),
        Some((1, -1)),
    ] {
        let error = check_api_version(reported, (1, 4)).unwrap_err().to_string();
        assert!(error.contains("expected"));
        assert!(error.contains("host 1.4"));
    }
}

#[test]
fn invalid_host_versions_cannot_authorize_a_library() {
    for host in [(-1, 0), (0, -1)] {
        let error = check_api_version(Some((0, 0)), host)
            .unwrap_err()
            .to_string();
        assert!(error.contains("expected a nonnegative host version"));
    }
}
