// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn typed_scope_round_trips_and_rejects_invalid_selection_atomically() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    let index = SketchInferenceScopeArgs { sketch_index: 0 };
    let id = files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .geometry()
        .entity_ids()[0]
        .0;
    let before = owner_revision(&files, document);
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    let initial = client.sketch().inference_scope(&index).unwrap();
    assert_eq!(initial.scope, INFERENCE_SCOPE_CURRENT_COMMAND);
    let selected = SketchInferenceScopeView {
        sketch_index: 0,
        scope: INFERENCE_SCOPE_SELECTED_GEOMETRY,
        entities: Some(vec![id]),
    };
    assert_eq!(
        client.sketch().set_inference_scope(&selected).unwrap(),
        selected
    );
    let invalid = SketchInferenceScopeView {
        entities: Some(vec![u64::MAX]),
        ..selected.clone()
    };
    assert!(client.sketch().set_inference_scope(&invalid).is_err());
    assert_eq!(client.sketch().inference_scope(&index).unwrap(), selected);
    assert!(
        client
            .sketch()
            .inference_scope(&SketchInferenceScopeArgs { sketch_index: 99 })
            .is_err()
    );
    drop(client);
    assert_eq!(
        Client::new(FileQueryHost::new(&files))
            .sketch()
            .inference_scope(&index)
            .unwrap(),
        selected
    );
    assert_eq!(owner_revision(&files, document), before);
}

#[test]
fn session_host_typed_scope_validates_ids_and_retains_the_session_value() {
    use oblikovati_automation::SessionHost;
    let mut workspace = Workspace::new(QueryObserver::default());
    let document = register(
        &mut workspace,
        UnitsOfMeasure::default(),
        &mut QueryGuids(0),
    );
    let sketch = workspace
        .commands(document)
        .unwrap()
        .run("Create circle", create_circle)
        .unwrap();
    let id = workspace
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .geometry()
        .entity_ids()[0]
        .0;
    let generation = workspace.session(document).unwrap().generation();
    let args = SketchInferenceScopeArgs { sketch_index: 0 };
    let selected = SketchInferenceScopeView {
        sketch_index: 0,
        scope: INFERENCE_SCOPE_SELECTED_GEOMETRY,
        entities: Some(vec![id]),
    };
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        assert_eq!(
            client.sketch().inference_scope(&args).unwrap().scope,
            INFERENCE_SCOPE_CURRENT_COMMAND
        );
        assert_eq!(
            client.sketch().set_inference_scope(&selected).unwrap(),
            selected
        );
        let invalid = SketchInferenceScopeView {
            entities: Some(vec![u64::MAX]),
            ..selected.clone()
        };
        assert!(client.sketch().set_inference_scope(&invalid).is_err());
        assert_eq!(client.sketch().inference_scope(&args).unwrap(), selected);
    }
    assert_eq!(
        Client::new(SessionQueryHost::new(&workspace))
            .sketch()
            .inference_scope(&args)
            .unwrap(),
        selected
    );
    assert_eq!(
        workspace.session(document).unwrap().generation(),
        generation
    );
}
