// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn total(owner: &mut TestOwner, density: f64) -> MassPropertiesResult {
    Client::new(owner.document_host())
        .analysis()
        .mass_properties(&MassPropertiesArgs {
            density_g_cm3: density,
            accuracy: String::new(),
        })
        .unwrap()
}
fn one(owner: &mut TestOwner) -> MassPropertiesResult {
    Client::new(owner.document_host())
        .body()
        .physical_properties(&BodyPhysicalPropertiesArgs::default())
        .unwrap()
}
fn health(owner: &mut TestOwner) -> ModelHealthResult {
    Client::new(owner.document_host())
        .analysis()
        .model_health(&ModelHealthArgs::default())
        .unwrap()
}
fn requires_update(owner: &mut TestOwner) -> bool {
    Client::new(owner.document_host())
        .documents()
        .requires_update()
        .unwrap()
        .requires_update
}
fn update(owner: &mut TestOwner, accept_errors_and_continue: bool) -> UpdateDocumentResult {
    Client::new(owner.document_host())
        .documents()
        .update(&UpdateDocumentArgs {
            accept_errors_and_continue,
        })
        .unwrap()
}
fn rebuild(owner: &mut TestOwner) -> UpdateDocumentResult {
    Client::new(owner.document_host())
        .documents()
        .rebuild(&UpdateDocumentArgs::default())
        .unwrap()
}
fn end_of_part(owner: &mut TestOwner) -> EndOfPartResult {
    Client::new(owner.document_host())
        .documents()
        .get_end_of_part(&oblikovati_api::json::EmptyObject {})
        .unwrap()
}
fn feature(owner: &mut TestOwner, id: u64) -> FeatureDetailResult {
    Client::new(owner.document_host())
        .features()
        .get(&FeatureRefArgs { id })
        .unwrap()
}
#[test]
fn one_body_analysis_is_identical_and_empty_part_is_zero_with_the_requested_density() {
    let (mut owner, awake) = owner();
    let empty = create(&mut owner, &awake);
    let result = total(&mut owner, 2.);
    assert_eq!(
        result,
        MassPropertiesResult {
            density_g_cm3: 2.,
            principal_axes: [[1., 0., 0.], [0., 1., 0.], [0., 0., 1.]],
            ..Default::default()
        }
    );
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: empty,
            force: true,
        })
        .unwrap();
    open_prism(&mut owner, false);
    assert_eq!(total(&mut owner, 0.), one(&mut owner));
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
}
#[test]
fn analysis_sums_current_bodies_without_union_and_tracks_edit_undo_and_failure() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, true);
    let both = total(&mut owner, 2.);
    assert!((both.volume_mm3 - 48000.).abs() < 1e-8);
    assert!((both.mass_g - 96.).abs() < 1e-10);
    assert!((both.inertia_yy_gmm2 - 16000.).abs() < 1e-8);
    set_parameter(&mut owner, "depth", "3 cm");
    assert!((total(&mut owner, 2.).volume_mm3 - 60000.).abs() < 1e-8);
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(total(&mut owner, 2.), both);
    set_parameter(&mut owner, "depth", "0 cm");
    assert_eq!(total(&mut owner, 0.), one(&mut owner));
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
}
#[test]
fn model_health_lists_only_the_failed_feature_after_current_replay() {
    let (mut owner, _) = owner();
    open_prism(&mut owner, false);
    assert_eq!(
        health(&mut owner),
        ModelHealthResult {
            overall: "ok".into(),
            ..Default::default()
        }
    );
    set_parameter(&mut owner, "depth", "0 cm");
    let result = health(&mut owner);
    assert_eq!((result.overall.as_str(), result.sick_count), ("sick", 1));
    let unhealthy = result.unhealthy.unwrap();
    assert_eq!((unhealthy.len(), unhealthy[0].status.as_str()), (1, "sick"));
    assert!(!unhealthy[0].name.is_empty());
    assert!(!unhealthy[0].reason.is_empty());
}
#[test]
fn requires_update_tracks_the_current_native_evaluation_generation() {
    let (mut empty_owner, awake) = owner();
    create(&mut empty_owner, &awake);
    assert!(!requires_update(&mut empty_owner));
    empty_owner.finish().unwrap();
    let (mut owner, _) = owner();
    open_prism(&mut owner, false);
    total(&mut owner, 0.);
    assert!(!requires_update(&mut owner));
    set_parameter(&mut owner, "depth", "3 cm");
    assert!(requires_update(&mut owner));
    total(&mut owner, 0.);
    assert!(!requires_update(&mut owner));
}
#[test]
fn update_refreshes_a_healthy_part_and_reports_or_refuses_failed_features() {
    let (mut owner, _) = owner();
    open_prism(&mut owner, false);
    assert_eq!(update(&mut owner, false), UpdateDocumentResult::default());
    assert_eq!(rebuild(&mut owner), UpdateDocumentResult::default());
    set_parameter(&mut owner, "depth", "0 cm");
    assert!(
        Client::new(owner.document_host())
            .documents()
            .update(&UpdateDocumentArgs::default())
            .is_err()
    );
    let result = update(&mut owner, true);
    assert!(!result.requires_update);
    let errors = result.errors.unwrap();
    assert_eq!((errors.len(), errors[0].kind.as_str()), (1, "extrude"));
    assert!(!errors[0].name.is_empty() && !errors[0].message.is_empty());
    assert!(!requires_update(&mut owner));
}
#[test]
fn end_of_part_round_trips_through_history_and_the_native_evaluation() {
    let (mut owner, _) = owner();
    open_prism(&mut owner, true);
    assert_eq!(
        end_of_part(&mut owner),
        EndOfPartResult {
            position: -1,
            rolled_back: false
        }
    );
    let rolled_back = Client::new(owner.document_host())
        .documents()
        .set_end_of_part(&SetEndOfPartArgs { position: 1 })
        .unwrap();
    assert_eq!(
        rolled_back,
        EndOfPartResult {
            position: 1,
            rolled_back: true
        }
    );
    assert_eq!(total(&mut owner, 0.).volume_mm3, 24000.);
    let restored = Client::new(owner.document_host())
        .documents()
        .set_end_of_part(&SetEndOfPartArgs { position: -1 })
        .unwrap();
    assert_eq!(
        restored,
        EndOfPartResult {
            position: -1,
            rolled_back: false
        }
    );
    assert_eq!(total(&mut owner, 0.).volume_mm3, 48000.);
    assert!(
        Client::new(owner.document_host())
            .documents()
            .set_end_of_part(&SetEndOfPartArgs { position: 3 })
            .is_err()
    );
    assert_eq!(end_of_part(&mut owner), restored);
}
#[test]
fn feature_lifecycle_edits_native_history_and_returns_current_detail() {
    let (mut owner, _) = owner();
    open_prism(&mut owner, true);
    let initial = feature(&mut owner, 1);
    assert_eq!(
        (
            initial.feature.index,
            initial.feature.feature_info.kind.as_str()
        ),
        (0, "extrude")
    );
    let renamed = Client::new(owner.document_host())
        .features()
        .rename(&RenameFeatureArgs {
            id: 1,
            name: "Base".into(),
        })
        .unwrap();
    assert_eq!(renamed.feature.feature_info.name, "Base");
    assert!(
        Client::new(owner.document_host())
            .features()
            .rename(&RenameFeatureArgs {
                id: 2,
                name: "Base".into()
            })
            .is_err()
    );
    let suppressed = Client::new(owner.document_host())
        .features()
        .set_suppressed(&SetFeatureSuppressedArgs {
            id: 1,
            suppressed: true,
        })
        .unwrap();
    assert_eq!(
        (
            suppressed.feature.feature_info.health.as_str(),
            total(&mut owner, 0.).volume_mm3
        ),
        ("suppressed", 24000.)
    );
    let restored = Client::new(owner.document_host())
        .features()
        .set_suppressed(&SetFeatureSuppressedArgs {
            id: 1,
            suppressed: false,
        })
        .unwrap();
    assert!(restored.feature.feature_info.health.is_empty());
    let moved = Client::new(owner.document_host())
        .features()
        .reorder(&ReorderFeatureArgs {
            id: 2,
            new_index: 0,
        })
        .unwrap();
    assert_eq!(moved.feature.index, 0);
    assert_eq!(total(&mut owner, 0.).volume_mm3, 48000.);
    assert_eq!(
        Client::new(owner.document_host())
            .features()
            .delete(&FeatureRefArgs { id: 1 })
            .unwrap(),
        DeleteFeatureResult {
            id: 1,
            deleted: true
        }
    );
    assert_eq!(total(&mut owner, 0.).volume_mm3, 24000.);
    assert!(
        Client::new(owner.document_host())
            .features()
            .get(&FeatureRefArgs { id: 1 })
            .is_err()
    );
}
#[test]
fn part_analysis_uses_assigned_density_and_validates_requests() {
    let (mut owner, _) = owner();
    assert!(
        Client::new(owner.document_host())
            .analysis()
            .mass_properties(&MassPropertiesArgs::default())
            .is_err()
    );
    open_prism(&mut owner, false);
    Client::new(owner.document_host())
        .materials()
        .assign(&AssignMaterialArgs {
            body_key: String::new(),
            material_id: "steel".into(),
        })
        .unwrap();
    assert_eq!(total(&mut owner, 0.), one(&mut owner));
    assert!(
        Client::new(owner.document_host())
            .analysis()
            .mass_properties(&MassPropertiesArgs {
                accuracy: "unknown".into(),
                ..Default::default()
            })
            .is_err()
    );
    assert!(
        Client::new(owner.document_host())
            .analysis()
            .mass_properties(&MassPropertiesArgs {
                density_g_cm3: f64::MAX,
                ..Default::default()
            })
            .is_err()
    );
}
