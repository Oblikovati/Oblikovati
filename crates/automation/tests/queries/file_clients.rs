// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_automation::SessionHost;

#[test]
fn typed_file_host_connections_preserve_document_history_and_focus() {
    let (mut files, id, mut guids) = created_files(false);
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    assert_eq!(
        client.client_applications().list().unwrap().clients,
        Some(vec![])
    );
    let first = client
        .client_applications()
        .register(&RegisterClientApplicationArgs { name: "cam".into() })
        .unwrap()
        .id;
    let second = client
        .client_applications()
        .register(&RegisterClientApplicationArgs { name: "cam".into() })
        .unwrap()
        .id;
    assert_eq!((first, second), (1, 2));
    assert!(
        client
            .client_applications()
            .unregister(&UnregisterClientApplicationArgs { id: first })
            .unwrap()
            .ok
    );
    assert_eq!(
        client
            .client_applications()
            .list()
            .unwrap()
            .clients
            .unwrap(),
        [ClientApplicationInfo {
            id: second,
            name: "cam".into()
        }]
    );
    assert!(
        client
            .client_applications()
            .register(&RegisterClientApplicationArgs::default())
            .is_err()
    );
    assert!(
        client
            .client_applications()
            .unregister(&UnregisterClientApplicationArgs { id: first })
            .is_err()
    );
    assert!(
        FileHost::new(&mut files, &mut guids, "test")
            .call(METHOD_CLIENT_APPS_REGISTER, b"{bad")
            .is_err()
    );
    assert_eq!(files.workspace().active_document(), Some(id));
    assert!(
        files
            .workspace()
            .session(id)
            .unwrap()
            .undo_labels()
            .is_empty()
    );
    assert_eq!(files.workspace().connected_clients().entries().len(), 1);
    let query = Client::new(SessionQueryHost::new(files.workspace()));
    assert_eq!(
        query.client_applications().list().unwrap().clients.unwrap()[0].id,
        second
    );
    assert!(
        query
            .client_applications()
            .register(&RegisterClientApplicationArgs {
                name: "forbidden".into()
            })
            .is_err()
    );
}

#[test]
fn bare_session_host_accepts_client_registration_without_an_open_document() {
    let mut workspace = Workspace::new(QueryObserver::default());
    let client = Client::new(SessionHost::new(&mut workspace));
    let registered = client
        .client_applications()
        .register(&RegisterClientApplicationArgs {
            name: "driver".into(),
        })
        .unwrap();
    assert_eq!(registered.id, 1);
    assert!(
        client
            .client_applications()
            .unregister(&UnregisterClientApplicationArgs { id: registered.id })
            .unwrap()
            .ok
    );
    assert_eq!(
        client
            .client_applications()
            .register(&RegisterClientApplicationArgs {
                name: "driver".into()
            })
            .unwrap()
            .id,
        2
    );
    assert!(workspace.documents().is_empty());
}
