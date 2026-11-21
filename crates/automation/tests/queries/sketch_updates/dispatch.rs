// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_automation::dispatch::owner_dispatch;
use std::{sync::mpsc, thread, time::Duration};
#[path = "../../support/dispatch_wakeup.rs"]
mod dispatch_wakeup;
use dispatch_wakeup::WakeSignal;
#[test]
fn a_worker_typed_client_edit_recomputes_only_when_the_borrowed_session_owner_drains() {
    let expected = direct_edit_radius();
    let (mut workspace, document, sketch) = circle_workspace();
    let (signal, awake) = mpsc::channel();
    let (mut owner, caller) = owner_dispatch(4, Duration::from_secs(2), WakeSignal(signal));
    let worker = start_width_edit(caller);
    awake.recv_timeout(Duration::from_secs(2)).unwrap();
    assert_eq!(circle_radius(&workspace, document, sketch), 1.0);
    let generation = workspace.session(document).unwrap().generation();
    drain_one_edit(&mut owner, &mut workspace);
    assert_eq!(worker.join().unwrap().unwrap().name, "width");
    assert_radius_bits(&workspace, document, sketch, expected);
    assert!(workspace.session(document).unwrap().generation() > generation);
    workspace.commands(document).unwrap().undo().unwrap();
    assert_eq!(circle_radius(&workspace, document, sketch), 1.0);
}
fn drain_one_edit(
    owner: &mut oblikovati_automation::dispatch::OwnerDispatch<WakeSignal>,
    workspace: &mut Workspace,
) {
    let host = SessionHost::new(workspace);
    assert_eq!(owner.drain(1, &host).unwrap(), 1);
}

fn direct_edit_radius() -> f64 {
    let (mut workspace, document, sketch) = circle_workspace();
    api_width(&mut workspace, "7 cm").unwrap();
    circle_radius(&workspace, document, sketch)
}
#[test]
fn a_timed_out_typed_edit_still_recomputes_once_when_the_owner_later_drains() {
    let expected = direct_edit_radius();
    let (mut workspace, document, sketch) = circle_workspace();
    let (signal, awake) = mpsc::channel();
    let (mut owner, caller) = owner_dispatch(1, Duration::from_millis(30), WakeSignal(signal));
    let worker = start_width_edit(caller);
    awake.recv_timeout(Duration::from_secs(2)).unwrap();
    assert_accepted_timeout(worker);
    assert_eq!(circle_radius(&workspace, document, sketch), 1.0);
    let before = workspace.session(document).unwrap().undo_labels().len();
    drain_one_edit(&mut owner, &mut workspace);
    assert_radius_bits(&workspace, document, sketch, expected);
    assert_one_late_edit(&mut workspace, document, before, &mut owner);
}

fn start_width_edit(
    caller: oblikovati_automation::dispatch::QueuedCaller<WakeSignal>,
) -> thread::JoinHandle<Result<ParameterInfo, oblikovati_api::Error>> {
    thread::spawn(move || {
        Client::new(caller)
            .parameters()
            .set(&numeric("width", "7 cm"))
    })
}
fn assert_radius_bits(
    workspace: &Workspace,
    document: u64,
    sketch: DocumentSketchId,
    expected: f64,
) {
    assert_eq!(
        circle_radius(workspace, document, sketch).to_bits(),
        expected.to_bits()
    );
}
fn assert_accepted_timeout(
    worker: thread::JoinHandle<Result<ParameterInfo, oblikovati_api::Error>>,
) {
    assert!(
        worker
            .join()
            .unwrap()
            .unwrap_err()
            .to_string()
            .contains("accepted request may still execute")
    );
}
fn assert_one_late_edit(
    workspace: &mut Workspace,
    document: u64,
    before: usize,
    owner: &mut oblikovati_automation::dispatch::OwnerDispatch<WakeSignal>,
) {
    assert_eq!(
        workspace.session(document).unwrap().undo_labels().len(),
        before + 1
    );
    assert_eq!(owner.drain(0, &SessionHost::new(workspace)).unwrap(), 0);
}
