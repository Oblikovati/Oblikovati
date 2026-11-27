// SPDX-License-Identifier: GPL-2.0-only

use super::*;

#[test]
fn discarded_private_candidate_keeps_live_scope_and_reservations() {
    let mut history = history();
    let source = history.evaluation_scope();
    let generation = history.generation();
    let (candidate, id) = history
        .prepare_private_edit("edit", |snapshot, ids, _| {
            let id = ids.allocate().unwrap();
            snapshot.parameters.set_expression(1, "20 mm")?;
            Ok::<_, ParameterError<u64>>(id)
        })
        .unwrap();
    assert_eq!(id, 2);
    assert_eq!(
        candidate
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        2.0
    );
    assert_eq!(candidate.source().generation(), generation);
    drop(candidate);
    assert!(source.is_current(&history));
    assert_eq!(width(&history), 1.0);
    assert_eq!(history.high_water(), 1);
    assert!(history.undo_labels().is_empty());
}

#[test]
fn accepted_private_candidate_is_one_undoable_edit() {
    let mut history = history();
    let (candidate, ()) = history
        .prepare_private_edit("edit", |snapshot, _, _| {
            snapshot.parameters.set_expression(1, "20 mm")
        })
        .unwrap();
    candidate.publish();
    assert_eq!(width(&history), 2.0);
    assert_eq!(history.undo_labels(), ["edit"]);
    history.undo().unwrap();
    assert_eq!(width(&history), 1.0);
    history.redo().unwrap();
    assert_eq!(width(&history), 2.0);
}

#[test]
fn failed_private_preflight_preserves_live_generation_and_ids() {
    let mut history = history();
    let generation = history.generation();
    let result = history.prepare_private_edit("edit", |snapshot, ids, _| {
        ids.allocate().unwrap();
        snapshot.parameters.set_expression(1, "(")
    });
    assert!(result.is_err());
    drop(result);
    assert_eq!(history.generation(), generation);
    assert_eq!(history.high_water(), 1);
    assert_eq!(width(&history), 1.0);
}

#[test]
fn preparation_refuses_an_open_transaction_without_changing_it() {
    let mut history = history();
    let token = history.begin("caller").unwrap();
    let generation = history.generation();
    let result = history.prepare_private_edit("nested", |_, _, _| Ok::<_, String>(()));
    assert!(result.is_err());
    drop(result);
    assert_eq!(history.generation(), generation);
    history.abort(&token).unwrap();
    assert!(history.undo_labels().is_empty());
}

#[test]
fn abandoned_candidate_seal_does_not_match_a_later_candidate() {
    let mut history = history();
    let (first, ()) = history
        .prepare_private_edit("first", |snapshot, _, _| {
            snapshot.parameters.set_expression(1, "20 mm")
        })
        .unwrap();
    let abandoned = first.seal();
    assert!(abandoned.matches(&first));
    drop(first);
    let (second, ()) = history
        .prepare_private_edit("second", |snapshot, _, _| {
            snapshot.parameters.set_expression(1, "30 mm")
        })
        .unwrap();
    assert!(!abandoned.matches(&second));
    assert!(second.seal().matches(&second));
}

fn named_body_key() -> (oblikovati_model::identity::DocumentGuid, Vec<u8>) {
    let document = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let key = oblikovati_model::identity::SemanticReferenceName::new(
        document,
        1,
        4,
        "fixture",
        "body",
        &[1],
    )
    .unwrap()
    .encode()
    .unwrap();
    (document, key)
}

#[test]
fn prepared_names_follow_only_their_accepted_candidate() {
    let mut history = history();
    let (document, key) = named_body_key();
    let (candidate, ()) = history
        .prepare_private_edit("depth", |snapshot, _, _| {
            snapshot.parameters.set_expression(1, "20 mm")
        })
        .unwrap();
    let names = oblikovati_model::identity::PreparedExactReferenceBindings::new(
        &candidate,
        document,
        [(key.clone(), 42)],
    )
    .unwrap();
    let (index, _) = names.publish(candidate).unwrap();
    assert_eq!(*index.bind(&history, document, &key, 4).unwrap(), 42);
    assert_eq!(width(&history), 2.0);
}

