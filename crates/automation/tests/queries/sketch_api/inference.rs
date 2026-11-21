// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_automation::{QUERY_METHODS, SessionHost};
use oblikovati_model::sketch_settings::{SketchSettings, default_sketch_settings};
fn input(priority: &str) -> InferenceOptionsView {
    InferenceOptionsView {
        infer_enabled: false,
        constrain_enabled: true,
        priority: priority.into(),
    }
}
fn get(files: &DiscoveryWorkspace) -> InferenceOptionsView {
    Client::new(FileQueryHost::new(files))
        .sketch()
        .inference_options()
        .unwrap()
}
fn set(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    input: &InferenceOptionsView,
) -> Result<InferenceOptionsView, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .set_inference_options(input)
}
fn owner_settings(files: &DiscoveryWorkspace, document: u64) -> Option<SketchSettings> {
    files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketch_settings
        .clone()
}
#[test]
fn canonical_query_reads_defaults_without_allocating_or_recording_history() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    files.save(document, "test", &mut source).unwrap();
    let before = owner_revision(&files, document);
    let view = get(&files);
    assert_eq!(
        view,
        InferenceOptionsView {
            infer_enabled: true,
            constrain_enabled: true,
            priority: "horizontalVertical".into()
        }
    );
    assert_eq!(owner_revision(&files, document), before);
    assert_eq!(owner_settings(&files, document), None);
    assert!(QUERY_METHODS.contains(&METHOD_SKETCH_GET_INFERENCE_OPTIONS));
    assert!(SKETCH_COMMAND_METHODS.contains(&METHOD_SKETCH_SET_INFERENCE_OPTIONS));
}
#[test]
fn canonical_priority_options_round_trip_for_every_document_kind() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY, DOCUMENT_DRAWING] {
        let (mut files, document, mut source) = configured_files(kind, Default::default());
        for priority in ["parallelPerpendicular", "none", "horizontalVertical"] {
            assert_eq!(
                set(&mut files, &mut source, &input(priority)).unwrap(),
                input(priority)
            );
            assert_eq!(get(&files), input(priority));
        }
        assert_eq!(files.workspace().active_document(), Some(document));
        assert_eq!(owner_revision(&files, document).1.len(), 3);
    }
}
#[test]
fn omitted_boolean_fields_replace_values_and_empty_priority_retains_current() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    set(&mut files, &mut source, &input("none")).unwrap();
    let host = FileHost::new(&mut files, &mut source, "test");
    let reply: InferenceOptionsView = serde_json::from_slice(
        &host
            .call(METHOD_SKETCH_SET_INFERENCE_OPTIONS, b"{}")
            .unwrap(),
    )
    .unwrap();
    assert!(!reply.infer_enabled && !reply.constrain_enabled);
    assert_eq!(reply.priority, "none");
}
#[test]
fn malformed_input_keeps_saved_document_unchanged() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    files.save(document, "test", &mut source).unwrap();
    let before = owner_revision(&files, document);
    for request in [
        b"[]".as_slice(),
        b"{",
        b"{\"priority\":4}",
        b"{\"inferEnabled\":\"bad\"}",
    ] {
        assert!(
            FileHost::new(&mut files, &mut source, "test")
                .call(METHOD_SKETCH_SET_INFERENCE_OPTIONS, request)
                .is_err()
        );
    }
    assert_eq!(owner_revision(&files, document), before);
    assert_eq!(owner_settings(&files, document), None);
}
#[test]
fn inference_edits_retain_grid_display_and_relax_metadata() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let mut custom = default_sketch_settings();
    custom.x_snap_spacing = 0.75;
    custom.enable_relax_mode = true;
    files
        .commands(document)
        .unwrap()
        .run("Custom defaults", |snapshot, _| {
            snapshot.sketch_settings = Some(custom.clone());
            Ok::<(), String>(())
        })
        .unwrap();
    set(&mut files, &mut source, &input("none")).unwrap();
    custom.infer_constraints = false;
    custom.constraint_priority = oblikovati_api::types::PRIORITY_NONE;
    assert_eq!(owner_settings(&files, document), Some(custom));
}
#[test]
fn empty_workspace_fallback_is_shared_by_new_clients_and_has_no_document() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    assert!(get(&files).infer_enabled);
    set(&mut files, &mut source, &input("none")).unwrap();
    assert_eq!(get(&files), input("none"));
    assert!(files.workspace().documents().is_empty());
    assert_eq!(source.0, 0);
    assert!(set(&mut files, &mut source, &input("bad")).is_err());
    assert_eq!(get(&files), input("none"));
}
#[test]
fn fallback_is_not_inherited_by_new_document_and_returns_after_close() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    set(&mut files, &mut source, &input("none")).unwrap();
    let document = create_kind(&mut files, DOCUMENT_PART, &mut source);
    assert_eq!(get(&files).priority, "horizontalVertical");
    set(&mut files, &mut source, &input("parallelPerpendicular")).unwrap();
    files.close_discarding_changes(document).unwrap();
    assert_eq!(get(&files), input("none"));
}
#[test]
fn session_only_host_shares_the_same_fallback_and_query_contract() {
    let mut workspace = Workspace::new(QueryObserver::default());
    let input = input("parallelPerpendicular");
    assert_eq!(
        Client::new(SessionHost::new(&mut workspace))
            .sketch()
            .set_inference_options(&input)
            .unwrap(),
        input
    );
    assert_eq!(
        Client::new(SessionQueryHost::new(&workspace))
            .sketch()
            .inference_options()
            .unwrap(),
        input
    );
}
#[test]
fn read_only_host_refuses_the_setter_without_changing_preferences() {
    let (files, document, _) = configured_files(DOCUMENT_PART, Default::default());
    assert!(
        Client::new(FileQueryHost::new(&files))
            .sketch()
            .set_inference_options(&input("none"))
            .is_err()
    );
    assert_eq!(owner_settings(&files, document), None);
}

