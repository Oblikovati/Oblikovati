// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::json::EmptyObject;
use oblikovati_automation::SessionHost;
fn member_workspace() -> (Workspace, u64) {
    let mut workspace = WorkspaceSession::new(QueryObserver::default());
    let id = register(
        &mut workspace,
        UnitsOfMeasure::default(),
        &mut QueryGuids(0),
    );
    workspace
        .commands(id)
        .unwrap()
        .run("Initialize part members", |snapshot, _| {
            snapshot.model_states = Some(Default::default());
            Ok::<_, String>(())
        })
        .unwrap();
    (workspace, id)
}
#[test]
fn typed_model_state_names_with_nul_are_rejected_atomically() {
    let (mut workspace, _) = member_workspace();
    let client = Client::new(SessionHost::new(&mut workspace));
    let before = client.model_states().list(&EmptyObject::default()).unwrap();
    let error = client
        .model_states()
        .create(&CreateModelStateArgs {
            name: "bad\0name".into(),
            ..Default::default()
        })
        .unwrap_err();
    assert!(error.to_string().contains("bad\\0name"));
    assert_eq!(
        client.model_states().list(&EmptyObject::default()).unwrap(),
        before
    );
}
#[test]
fn typed_model_state_lifecycle_keeps_copy_inactive_and_rename_identity_stable() {
    let (mut workspace, id) = member_workspace();
    let created;
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        let primary = client
            .model_states()
            .list(&EmptyObject::default())
            .unwrap()
            .model_states
            .unwrap();
        assert_eq!(primary.len(), 1);
        assert!(primary[0].primary && primary[0].active);
        created = client
            .model_states()
            .create(&CreateModelStateArgs {
                name: "Variant".into(),
                ..Default::default()
            })
            .unwrap()
            .model_state
            .id;
        let activated = client
            .model_states()
            .activate(&RepRef { id: created })
            .unwrap()
            .model_state;
        assert!(activated.active);
        let renamed = client
            .model_states()
            .rename(&ModelStateNameArgs {
                id: created,
                name: "Large".into(),
            })
            .unwrap()
            .model_state;
        assert_eq!(renamed.id, created);
        assert_eq!(renamed.name, "Large");
        let copied = client
            .model_states()
            .copy(&ModelStateNameArgs {
                id: created,
                name: "Copy".into(),
            })
            .unwrap()
            .model_state;
        assert_ne!(copied.id, created);
        assert!(!copied.active && !copied.primary);
        assert!(
            client
                .model_states()
                .rename(&ModelStateNameArgs {
                    id: 1,
                    name: "Renamed Primary".into()
                })
                .is_err()
        );
        assert!(client.model_states().delete(&RepRef { id: 1 }).is_err());
    }
    workspace.commands(id).unwrap().undo().unwrap();
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .model_states
            .as_ref()
            .unwrap()
            .list()
            .len(),
        2
    );
    workspace.commands(id).unwrap().redo().unwrap();
    let client = Client::new(SessionHost::new(&mut workspace));
    let remaining = client
        .model_states()
        .delete(&RepRef { id: created })
        .unwrap()
        .model_states
        .unwrap();
    assert_eq!(remaining.len(), 2);
    assert!(remaining[0].active);
}
