// SPDX-License-Identifier: GPL-2.0-only
//! Assembly producer lookup stays on the Express path.
use super::*;

#[test]
fn part_creation_does_not_hash_the_assembly_producer() {
    let (mut owner, awake) = owner();
    assert!(!owner.part_scenes.has_assembly_producer_digest());
    let document = create(&mut owner, &awake);
    assert!(!owner.part_scenes.has_assembly_producer_digest());
    assert_eq!(
        owner.documents().workspace().active_document(),
        Some(document)
    );
}

#[test]
fn express_open_requests_the_injected_assembly_producer() {
    let (mut owner, _) = owner();
    owner
        .part_scenes
        .set_test_producer_digest(Err("test producer unavailable".into()));
    let error = Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: document_fixture_path!("/memory/missing.oad").into(),
            express: true,
            ..Default::default()
        })
        .unwrap_err();
    assert!(
        error.to_string().contains("test producer unavailable"),
        "{error}"
    );
}
