// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn visibility_fixture(
    kind: DocumentType,
) -> (DiscoveryWorkspace, u64, DocumentSketchId, QueryGuids) {
    let (mut files, document, source) = configured_files(kind, Default::default());
    let sketch = insert_circle(&mut files, document);
    (files, document, sketch, source)
}
fn visibility_call(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    value: &str,
) -> Result<SketchInfo, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .set_property(&SetSketchPropertyArgs {
            sketch_index: 0,
            property: "visible".into(),
            value: value.into(),
        })
}
fn visibility_info(files: &DiscoveryWorkspace) -> SketchInfo {
    Client::new(FileQueryHost::new(files))
        .sketch()
        .get(&SketchArgs::default())
        .unwrap()
}

#[test]
fn visibility_property_updates_both_owners_and_supports_history() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        let (mut files, document, sketch, mut source) = visibility_fixture(kind);
        let before = files
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .clone();
        let initial = visibility_info(&files);
        assert!(initial.visible);
        let hidden = visibility_call(&mut files, &mut source, "false").unwrap();
        assert!(!hidden.visible);
        assert_eq!((hidden.dof, hidden.healthy), (initial.dof, initial.healthy));
        let after = files.workspace().session(document).unwrap().snapshot();
        assert!(
            before
                .sketches
                .get(sketch)
                .unwrap()
                .geometry()
                .same_recipe(after.sketches.get(sketch).unwrap().geometry())
        );
        assert!(!visibility_info(&files).visible);
        files.commands(document).unwrap().undo().unwrap();
        assert!(visibility_info(&files).visible);
        files.commands(document).unwrap().redo().unwrap();
        assert!(!visibility_info(&files).visible);
    }
}

#[test]
fn visibility_accepts_go_boolean_spellings_and_unchanged_values_do_not_create_history() {
    let (mut files, document, _, mut source) = visibility_fixture(DOCUMENT_PART);
    for (expected, values) in [
        (true, ["1", "t", "T", "TRUE", "true", "True"]),
        (false, ["0", "f", "F", "FALSE", "false", "False"]),
    ] {
        files
            .commands(document)
            .unwrap()
            .set_sketch_visibility(DocumentSketchId(1), expected)
            .unwrap();
        for value in values {
            let revision = owner_revision(&files, document);
            assert_eq!(
                visibility_call(&mut files, &mut source, value)
                    .unwrap()
                    .visible,
                expected
            );
            assert_eq!(owner_revision(&files, document), revision);
        }
    }
}

#[test]
fn invalid_property_values_and_indices_leave_every_owned_state_unchanged() {
    let (mut files, document, _, mut source) = visibility_fixture(DOCUMENT_PART);
    let before = files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .clone();
    let revision = owner_revision(&files, document);
    for (property, value, index) in [
        ("name", "false", 0),
        ("visible", " true ", 0),
        ("visible", "yes", 0),
        ("visible", "TrUe", 0),
        ("visible", "false", -1),
        ("visible", "false", i64::MAX),
    ] {
        let args = SetSketchPropertyArgs {
            sketch_index: index,
            property: property.into(),
            value: value.into(),
        };
        let result = Client::new(FileHost::new(&mut files, &mut source, "test"))
            .sketch()
            .set_property(&args);
        assert!(result.is_err(), "{args:?}: {result:?}");
        assert_eq!(owner_revision(&files, document), revision);
        assert!(
            before.sketches.same_evaluated_state(
                &files
                    .workspace()
                    .session(document)
                    .unwrap()
                    .snapshot()
                    .sketches
            )
        );
    }
}

#[test]
fn visibility_replies_keep_the_active_edit_guard() {
    let (mut files, document, sketch, mut source) = visibility_fixture(DOCUMENT_PART);
    let edit = inspection::inspection_edit(&mut files, document, sketch);
    let client =
        Client::new(FileHost::new(&mut files, &mut source, "test").with_sketch_edit(Some(edit)));
    let response = client
        .sketch()
        .set_property(&SetSketchPropertyArgs {
            sketch_index: 0,
            property: "visible".into(),
            value: "false".into(),
        })
        .unwrap();
    assert!(response.editing);
    assert!(!response.visible);
    assert!(client.sketch().get(&SketchArgs::default()).unwrap().editing);
}

#[test]
fn visibility_does_not_require_convergence_or_change_current_geometry() {
    let (mut files, document, sketch, mut source) = files_with_circle(SolveOptions {
        max_iterations: 0,
        ..Default::default()
    });
    files
        .commands(document)
        .unwrap()
        .run("Unsatisfied radius", |snapshot, _| {
            let mut geometry = snapshot.sketches.get(sketch)?.geometry().clone();
            geometry.set_radius(SketchEntityId(2), 2.0)?;
            snapshot.sketches.replace_geometry(sketch, geometry)
        })
        .unwrap();
    let before = files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .clone();
    assert!(!visibility_info(&files).healthy);
    let response = visibility_call(&mut files, &mut source, "false").unwrap();
    assert!(!response.visible && !response.healthy);
    let after = files.workspace().session(document).unwrap().snapshot();
    assert!(
        before
            .sketches
            .get(sketch)
            .unwrap()
            .geometry()
            .same_recipe(after.sketches.get(sketch).unwrap().geometry())
    );
}

#[test]
fn an_unavailable_inspection_reply_refuses_visibility_without_adoption() {
    let (mut files, document, sketch, mut source) = visibility_fixture(DOCUMENT_PART);
    files
        .commands(document)
        .unwrap()
        .run("Unavailable width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "futureWidth")
        })
        .unwrap();
    let before = files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .clone();
    let revision = owner_revision(&files, document);
    let cause = visibility_call(&mut files, &mut source, "false").unwrap_err();
    assert!(cause.to_string().contains("futureWidth"), "{cause}");
    let after = files.workspace().session(document).unwrap().snapshot();
    assert!(before.sketches.same_evaluated_state(&after.sketches));
    assert!(after.sketches.get(sketch).unwrap().visible());
    assert_eq!(owner_revision(&files, document), revision);
}