#[test]
fn abandoned_names_cannot_publish_a_later_candidate() {
    let mut history = history();
    let generation = history.generation();
    let (document, key) = named_body_key();
    let (first, ()) = history
        .prepare_private_edit("first", |snapshot, _, _| {
            snapshot.parameters.set_expression(1, "20 mm")
        })
        .unwrap();
    let names = oblikovati_model::identity::PreparedExactReferenceBindings::new(
        &first,
        document,
        [(key, 42)],
    )
    .unwrap();
    drop(first);
    let (second, ()) = history
        .prepare_private_edit("second", |snapshot, _, _| {
            snapshot.parameters.set_expression(1, "30 mm")
        })
        .unwrap();
    assert!(names.publish(second).is_err());
    assert_eq!(history.generation(), generation);
    assert_eq!(width(&history), 1.0);
    assert!(history.undo_labels().is_empty());
}

#[test]
fn duplicate_prepared_names_refuse_before_publication() {
    let mut history = history();
    let generation = history.generation();
    let (document, key) = named_body_key();
    let (candidate, ()) = history
        .prepare_private_edit("depth", |snapshot, _, _| {
            snapshot.parameters.set_expression(1, "20 mm")
        })
        .unwrap();
    let names = oblikovati_model::identity::PreparedExactReferenceBindings::new(
        &candidate,
        document,
        [(key.clone(), 1), (key, 2)],
    );
    assert!(names.is_err());
    drop(candidate);
    assert_eq!(history.generation(), generation);
    assert_eq!(width(&history), 1.0);
}

#[test]
fn sealed_names_and_dependent_refresh_publish_together() {
    let mut source = history();
    let mut dependent = history();
    let (document, key) = named_body_key();
    let (candidate, ()) = source
        .prepare_private_edit("depth", |snapshot, _, _| {
            snapshot.parameters.set_expression(1, "20 mm")
        })
        .unwrap();
    let names = oblikovati_model::identity::PreparedExactReferenceBindings::new(
        &candidate,
        document,
        [(key.clone(), 42)],
    )
    .unwrap();
    let target = dependent
        .prepare_refresh(|snapshot| snapshot.parameters.set_expression(1, "30 mm"))
        .unwrap();
    let target = PreparedDependentRefresh::new(&candidate, target);
    let ((index, changed), _) = publish_prepared(candidate, (names, target)).unwrap();
    assert!(changed);
    assert_eq!(width(&source), 2.0);
    assert_eq!(width(&dependent), 3.0);
    assert_eq!(*index.bind(&source, document, &key, 4).unwrap(), 42);
}

#[test]
fn stale_names_refuse_both_the_source_and_dependent_refresh() {
    let mut source = history();
    let mut dependent = history();
    let (document, key) = named_body_key();
    let (first, ()) = source
        .prepare_private_edit("first", |snapshot, _, _| {
            snapshot.parameters.set_expression(1, "20 mm")
        })
        .unwrap();
    let names = oblikovati_model::identity::PreparedExactReferenceBindings::new(
        &first,
        document,
        [(key, 42)],
    )
    .unwrap();
    drop(first);
    let (second, ()) = source
        .prepare_private_edit("second", |snapshot, _, _| {
            snapshot.parameters.set_expression(1, "30 mm")
        })
        .unwrap();
    let target = dependent
        .prepare_refresh(|snapshot| snapshot.parameters.set_expression(1, "40 mm"))
        .unwrap();
    let target = PreparedDependentRefresh::new(&second, target);
    assert!(publish_prepared(second, (target, names)).is_err());
    assert_eq!(width(&source), 1.0);
    assert_eq!(width(&dependent), 1.0);
    assert!(source.undo_labels().is_empty());
}

