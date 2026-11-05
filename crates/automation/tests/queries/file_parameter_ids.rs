// SPDX-License-Identifier: GPL-2.0-only

use super::*;

#[test]
fn parameter_reservations_survive_undo_and_normal_or_copy_save() {
    for copy in [false, true] {
        let (mut files, id, mut guids) = created_files(false);
        files.save(id, "test", &mut guids).unwrap();
        let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
        client
            .parameters()
            .add(&ParameterSetArgs {
                name: "temporary".into(),
                expression: "3 cm".into(),
                ..Default::default()
            })
            .unwrap();
        client.transactions().undo().unwrap();
        assert!(files.workspace().needs_save(id).unwrap());
        let path = if copy {
            "/fixtures/copy.opd"
        } else {
            "/fixtures/part.opd"
        };
        let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
        if copy {
            client
                .documents()
                .save_copy_as(&SaveCopyAsArgs {
                    document: id,
                    target_file_name: path.into(),
                    ..Default::default()
                })
                .unwrap();
        }
        client
            .documents()
            .close(&CloseDocumentArgs { id, force: copy })
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
                full_document_name: path.into(),
                ..Default::default()
            })
            .unwrap();
        client
            .parameters()
            .add(&ParameterSetArgs {
                name: "next".into(),
                expression: "4 cm".into(),
                ..Default::default()
            })
            .unwrap();
        let parameters = &files
            .workspace()
            .session(reopened.id)
            .unwrap()
            .snapshot()
            .parameters;
        assert_eq!(parameters.find("width"), Some(1));
        assert_eq!(parameters.find("next"), Some(3));
    }
}
