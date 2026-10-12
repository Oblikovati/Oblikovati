// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::run_window_with_owner;
use oblikovati_api::{client::Client, wire::*};
use std::sync::mpsc::{self, Receiver, TryRecvError};
struct WorkflowWindow {
    native: NativeHeadWindow,
    done: Receiver<Result<u64, String>>,
    reopened: Option<u64>,
}
impl HeadWindow for WorkflowWindow {
    fn events(&mut self, limit: usize) -> Result<Vec<ViewportEvent>, String> {
        let mut events = self.native.events(limit)?;
        match self.done.try_recv() {
            Ok(result) => {
                self.reopened = Some(result?);
                events.push(ViewportEvent::CloseRequested);
            }
            Err(TryRecvError::Empty) => {}
            Err(TryRecvError::Disconnected) => {
                return Err("native workflow disconnected: expected completion".into());
            }
        }
        Ok(events)
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        self.native.present()
    }
    fn wait(&mut self, timeout: Option<std::time::Duration>) -> Result<bool, String> {
        self.native.wait(timeout)
    }
    fn finish(&mut self) -> Result<(), String> {
        self.native.finish()
    }
}
#[test]
fn native_frames_drain_typed_worker_document_calls_and_wake_after_completion() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("native-owner.opd");
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    let (done, completed) = mpsc::channel();
    let worker = start_workflow(&owner, &native, path.to_str().unwrap().to_owned(), done);
    let mut window = WorkflowWindow {
        native,
        done: completed,
        reopened: None,
    };
    let summary = run_window_with_owner(&mut window, HeadOptions::default(), &mut owner).unwrap();
    worker.join().unwrap();
    verify_native_owner(&owner, window.reopened.unwrap(), &path, summary);
}
fn start_workflow(
    owner: &NativeDesktopOwner,
    native: &NativeHeadWindow,
    name: String,
    done: mpsc::Sender<Result<u64, String>>,
) -> std::thread::JoinHandle<()> {
    let caller = owner.caller();
    let wake = native.wake.clone();
    std::thread::spawn(move || {
        done.send(document_workflow(Client::new(caller), name))
            .unwrap();
        wake.wake().unwrap();
    })
}
fn document_workflow(
    client: Client<oblikovati_automation::dispatch::QueuedCaller<NativeOwnerWake>>,
    path: String,
) -> Result<u64, String> {
    let document = client
        .documents()
        .create(&CreateDocumentArgs {
            type_: "part".into(),
            name: "Native owner part".into(),
            ..Default::default()
        })
        .map_err(|error| error.to_string())?;
    save_and_close(&client, document.id, &path)?;
    client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path,
            ..Default::default()
        })
        .map(|reply| reply.id)
        .map_err(|error| error.to_string())
}
fn save_and_close(
    client: &Client<oblikovati_automation::dispatch::QueuedCaller<NativeOwnerWake>>,
    document: u64,
    path: &str,
) -> Result<(), String> {
    client
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: path.into(),
        })
        .map_err(|error| error.to_string())?;
    client
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: false,
        })
        .map_err(|error| error.to_string())?;
    Ok(())
}
fn verify_native_owner(
    owner: &NativeDesktopOwner,
    document: u64,
    path: &std::path::Path,
    summary: HeadSummary,
) {
    assert!(summary.completed > 0);
    assert_eq!(owner.documents().workspace().documents(), [document]);
    assert!(!owner.documents().workspace().needs_save(document).unwrap());
    assert_eq!(owner.documents().path(document).unwrap(), path);
    assert!(path.is_file());
    assert!(owner.refused_close().is_empty());
}
