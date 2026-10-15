// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::DesktopObservations;
use oblikovati_api::{client::Client, wire::*};
use std::{
    sync::mpsc::{self, Receiver, Sender},
    thread,
    time::Duration,
};
#[path = "owner/files.rs"]
mod files;
use files::{MemoryDocumentFiles, OwnerThreadGuids};
#[derive(Clone)]
struct RecordedWake(Sender<()>);
impl DispatchWakeup for RecordedWake {
    fn wake(&self) -> Result<(), String> {
        self.0.send(()).map_err(|error| error.to_string())
    }
}
type TestOwner = DesktopOwner<
    DesktopObservations,
    DesktopObservations,
    MemoryDocumentFiles,
    OwnerThreadGuids,
    DesktopObservations,
    RecordedWake,
>;
fn owner() -> (TestOwner, Receiver<()>) {
    owner_with_options(Default::default())
}
fn owner_with_options(
    options: oblikovati_solve::nonlinear::SolveOptions,
) -> (TestOwner, Receiver<()>) {
    let observations = DesktopObservations::default();
    let files = FileWorkspace::new_with_sketch_options(
        observations.clone(),
        MemoryDocumentFiles::default(),
        options,
    );
    let (signal, awake) = mpsc::channel();
    (
        DesktopOwner::new(
            files,
            OwnerThreadGuids::default(),
            observations,
            "alpha".into(),
            RecordedWake(signal),
            DesktopDispatchOptions::default(),
        ),
        awake,
    )
}
fn invoke<R: Send + 'static>(
    owner: &mut TestOwner,
    awake: &Receiver<()>,
    call: impl FnOnce(Client<QueuedCaller<RecordedWake>>) -> R + Send + 'static,
) -> R {
    let caller = owner.caller();
    let worker = thread::spawn(move || call(Client::new(caller)));
    awake.recv_timeout(Duration::from_secs(2)).unwrap();
    assert!(owner.poll(32).unwrap().changed);
    worker.join().unwrap()
}
fn create(owner: &mut TestOwner, awake: &Receiver<()>) -> u64 {
    invoke(owner, awake, |client| {
        client
            .documents()
            .create(&CreateDocumentArgs {
                type_: "part".into(),
                name: "Owner part".into(),
                ..Default::default()
            })
            .unwrap()
            .id
    })
}