#[test]
fn nested_prepared_values_publish_with_their_source() {
    let mut source = history();
    let (candidate, ()) = source
        .prepare_private_edit("depth", |snapshot, _, _| {
            snapshot.parameters.set_expression(1, "20 mm")
        })
        .unwrap();
    let values: Vec<PreparedChoice<PreparedValue<u32>, PreparedValue<String>>> = vec![
        PreparedChoice::First(PreparedValue::new(&candidate, 7)),
        PreparedChoice::Second(PreparedValue::new(&candidate, "body".into())),
    ];
    let (values, _) = publish_prepared(candidate, values).unwrap();
    assert!(matches!(&values[0], PreparedChoice::First(7)));
    assert!(matches!(&values[1], PreparedChoice::Second(value) if value == "body"));
    assert_eq!(width(&source), 2.0);
}

#[test]
fn dependent_names_bind_to_the_refreshed_target_history() {
    let mut source = history();
    let mut dependent = history();
    let (document, key) = named_body_key();
    let (candidate, ()) = source
        .prepare_private_edit("source", |snapshot, _, _| {
            snapshot.parameters.set_expression(1, "20 mm")
        })
        .unwrap();
    let refresh = dependent
        .prepare_refresh(|snapshot| snapshot.parameters.set_expression(1, "30 mm"))
        .unwrap();
    let refresh = PreparedDependentRefresh::new(&candidate, refresh);
    let names = oblikovati_model::identity::PreparedExactReferenceBindings::for_dependent(
        &refresh,
        document,
        [(key.clone(), 42)],
    )
    .unwrap();
    let publication = PreparedDependentPublication::new(refresh, names);
    let ((changed, _, index), _) = publish_prepared(candidate, publication).unwrap();
    assert!(changed);
    assert_eq!(*index.bind(&dependent, document, &key, 4).unwrap(), 42);
    assert_eq!(width(&source), 2.0);
    assert_eq!(width(&dependent), 3.0);
}

#[test]
fn dependent_names_from_another_history_refuse_all_publication() {
    let mut source = history();
    let mut first = history();
    let mut second = history();
    let (document, key) = named_body_key();
    let (candidate, ()) = source
        .prepare_private_edit("source", |snapshot, _, _| {
            snapshot.parameters.set_expression(1, "20 mm")
        })
        .unwrap();
    let refresh = first
        .prepare_refresh(|snapshot| snapshot.parameters.set_expression(1, "30 mm"))
        .unwrap();
    let refresh = PreparedDependentRefresh::new(&candidate, refresh);
    let names = oblikovati_model::identity::PreparedExactReferenceBindings::for_dependent(
        &refresh,
        document,
        [(key, 42)],
    )
    .unwrap();
    drop(refresh);
    let other = second
        .prepare_refresh(|snapshot| snapshot.parameters.set_expression(1, "40 mm"))
        .unwrap();
    let other = PreparedDependentRefresh::new(&candidate, other);
    assert!(publish_prepared(candidate, PreparedDependentPublication::new(other, names)).is_err());
    assert_eq!(width(&source), 1.0);
    assert_eq!(width(&first), 1.0);
    assert_eq!(width(&second), 1.0);
}

#[test]
fn prepared_undo_and_redo_keep_the_cursor_until_publication() {
    let mut history = history();
    let token = history.begin("width").unwrap();
    set_width(&mut history, &token, "20 mm");
    history.commit(&token, false).unwrap();
    let generation = history.generation();
    let undo = history
        .prepare_private_navigation(PreparedNavigation::Undo)
        .unwrap();
    assert_eq!(undo.source().generation(), generation);
    assert_eq!(undo.source().undo_labels(), ["width"]);
    drop(undo);
    assert_eq!(history.generation(), generation);
    assert_eq!(width(&history), 2.0);
    let undo = history
        .prepare_private_navigation(PreparedNavigation::Undo)
        .unwrap();
    undo.publish();
    assert_eq!(width(&history), 1.0);
    assert!(history.undo_labels().is_empty());
    assert_eq!(history.redo_labels(), ["width"]);
    let redo = history
        .prepare_private_navigation(PreparedNavigation::Redo)
        .unwrap();
    redo.publish();
    assert_eq!(width(&history), 2.0);
    assert_eq!(history.undo_labels(), ["width"]);
    assert!(history.redo_labels().is_empty());
}
