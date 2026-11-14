// SPDX-License-Identifier: GPL-2.0-only
//! Selection queries follow active document changes.
use super::*;

#[test]
fn one_api_transport_reads_the_current_document_selection_after_each_switch() {
    let (mut owner, awake) = owner();
    let first = create(&mut owner, &awake);
    let sketch = owner
        .create_sketch_for_edit(first, &sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    owner.finish_sketch_edit().unwrap();
    owner
        .select_browser_sketch_from_gui(first, sketch, false)
        .unwrap();
    let client = Client::new(owner.document_host());
    assert_eq!(client.model().selection().unwrap().count, 1);
    client
        .documents()
        .activate(&ActivateDocumentArgs { id: first })
        .unwrap();
    assert_eq!(client.model().selection().unwrap().count, 1);
    let second = client
        .documents()
        .create(&CreateDocumentArgs {
            type_: "part".into(),
            name: "Second selection document".into(),
            ..Default::default()
        })
        .unwrap()
        .id;
    assert_eq!(client.model().selection().unwrap().count, 0);
    client
        .documents()
        .activate(&ActivateDocumentArgs { id: first })
        .unwrap();
    assert_eq!(client.model().selection().unwrap().count, 0);
    client
        .documents()
        .activate(&ActivateDocumentArgs { id: second })
        .unwrap();
    assert_eq!(client.model().selection().unwrap().count, 0);
}
