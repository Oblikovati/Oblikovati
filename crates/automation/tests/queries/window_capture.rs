// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_automation::{FileHost, WINDOW_CAPTURE_METHODS, WindowCaptureAccess};
#[derive(Default)]
struct RecordedCaptureOwner {
    requests: Vec<CaptureWindowArgs>,
    refuse: bool,
}
impl WindowCaptureAccess for RecordedCaptureOwner {
    fn request_window_capture(
        &mut self,
        args: &CaptureWindowArgs,
    ) -> Result<CaptureWindowResult, String> {
        if self.refuse {
            return Err(format!(
                "capture {:?}: expected a ready fake drawable",
                args.path
            ));
        }
        self.requests.push(args.clone());
        Ok(CaptureWindowResult {
            path: args.path.clone(),
            width: 123,
            height: 87,
        })
    }
}
#[test]
fn typed_capture_uses_the_injected_view_owner_without_editing_document_history() {
    let (mut files, document, mut source) =
        super::sketch_api::configured_files(DOCUMENT_PART, Default::default());
    let before = super::sketch_api::owner_revision(&files, document);
    let mut captures = RecordedCaptureOwner::default();
    let args = CaptureWindowArgs {
        path: "chosen.png".into(),
    };
    let reply = Client::new(
        FileHost::new(&mut files, &mut source, "test").with_window_capture(&mut captures),
    )
    .view()
    .capture_window(&args)
    .unwrap();
    assert_eq!(
        reply,
        CaptureWindowResult {
            path: "chosen.png".into(),
            width: 123,
            height: 87
        }
    );
    assert_eq!(captures.requests, [args]);
    assert_eq!(before, super::sketch_api::owner_revision(&files, document));
    assert!(WINDOW_CAPTURE_METHODS.contains(&METHOD_VIEWPORT_CAPTURE_WINDOW));
    assert!(
        oblikovati_automation::file_host_methods()
            .iter()
            .any(|method| method == METHOD_VIEWPORT_CAPTURE_WINDOW)
    );
}
#[test]
fn capture_without_a_document_still_requires_a_live_view_owner_and_valid_arguments() {
    let mut files = DiscoveryWorkspace::new_with_sketch_options(
        QueryObserver::default(),
        DiscoveryFiles::default(),
        Default::default(),
    );
    let mut source = QueryGuids(0);
    let mut captures = RecordedCaptureOwner::default();
    assert!(
        Client::new(
            FileHost::new(&mut files, &mut source, "test").with_window_capture(&mut captures)
        )
        .view()
        .capture_window(&Default::default())
        .is_ok()
    );
    let host = FileHost::new(&mut files, &mut source, "test").with_window_capture(&mut captures);
    assert!(
        host.call(METHOD_VIEWPORT_CAPTURE_WINDOW, b"{\"path\":5}")
            .is_err()
    );
}
#[test]
fn missing_or_refusing_view_owners_return_named_errors_without_queue_adoption() {
    let (mut files, _, mut source) =
        super::sketch_api::configured_files(DOCUMENT_PART, Default::default());
    let args = CaptureWindowArgs {
        path: "chosen.png".into(),
    };
    let cause = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .view()
        .capture_window(&args)
        .unwrap_err();
    assert!(cause.0.contains("chosen.png") && cause.0.contains("injected live desktop"));
    let mut captures = RecordedCaptureOwner {
        refuse: true,
        ..Default::default()
    };
    let cause = Client::new(
        FileHost::new(&mut files, &mut source, "test").with_window_capture(&mut captures),
    )
    .view()
    .capture_window(&args)
    .unwrap_err();
    assert!(cause.0.contains("ready fake drawable"));
    assert!(captures.requests.is_empty());
}
