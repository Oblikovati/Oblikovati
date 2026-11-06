// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::HeadWork;
use crate::native::addin_services_tests::isolation;
use std::{
    cell::RefCell,
    path::{Path, PathBuf},
    rc::Rc,
};
#[derive(Default)]
struct CaptureFileRecord {
    attempts: Vec<PathBuf>,
    images: Vec<oblikovati_viewport::ViewportCapture>,
}
struct RecordedCaptureFiles {
    record: Rc<RefCell<CaptureFileRecord>>,
    refuse: bool,
}
impl crate::WindowCaptureFiles for RecordedCaptureFiles {
    fn write_capture(
        &mut self,
        path: &Path,
        image: &oblikovati_viewport::ViewportCapture,
    ) -> Result<(), String> {
        self.record.borrow_mut().attempts.push(path.into());
        if self.refuse {
            return Err(format!(
                "fake capture path {path:?}: expected a writable destination"
            ));
        }
        self.record.borrow_mut().images.push(image.clone());
        Ok(())
    }
}
fn recorded_window(refuse: bool) -> (NativeHeadWindow, Rc<RefCell<CaptureFileRecord>>) {
    let record = Rc::new(RefCell::new(CaptureFileRecord::default()));
    let files = RecordedCaptureFiles {
        record: record.clone(),
        refuse,
    };
    (
        NativeHeadWindow::open_with_capture_files(Box::new(files)).unwrap(),
        record,
    )
}
fn run_capture_frames(window: &mut NativeHeadWindow) {
    crate::frames::run_frames(
        window,
        crate::HeadOptions { frames: Some(3) },
        &mut crate::work::IdleHeadWork,
    )
    .unwrap();
}

#[test]
fn capture_api_transfers_owner_requests_to_the_next_completed_native_frame() {
    let case = "native::capture::tests::capture_api_transfers_owner_requests_to_the_next_completed_native_frame";
    if isolation::case("OBLIKOVATI_CAPTURE_OWNER_FRAME", case) {
        return;
    }
    let (mut window, record) = recorded_window(false);
    let mut owner = native_owner(window.wake.clone()).unwrap();
    crate::frames::run_frames(
        &mut window,
        crate::HeadOptions { frames: Some(3) },
        &mut owner,
    )
    .unwrap();
    let reply = oblikovati_api::client::Client::new(owner.document_host())
        .view()
        .capture_window(&oblikovati_api::wire::CaptureWindowArgs {
            path: "owner.png".into(),
        })
        .unwrap();
    assert!(record.borrow().attempts.is_empty());
    crate::frames::run_frames(
        &mut window,
        crate::HeadOptions { frames: Some(3) },
        &mut owner,
    )
    .unwrap();
    assert_eq!(record.borrow().attempts, [PathBuf::from(&reply.path)]);
    let extent = record.borrow().images[0].extent.map(i64::from);
    assert_eq!(extent, [reply.width, reply.height]);
    owner.finish().unwrap();
    window.finish().unwrap();
}
#[test]
fn completed_window_capture_contains_gui_pixels_and_retires_the_batch_in_order() {
    let case = "native::capture::tests::completed_window_capture_contains_gui_pixels_and_retires_the_batch_in_order";
    if isolation::case("OBLIKOVATI_COMPLETED_WINDOW_CAPTURE", case) {
        return;
    }
    let (mut window, record) = recorded_window(false);
    let extent = window.window_capture_extent().unwrap().unwrap();
    let paths: Vec<PathBuf> = ["first.png", "second.png"].map(PathBuf::from).into();
    window
        .request_window_captures(
            paths
                .iter()
                .map(|path| crate::WindowCaptureJob {
                    path: path.clone(),
                    extent,
                })
                .collect(),
        )
        .unwrap();
    assert!(!window.window_capture_ready());
    run_capture_frames(&mut window);
    assert!(window.window_capture_ready());
    assert_eq!(record.borrow().attempts, paths);
    let images = &record.borrow().images;
    assert_eq!(images.len(), 2);
    assert_eq!(images[0], images[1]);
    assert_eq!(images[0].extent, extent);
    assert!(
        images[0]
            .rgba
            .as_chunks::<4>()
            .0
            .iter()
            .any(|pixel| pixel[..3] != [0; 3])
    );
    window.finish().unwrap();
}
#[test]
fn capture_resize_and_file_failures_are_local_visible_and_leave_the_window_live() {
    let case = "native::capture::tests::capture_resize_and_file_failures_are_local_visible_and_leave_the_window_live";
    if isolation::case("OBLIKOVATI_CAPTURE_FAILURE", case) {
        return;
    }
    for refuse in [false, true] {
        assert_local_capture_decline(refuse);
    }
}
fn assert_local_capture_decline(refuse: bool) {
    let (mut window, record) = recorded_window(refuse);
    let actual = window.window_capture_extent().unwrap().unwrap();
    let extent = if refuse { actual } else { [1, 1] };
    window
        .request_window_captures(vec![crate::WindowCaptureJob {
            path: "declined.png".into(),
            extent,
        }])
        .unwrap();
    run_capture_frames(&mut window);
    assert_eq!(record.borrow().attempts.len(), usize::from(refuse));
    assert!(record.borrow().images.is_empty());
    let notice = window.command_notice.as_ref().unwrap().message();
    assert!(
        notice.contains("declined.png")
            && notice.contains(if refuse {
                "writable destination"
            } else {
                "drawable size"
            })
    );
    assert!(window.window_capture_ready());
    assert_eq!(
        window.present().unwrap(),
        crate::HeadPresentation::Completed
    );
    window.finish().unwrap();
}
#[test]
fn a_second_or_oversized_batch_cannot_replace_accepted_capture_work() {
    let case =
        "native::capture::tests::a_second_or_oversized_batch_cannot_replace_accepted_capture_work";
    if isolation::case("OBLIKOVATI_CAPTURE_ADMISSION", case) {
        return;
    }
    let (mut window, record) = recorded_window(false);
    let extent = window.window_capture_extent().unwrap().unwrap();
    let job = crate::WindowCaptureJob {
        path: "accepted.png".into(),
        extent,
    };
    assert!(
        window
            .request_window_captures(vec![
                job.clone();
                crate::window_capture::WINDOW_CAPTURE_QUEUE_CAPACITY + 1
            ])
            .is_err()
    );
    window.request_window_captures(vec![job.clone()]).unwrap();
    assert!(window.request_window_captures(vec![job]).is_err());
    run_capture_frames(&mut window);
    assert_eq!(record.borrow().attempts, [PathBuf::from("accepted.png")]);
    window.finish().unwrap();
}
