// SPDX-License-Identifier: GPL-2.0-only

use super::*;

#[test]
fn removing_the_last_property_preserves_its_set_through_file_save_and_reopen() {
    let (mut files, id, mut guids) = created_files(false);
    files
        .commands(id)
        .unwrap()
        .run("Custom set", |snapshot, _| {
            snapshot
                .properties
                .put("Reserved", "temporary", VariantValue::Integer(1))?;
            assert!(snapshot.properties.remove("Reserved", "temporary"));
            snapshot.properties.ensure_set("Next")
        })
        .unwrap();
    let expected = files
        .workspace()
        .session(id)
        .unwrap()
        .snapshot()
        .properties
        .clone();
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    client
        .documents()
        .save(&SaveDocumentArgs { document: id })
        .unwrap();
    client
        .documents()
        .close(&CloseDocumentArgs { id, force: false })
        .unwrap();
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 1,
    };
    let client = Client::new(
        FileHost::new(&mut files, &mut guids, "test").with_observer_factory(&mut factory),
    );
    let reopened = client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: "/fixtures/part.opd".into(),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(
        files
            .workspace()
            .session(reopened.id)
            .unwrap()
            .snapshot()
            .properties,
        expected
    );
}