#[path = "inference/history.rs"]
mod history;
#[test]
fn invalid_priority_keeps_saved_settings_and_generation_unchanged() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    files.save(document, "test", &mut source).unwrap();
    let before = owner_revision(&files, document);
    let error = set(&mut files, &mut source, &input("bad")).unwrap_err();
    assert!(error.to_string().contains("bad"));
    assert_eq!(owner_revision(&files, document), before);
    assert_eq!(owner_settings(&files, document), None);
}
#[test]
fn session_host_queries_its_own_changed_fallback() {
    let mut workspace = Workspace::new(QueryObserver::default());
    let client = Client::new(SessionHost::new(&mut workspace));
    client
        .sketch()
        .set_inference_options(&input("none"))
        .unwrap();
    assert_eq!(client.sketch().inference_options().unwrap(), input("none"));
}
#[test]
fn session_host_routes_the_setter_to_the_active_document() {
    let mut workspace = Workspace::new(QueryObserver::default());
    let document = register(
        &mut workspace,
        UnitsOfMeasure::default(),
        &mut QueryGuids(0),
    );
    Client::new(SessionHost::new(&mut workspace))
        .sketch()
        .set_inference_options(&input("none"))
        .unwrap();
    let session = workspace.session(document).unwrap();
    let settings = session.snapshot().effective_sketch_settings();
    assert_eq!(
        settings.constraint_priority,
        oblikovati_api::types::PRIORITY_NONE
    );
    assert_eq!(session.undo_labels(), ["Set sketch inference options"]);
}
#[test]
fn query_ignores_unused_arguments_as_the_go_getter_does() {
    let (files, _, _) = configured_files(DOCUMENT_PART, Default::default());
    let expected = get(&files);
    let bytes = FileQueryHost::new(&files)
        .call(METHOD_SKETCH_GET_INFERENCE_OPTIONS, b"ignored")
        .unwrap();
    assert_eq!(
        serde_json::from_slice::<InferenceOptionsView>(&bytes).unwrap(),
        expected
    );
}
