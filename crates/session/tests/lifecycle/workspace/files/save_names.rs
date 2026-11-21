// SPDX-License-Identifier: GPL-2.0-only
use super::super::*;

fn saved_name(host: &Host, name: &str) -> String {
    let bytes = host.files().contents.borrow()[&path(name)].clone();
    DocumentPackage::decode(&bytes)
        .unwrap()
        .manifest()
        .display_name
        .clone()
}
#[test]
fn ordinary_save_writes_the_current_derived_title() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    host.save(id, "beta", &mut source).unwrap();
    assert_eq!(saved_name(&host, "a.obk"), "a");
    assert_eq!(host.workspace().display_name(id).unwrap(), "a");
}
#[test]
fn save_as_writes_the_destination_title_and_retains_derived_ownership() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    host.save_as(id, &path("new.obk"), "beta", &mut source)
        .unwrap();
    assert_eq!(saved_name(&host, "new.obk"), "new");
    host.save_as(id, &path("later.obk"), "beta", &mut source)
        .unwrap();
    assert_eq!(saved_name(&host, "later.obk"), "later");
}
#[test]
fn authored_titles_survive_save_as_and_source_titles_survive_copy() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = create_authored(&mut host, &mut source);
    host.save_as(id, &path("later.obk"), "beta", &mut source)
        .unwrap();
    assert_eq!(saved_name(&host, "later.obk"), "Authored Ω");
    host.save_copy(id, &path("copy.obk"), None, "beta", &mut source)
        .unwrap();
    assert_eq!(saved_name(&host, "copy.obk"), "Authored Ω");
}
#[test]
fn derived_source_title_is_retained_by_save_copy_without_rebinding() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    host.save_copy(id, &path("copy.obk"), None, "beta", &mut source)
        .unwrap();
    assert_eq!(saved_name(&host, "copy.obk"), "a");
    assert_eq!(host.path(id).unwrap(), path("a.obk"));
    assert_eq!(host.files().contents.borrow()[&path("a.obk")], fixture(1));
}

fn create_authored(host: &mut Host, source: &mut SequenceGuids) -> u64 {
    let package = DocumentPackage::empty(oblikovati_api::types::DOCUMENT_PART, "Authored Ω");
    host.create(
        &path("new.obk"),
        package,
        RecordingObserver::default(),
        "beta",
        source,
    )
    .unwrap()
    .session_id
}
