// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn capture_admission_requires_a_current_drawable_and_keeps_failures_out_of_the_queue() {
    let mut captures = DesktopWindowCaptures::new("captures".into(), 8);
    for extent in [None, Some([0, 10]), Some([10, 0]), Some([u32::MAX, 1])] {
        captures.observe_extent(extent);
        assert!(
            captures
                .request_window_capture(&Default::default())
                .unwrap_err()
                .contains("expected an observed positive")
        );
        assert!(captures.take_pending().is_empty());
        assert_eq!(captures.sequence, 0);
    }
}
#[test]
fn capture_paths_are_unique_portable_and_keep_request_order_and_extent() {
    let mut captures = DesktopWindowCaptures::new("captures".into(), 8);
    captures.observe_extent(Some([100, 80]));
    let first = captures
        .request_window_capture(&Default::default())
        .unwrap();
    let second = captures
        .request_window_capture(&Default::default())
        .unwrap();
    assert_ne!(first.path, second.path);
    assert_eq!((first.width, first.height), (100, 80));
    assert_eq!(
        PathBuf::from(&first.path),
        PathBuf::from("captures").join("oblikovati-window-8-1.png")
    );
    let requested = captures
        .request_window_capture(&CaptureWindowArgs {
            path: "chosen.png".into(),
        })
        .unwrap();
    let jobs = captures.take_pending();
    assert_eq!(
        jobs.iter().map(|job| job.path.clone()).collect::<Vec<_>>(),
        vec![
            PathBuf::from(first.path),
            PathBuf::from(second.path),
            PathBuf::from(requested.path)
        ]
    );
    assert!(jobs.iter().all(|job| job.extent == [100, 80]));
    assert!(captures.take_pending().is_empty());
}
#[test]
fn full_queue_invalid_paths_and_sequence_exhaustion_refuse_without_adopting_work() {
    let mut captures = DesktopWindowCaptures::new("captures".into(), 8);
    captures.observe_extent(Some([100, 80]));
    assert!(
        captures
            .request_window_capture(&CaptureWindowArgs {
                path: "bad\0path".into()
            })
            .is_err()
    );
    for _ in 0..WINDOW_CAPTURE_QUEUE_CAPACITY {
        captures
            .request_window_capture(&Default::default())
            .unwrap();
    }
    assert!(
        captures
            .request_window_capture(&Default::default())
            .unwrap_err()
            .contains("pending requests")
    );
    assert_eq!(captures.take_pending().len(), WINDOW_CAPTURE_QUEUE_CAPACITY);
    assert!(captures.request_window_capture(&Default::default()).is_ok());
    captures.take_pending();
    captures.sequence = u64::MAX;
    assert!(
        captures
            .request_window_capture(&Default::default())
            .unwrap_err()
            .contains("unused u64")
    );
    assert!(captures.take_pending().is_empty());
}
#[cfg(unix)]
#[test]
fn non_utf8_default_directory_is_declined_before_queue_adoption() {
    use std::{ffi::OsString, os::unix::ffi::OsStringExt};
    let directory = PathBuf::from(OsString::from_vec(vec![255]));
    let mut captures = DesktopWindowCaptures::new(directory, 8);
    captures.observe_extent(Some([100, 80]));
    assert!(
        captures
            .request_window_capture(&Default::default())
            .unwrap_err()
            .contains("expected UTF-8")
    );
    assert!(captures.take_pending().is_empty());
    assert_eq!(captures.sequence, 0);
}
