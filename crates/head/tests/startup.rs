// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{HeadPresentation, HeadWork, HeadWorkStatus};
use oblikovati_api::{
    client::Caller,
    wire::{DocumentInfo, METHOD_DOCUMENTS_OPEN},
};
use oblikovati_viewport::ViewportEvent;
use std::{cell::RefCell, rc::Rc};
#[derive(Default)]
struct StartupReceipt {
    opened: Vec<OpenDocumentArgs>,
    retirements: Vec<&'static str>,
    presents: usize,
    failed_path: Option<String>,
}
#[derive(Clone)]
struct FakeStartupCaller {
    receipt: Rc<RefCell<StartupReceipt>>,
}
impl Caller for FakeStartupCaller {
    fn call(&self, method: &str, bytes: &[u8]) -> Result<Vec<u8>, oblikovati_api::Error> {
        assert_eq!(method, METHOD_DOCUMENTS_OPEN);
        let request: OpenDocumentArgs = serde_json::from_slice(bytes).unwrap();
        let mut receipt = self.receipt.borrow_mut();
        receipt.opened.push(request.clone());
        if receipt.failed_path.as_deref() == Some(&request.full_document_name) {
            return Err(oblikovati_api::Error("named startup read failure".into()));
        }
        serde_json::to_vec(&DocumentInfo {
            id: receipt.opened.len() as u64,
            ..Default::default()
        })
        .map_err(|cause| oblikovati_api::Error(cause.to_string()))
    }
}
struct FakeStartupOwner {
    caller: FakeStartupCaller,
}
impl ScriptDocumentOwner for FakeStartupOwner {
    type Host<'a> = FakeStartupCaller;
    fn script_host(&mut self) -> Self::Host<'_> {
        self.caller.clone()
    }
}
impl HeadWork for FakeStartupOwner {
    fn poll(&mut self, _: usize) -> Result<HeadWorkStatus, String> {
        Ok(Default::default())
    }
    fn close_allowed(&mut self) -> Result<bool, String> {
        Ok(true)
    }
    fn finish(&mut self) -> Result<(), String> {
        self.caller.receipt.borrow_mut().retirements.push("owner");
        Ok(())
    }
}
struct FakeStartupWindow {
    receipt: Rc<RefCell<StartupReceipt>>,
}
impl HeadWindow for FakeStartupWindow {
    fn events(&mut self, _: usize) -> Result<Vec<ViewportEvent>, String> {
        Ok(if self.receipt.borrow().presents > 0 {
            vec![ViewportEvent::CloseRequested]
        } else {
            Vec::new()
        })
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        self.receipt.borrow_mut().presents += 1;
        Ok(HeadPresentation::Completed)
    }
    fn wait(&mut self, _: Option<std::time::Duration>) -> Result<bool, String> {
        Ok(false)
    }
    fn finish(&mut self) -> Result<(), String> {
        self.receipt.borrow_mut().retirements.push("window");
        Ok(())
    }
}
fn launch_fixture(failed_path: Option<&str>) -> (FakeStartupOwner, FakeStartupWindow) {
    let receipt = Rc::new(RefCell::new(StartupReceipt {
        failed_path: failed_path.map(str::to_owned),
        ..Default::default()
    }));
    (
        FakeStartupOwner {
            caller: FakeStartupCaller {
                receipt: receipt.clone(),
            },
        },
        FakeStartupWindow { receipt },
    )
}
#[test]
fn startup_files_use_typed_visible_open_requests_before_the_first_frame() {
    let (mut owner, mut window) = launch_fixture(None);
    let paths = [PathBuf::from("first part.opd"), PathBuf::from("second.opd")];
    let summary = run_window_with_documents(
        &mut window,
        HeadOptions { frames: Some(1) },
        &mut owner,
        &paths,
    )
    .unwrap();
    assert_eq!(summary.completed, 1);
    check_open_receipt(&window.receipt.borrow());
}
fn check_open_receipt(receipt: &StartupReceipt) {
    let names: Vec<_> = receipt
        .opened
        .iter()
        .map(|request| request.full_document_name.as_str())
        .collect();
    assert_eq!(names, ["first part.opd", "second.opd"]);
    assert!(
        receipt
            .opened
            .iter()
            .all(|request| request.visible && !request.defer_content)
    );
    assert_eq!(receipt.retirements, ["owner", "window"]);
}
#[test]
fn startup_failure_stops_later_opens_and_retires_without_presenting_a_frame() {
    let (mut owner, mut window) = launch_fixture(Some("broken.opd"));
    let paths = ["first.opd", "broken.opd", "later.opd"].map(PathBuf::from);
    let error = run_window_with_documents(
        &mut window,
        HeadOptions { frames: Some(1) },
        &mut owner,
        &paths,
    )
    .unwrap_err();
    assert!(error.contains("broken.opd") && error.contains("named startup read failure"));
    let receipt = window.receipt.borrow();
    assert_eq!(receipt.opened.len(), 2);
    assert_eq!(receipt.presents, 0);
    assert_eq!(receipt.retirements, ["owner", "window"]);
}
#[test]
fn empty_or_non_utf8_startup_names_are_named_declines() {
    use std::os::unix::ffi::OsStringExt;
    for path in [
        PathBuf::new(),
        PathBuf::from(std::ffi::OsString::from_vec(vec![0xff])),
    ] {
        let error = startup_open_request(&path).unwrap_err();
        assert!(error.contains("expected a nonempty UTF-8 path"));
        let (mut owner, mut window) = launch_fixture(None);
        let options = HeadOptions { frames: Some(1) };
        assert!(run_window_with_documents(&mut window, options, &mut owner, &[path]).is_err());
        assert!(window.receipt.borrow().opened.is_empty());
        assert_eq!(window.receipt.borrow().retirements, ["owner", "window"]);
    }
}

#[test]
fn startup_with_no_paths_uses_the_same_idle_wait_and_close_policy() {
    let (mut owner, mut window) = launch_fixture(None);
    let summary =
        run_window_with_documents(&mut window, HeadOptions::default(), &mut owner, &[]).unwrap();
    assert_eq!(summary.completed, 1);
    assert!(window.receipt.borrow().opened.is_empty());
    assert_eq!(window.receipt.borrow().retirements, ["owner", "window"]);
}