#[test]
fn selection_filter_names_round_trip_to_the_host_kinds() {
    let filter = AmbientSelectionFilter::default();
    let view = gui_selection_filter(&filter);
    assert_eq!(view.entries.len(), 12);
    assert!(
        view.entries
            .iter()
            .all(|entry| selection_kind(&entry.kind).is_ok())
    );
    assert_eq!(
        selection_kind("not-a-kind").unwrap_err(),
        "selection kind \"not-a-kind\": expected a filterable desktop kind"
    );
}
#[test]
fn typed_workers_create_save_reopen_and_close_through_the_document_owner() {
    let (mut owner, awake) = owner();
    assert!(owner.close_allowed().unwrap());
    let id = create(&mut owner, &awake);
    assert!(!owner.close_allowed().unwrap());
    assert_eq!(owner.refused_close(), [id]);
    invoke(&mut owner, &awake, move |client| {
        client
            .documents()
            .save_as(&SaveDocumentAsArgs {
                document: id,
                new_full_document_name: "/memory/part.opd".into(),
            })
            .unwrap()
    });
    assert!(owner.close_allowed().unwrap());
    verify_transaction_close_guard(&mut owner, &awake);
    verify_close_and_reopen(&mut owner, &awake, id);
}
fn verify_close_and_reopen(owner: &mut TestOwner, awake: &Receiver<()>, id: u64) {
    invoke(owner, awake, move |client| {
        client
            .documents()
            .close(&CloseDocumentArgs { id, force: false })
            .unwrap()
    });
    assert!(owner.documents().workspace().documents().is_empty());
    verify_reopened_owner(owner, awake);
}
fn verify_reopened_owner(owner: &mut TestOwner, awake: &Receiver<()>) {
    let reopened = invoke(owner, awake, |client| {
        client
            .documents()
            .open(&OpenDocumentArgs {
                full_document_name: "/memory/part.opd".into(),
                ..Default::default()
            })
            .unwrap()
    });
    assert!(
        owner
            .documents()
            .workspace()
            .documents()
            .contains(&reopened.id)
    );
    assert!(owner.close_allowed().unwrap());
}
fn verify_transaction_close_guard(owner: &mut TestOwner, awake: &Receiver<()>) {
    invoke(owner, awake, |client| {
        client
            .transactions()
            .begin(&TransactionBeginArgs {
                label: "edit".into(),
            })
            .unwrap()
    });
    assert!(!owner.close_allowed().unwrap());
    invoke(owner, awake, |client| {
        client.transactions().abort().unwrap()
    });
    assert!(owner.close_allowed().unwrap());
}
#[test]
fn owner_thread_transport_is_declined_and_retirement_releases_admitted_waiters() {
    let (mut owner, awake) = owner();
    assert!(
        Client::new(owner.caller())
            .documents()
            .list()
            .unwrap_err()
            .0
            .contains("owner")
    );
    let client = owner.caller();
    let worker = thread::spawn(move || Client::new(client).documents().list());
    awake.recv_timeout(Duration::from_secs(2)).unwrap();
    assert!(!owner.close_allowed().unwrap());
    owner.finish().unwrap();
    assert!(worker.join().unwrap().unwrap_err().0.contains("closed"));
    assert!(owner.documents().workspace().documents().is_empty());
    assert!(owner.poll(0).unwrap_err().contains("positive bounded"));
}
#[test]
fn drain_budget_preserves_remaining_requests_for_the_next_frame() {
    let (mut owner, awake) = owner();
    let first = owner.caller();
    let first = thread::spawn(move || Client::new(first).documents().list());
    awake.recv_timeout(Duration::from_secs(2)).unwrap();
    let second = owner.caller();
    let second = thread::spawn(move || Client::new(second).documents().list());
    awake.recv_timeout(Duration::from_secs(2)).unwrap();
    verify_two_frame_drain(&mut owner, first, second);
}
fn verify_two_frame_drain(
    owner: &mut TestOwner,
    first: thread::JoinHandle<Result<ListDocumentsResult, oblikovati_api::Error>>,
    second: thread::JoinHandle<Result<ListDocumentsResult, oblikovati_api::Error>>,
) {
    let activity = owner.poll(1).unwrap();
    assert!(activity.changed && activity.pending && activity.wake_after.is_none());
    assert!(first.join().unwrap().is_ok());
    assert_eq!(
        owner.poll(1).unwrap(),
        HeadWorkStatus {
            changed: true,
            pending: false,
            wake_after: None
        }
    );
    assert!(second.join().unwrap().is_ok());
    assert_eq!(owner.poll(1).unwrap(), HeadWorkStatus::default());
}
#[test]
fn retiring_the_queue_preserves_unsaved_document_ownership_for_recovery() {
    let (mut owner, awake) = owner();
    let id = create(&mut owner, &awake);
    let transport = owner.caller();
    let files = owner.into_documents();
    assert_eq!(files.workspace().documents(), [id]);
    assert!(files.path(id).is_err());
    let worker = thread::spawn(move || Client::new(transport).documents().list());
    assert!(worker.join().unwrap().unwrap_err().0.contains("closed"));
}
#[test]
fn ui_commands_borrow_the_same_canonical_owner_without_worker_admission() {
    let (mut owner, awake) = owner();
    let document = Client::new(owner.document_host())
        .documents()
        .create(&CreateDocumentArgs {
            type_: "part".into(),
            name: "UI part".into(),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(owner.documents().workspace().documents(), [document.id]);
    assert!(awake.try_recv().is_err());
    let activity = owner.poll(32).unwrap();
    assert!(activity.changed && !activity.pending);
    assert_eq!(owner.poll(32).unwrap(), HeadWorkStatus::default());
    assert!(!owner.close_allowed().unwrap());
}

#[path = "owner/part_scene.rs"]
mod part_scene;

#[path = "owner/commands.rs"]
mod commands;

#[path = "owner/gui_commands.rs"]
mod gui_commands;
#[path = "owner/parameter_view.rs"]
mod parameter_view;

#[path = "owner/slot_input.rs"]
mod slot_input;

#[path = "owner/sketch_edit.rs"]
mod sketch_edit;

#[path = "owner/rectangle_input.rs"]
mod rectangle_input;

#[path = "owner/circle_input.rs"]
mod circle_input;

#[path = "owner/arc_input.rs"]
mod arc_input;

#[path = "owner/ellipse_input.rs"]
mod ellipse_input;

#[path = "owner/polygon_input.rs"]
mod polygon_input;

#[path = "owner/arc_slot_input.rs"]
mod arc_slot_input;

#[path = "owner/point_input.rs"]
mod point_input;

#[path = "owner/cube_ui.rs"]
mod cube_ui;
