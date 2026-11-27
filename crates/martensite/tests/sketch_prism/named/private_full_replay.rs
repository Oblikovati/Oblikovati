// SPDX-License-Identifier: GPL-2.0-only
//! The first feature is replayed from the private model-state candidate (#4122).
use super::*;
use oblikovati_martensite::{capture_full_named_candidate, complete_named_checkpoints};

#[test]
fn discarded_full_replay_keeps_the_old_named_body_current() {
    let mut history = owner(selected_prism_document());
    let old = named(&history);
    let generation = history.generation();
    let (prepared, ()) = history
        .prepare_private_edit("depth", |snapshot, _, _| {
            snapshot.parameters.set_expression(3, "3 cm")
        })
        .unwrap();
    let captured =
        capture_full_named_candidate(7, prepared.source(), prepared.snapshot(), document(), None)
            .unwrap();
    drop(captured);
    drop(prepared);
    assert_eq!(history.generation(), generation);
    assert_faces(&old, &history, 2.);
}

#[test]
fn full_replay_captures_the_first_feature_with_the_new_depth() {
    let mut history = owner(selected_prism_document());
    let (prepared, ()) = history
        .prepare_private_edit("depth", |snapshot, _, _| {
            snapshot.parameters.set_expression(3, "3 cm")
        })
        .unwrap();
    let captured =
        capture_full_named_candidate(7, prepared.source(), prepared.snapshot(), document(), None)
            .unwrap();
    let transition = prepared.publish();
    let completed =
        complete_named_checkpoints(captured, 7, &transition, &history, document()).unwrap();
    let checkpoints = completed.into_current(7, &history).unwrap();
    let body = checkpoints.result().body(FeatureId(1)).unwrap();
    assert_analytic_scalar(body.volume(&history, document()).unwrap(), 36.);
    assert_eq!(history.undo_labels(), ["depth"]);
}

#[test]
fn prepared_real_body_and_name_indices_publish_with_the_candidate() {
    let mut history = owner(selected_prism_document());
    let (candidate, ()) = history
        .prepare_private_edit("depth", |snapshot, _, _| {
            snapshot.parameters.set_expression(3, "3 cm")
        })
        .unwrap();
    let body = oblikovati_martensite::prepare_full_named_publication(&candidate, document(), None)
        .unwrap();
    let (checkpoints, _) = oblikovati_model::history::publish_prepared(candidate, body).unwrap();
    let body = checkpoints.result().body(FeatureId(1)).unwrap();
    assert_analytic_scalar(body.volume(&history, document()).unwrap(), 36.);
    assert!(
        !body
            .reference_keys(&history, document())
            .unwrap()
            .is_empty()
    );
}

#[test]
fn abandoned_real_body_cannot_publish_a_later_candidate_or_dependent() {
    let mut history = owner(selected_prism_document());
    let mut dependent = owner(selected_prism_document());
    let old = named(&history);
    let generation = history.generation();
    let (first, ()) = history
        .prepare_private_edit("first", |snapshot, _, _| {
            snapshot.parameters.set_expression(3, "3 cm")
        })
        .unwrap();
    let body =
        oblikovati_martensite::prepare_full_named_publication(&first, document(), None).unwrap();
    drop(first);
    let (second, ()) = history
        .prepare_private_edit("second", |snapshot, _, _| {
            snapshot.parameters.set_expression(3, "4 cm")
        })
        .unwrap();
    let refresh = dependent
        .prepare_refresh(|snapshot| snapshot.parameters.set_expression(3, "5 cm"))
        .unwrap();
    let refresh = oblikovati_model::history::PreparedDependentRefresh::new(&second, refresh);
    assert!(oblikovati_model::history::publish_prepared(second, (refresh, body)).is_err());
    assert_eq!(history.generation(), generation);
    assert_faces(&old, &history, 2.);
    assert_eq!(
        dependent
            .snapshot()
            .parameters
            .get(3)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        2.0
    );
}

#[test]
fn source_and_dependent_real_solids_publish_under_their_own_scopes() {
    use oblikovati_model::history::{
        PreparedDependentPublication, PreparedDependentRefresh, publish_prepared,
    };
    let mut history = owner(selected_prism_document());
    let mut dependent = owner(selected_prism_document());
    let (candidate, ()) = history
        .prepare_private_edit("source", |snapshot, _, _| {
            snapshot.parameters.set_expression(3, "3 cm")
        })
        .unwrap();
    let source_body =
        oblikovati_martensite::prepare_full_named_publication(&candidate, document(), None)
            .unwrap();
    let refresh = dependent
        .prepare_refresh(|snapshot| snapshot.parameters.set_expression(3, "4 cm"))
        .unwrap();
    let refresh = PreparedDependentRefresh::new(&candidate, refresh);
    let dependent_body = oblikovati_martensite::prepare_dependent_named_publication(
        &candidate,
        &refresh,
        document(),
        None,
    )
    .unwrap();
    let ((source_bodies, (changed, _, dependent_bodies)), _) = publish_prepared(
        candidate,
        (
            source_body,
            PreparedDependentPublication::new(refresh, dependent_body),
        ),
    )
    .unwrap();
    assert!(changed);
    let source_body = source_bodies.result().body(FeatureId(1)).unwrap();
    let dependent_body = dependent_bodies.result().body(FeatureId(1)).unwrap();
    assert_analytic_scalar(source_body.volume(&history, document()).unwrap(), 36.);
    assert_analytic_scalar(dependent_body.volume(&dependent, document()).unwrap(), 48.);
    assert!(
        !dependent_body
            .reference_keys(&dependent, document())
            .unwrap()
            .is_empty()
    );
}
