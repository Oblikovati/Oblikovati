// SPDX-License-Identifier: GPL-2.0-only
//! Typed blends reuse the current document replay for edge ownership.
use super::*;
use oblikovati_automation::{
    CertifiedFilletAccess, FileHost, FilletAccess, PartReplayService, PartUpdateAccess,
};
use oblikovati_model::{
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::{DocumentGuid, SemanticReferenceName},
};
fn count_blend_replays(reuse: bool, chamfer: bool) -> (u64, u64, u64) {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let text = counted_edge_key(&owner, document);
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let backend = CertifiedFilletAccess::new(policy).unwrap();
    verify_declines_do_not_count_replays(&owner, document, &backend);
    let mut replay = PartReplayService::with_policy(policy).unwrap();
    replay.update_part(&mut owner.files, document).unwrap();
    let initial = replay.replay_count();
    create_counted_blend(&mut owner, &backend, &mut replay, text, reuse, chamfer);
    replay.update_part(&mut owner.files, document).unwrap();
    let (body, validation) = backend.replay_counts();
    (body, validation, replay.replay_count() - initial)
}

fn verify_declines_do_not_count_replays(
    owner: &TestOwner,
    document: u64,
    backend: &CertifiedFilletAccess,
) {
    let workspace = owner.files.workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    assert!(backend.body(session.history(), guid, &[]).is_err());
    assert!(
        backend
            .validate(guid, session.snapshot(), FeatureId(1))
            .is_err()
    );
    assert_eq!(backend.replay_counts(), (0, 0));
}

fn counted_edge_key(owner: &TestOwner, document: u64) -> String {
    let guid = owner
        .files
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    let edge =
        SemanticReferenceName::new(guid, 1, 2, "sketch-prism", "start-edge-curve", &[1, 101])
            .unwrap()
            .encode()
            .unwrap();
    edge.iter().map(|byte| format!("{byte:02x}")).collect()
}

fn create_counted_blend(
    owner: &mut TestOwner,
    backend: &CertifiedFilletAccess,
    replay: &mut PartReplayService,
    text: String,
    reuse: bool,
    chamfer: bool,
) {
    let host =
        FileHost::new(&mut owner.files, &mut owner.guids, &owner.version).with_fillet(backend);
    let host = if reuse {
        host.with_part_update(replay)
    } else {
        host
    };
    let client = Client::new(host);
    if chamfer {
        client
            .features()
            .add_feature(&oblikovati_api::wire::featureargs::Chamfer {
                edge_refs: Some(vec![text]),
                distance: "1 mm".into(),
                ..Default::default()
            })
            .unwrap();
    } else {
        client
            .features()
            .add_feature(&oblikovati_api::wire::featureargs::Fillet {
                edge_refs: Some(vec![text]),
                radius: "1 mm".into(),
                ..Default::default()
            })
            .unwrap();
    }
}

#[test]
fn fillet_creation_counts_body_candidate_and_published_replays() {
    let before = count_blend_replays(false, false);
    let current = count_blend_replays(true, false);
    assert_eq!(before, (1, 1, 1));
    assert_eq!(current, (0, 0, 1));
}

#[test]
fn chamfer_creation_counts_body_candidate_and_published_replays() {
    assert_eq!(count_blend_replays(false, true), (1, 1, 1));
    assert_eq!(count_blend_replays(true, true), (0, 0, 1));
}

struct RejectCandidate(CertifiedFilletAccess);
impl FilletAccess for RejectCandidate {
    fn accepts_policy(&self, policy: Option<oblikovati_martensite::PrismResolutionPolicy>) -> bool {
        self.0.accepts_policy(policy)
    }
    fn body(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        edges: &[Vec<u8>],
    ) -> Result<Vec<u8>, String> {
        self.0.body(history, document, edges)
    }
    fn validate(
        &self,
        _: DocumentGuid,
        _: &DocumentSnapshot,
        _: FeatureId,
    ) -> Result<usize, String> {
        Err("injected candidate refusal".into())
    }
    fn validate_chamfer(
        &self,
        _: DocumentGuid,
        _: &DocumentSnapshot,
        _: FeatureId,
    ) -> Result<usize, String> {
        Err("injected candidate refusal".into())
    }
}

#[test]
fn injected_validator_still_rejects_a_candidate_with_a_current_prefix() {
    assert_injected_rejection(false);
    assert_injected_rejection(true);
}

fn assert_injected_rejection(chamfer: bool) {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let text = counted_edge_key(&owner, document);
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let backend = RejectCandidate(CertifiedFilletAccess::new(policy).unwrap());
    let mut replay = PartReplayService::with_policy(policy).unwrap();
    replay.update_part(&mut owner.files, document).unwrap();
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let host = FileHost::new(&mut owner.files, &mut owner.guids, &owner.version)
        .with_fillet(&backend)
        .with_part_update(&mut replay);
    let client = Client::new(host);
    let result = if chamfer {
        client
            .features()
            .add_feature(&oblikovati_api::wire::featureargs::Chamfer {
                edge_refs: Some(vec![text]),
                distance: "1 mm".into(),
                ..Default::default()
            })
    } else {
        client
            .features()
            .add_feature(&oblikovati_api::wire::featureargs::Fillet {
                edge_refs: Some(vec![text]),
                radius: "1 mm".into(),
                ..Default::default()
            })
    };
    assert!(
        result
            .unwrap_err()
            .to_string()
            .contains("injected candidate refusal")
    );
    let session = owner.files.workspace().session(document).unwrap();
    assert_eq!(session.generation(), generation);
    assert_eq!(session.snapshot().work_geometry.features().len(), 1);
}

#[test]
fn rejected_fillet_tail_does_not_commit_or_invalidate_current_result() {
    reject_large_blend(false);
}

#[test]
fn rejected_chamfer_tail_does_not_commit_or_invalidate_current_result() {
    reject_large_blend(true);
}

fn reject_large_blend(chamfer: bool) {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let text = counted_edge_key(&owner, document);
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let backend = CertifiedFilletAccess::new(policy).unwrap();
    let mut replay = PartReplayService::with_policy(policy).unwrap();
    replay.update_part(&mut owner.files, document).unwrap();
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let host = FileHost::new(&mut owner.files, &mut owner.guids, &owner.version)
        .with_fillet(&backend)
        .with_part_update(&mut replay);
    let client = Client::new(host);
    let result = if chamfer {
        client
            .features()
            .add_feature(&oblikovati_api::wire::featureargs::Chamfer {
                edge_refs: Some(vec![text]),
                distance: "1000 mm".into(),
                ..Default::default()
            })
    } else {
        client
            .features()
            .add_feature(&oblikovati_api::wire::featureargs::Fillet {
                edge_refs: Some(vec![text]),
                radius: "1000 mm".into(),
                ..Default::default()
            })
    };
    assert!(result.is_err());
    let session = owner.files.workspace().session(document).unwrap();
    assert_eq!(session.generation(), generation);
    assert_eq!(session.snapshot().work_geometry.features().len(), 1);
    assert!(replay.current_part(&owner.files, document).is_ok());
    assert_eq!(backend.replay_counts(), (0, 0));
}

struct NoBodyReplay(CertifiedFilletAccess);
impl FilletAccess for NoBodyReplay {
    fn accepts_policy(&self, policy: Option<oblikovati_martensite::PrismResolutionPolicy>) -> bool {
        self.0.accepts_policy(policy)
    }
    fn body(
        &self,
        _: &DocumentEditHistory,
        _: DocumentGuid,
        _: &[Vec<u8>],
    ) -> Result<Vec<u8>, String> {
        Err("unexpected independent body replay".into())
    }
    fn validate(
        &self,
        document: DocumentGuid,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
    ) -> Result<usize, String> {
        self.0.validate(document, candidate, feature)
    }
    fn validate_chamfer(
        &self,
        document: DocumentGuid,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
    ) -> Result<usize, String> {
        self.0.validate_chamfer(document, candidate, feature)
    }
}

#[test]
fn typed_blends_resolve_edges_from_the_current_replay() {
    for (chamfer, headless) in [(false, false), (true, false), (false, true), (true, true)] {
        let (owner, _) = owner();
        let mut owner = owner
            .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
            .unwrap();
        let document = open_prism(&mut owner, false);
        owner.take_scene().unwrap().unwrap();
        let guid = owner
            .files
            .workspace()
            .identity(document)
            .unwrap()
            .internal_name;
        let edge =
            SemanticReferenceName::new(guid, 1, 2, "sketch-prism", "start-edge-curve", &[1, 101])
                .unwrap()
                .encode()
                .unwrap();
        let text: String = edge.iter().map(|byte| format!("{byte:02x}")).collect();
        let backend = NoBodyReplay(
            CertifiedFilletAccess::new(crate::part_scene::resolution_policy::desktop_policy())
                .unwrap(),
        );
        let mut replay =
            PartReplayService::with_policy(crate::part_scene::resolution_policy::desktop_policy())
                .unwrap();
        if headless {
            replay.update_part(&mut owner.files, document).unwrap();
        }
        let host =
            FileHost::new(&mut owner.files, &mut owner.guids, &owner.version).with_fillet(&backend);
        let host = if headless {
            host.with_part_update(&mut replay)
        } else {
            host.with_part_topology(&mut owner.part_scenes)
        };
        let client = Client::new(host);
        if chamfer {
            let request = oblikovati_api::wire::featureargs::Chamfer {
                edge_refs: Some(vec![text]),
                distance: "1 mm".into(),
                ..Default::default()
            };
            client.features().add_feature(&request).unwrap();
        } else {
            let request = oblikovati_api::wire::featureargs::Fillet {
                edge_refs: Some(vec![text]),
                radius: "1 mm".into(),
                ..Default::default()
            };
            client.features().add_feature(&request).unwrap();
        }
    }
}

#[test]
fn typed_blends_decline_replay_owner_with_wrong_policy() {
    for (chamfer, absent) in [(false, true), (true, true), (false, false), (true, false)] {
        assert_wrong_policy_uses_certified_backend(chamfer, absent);
    }
}

fn assert_wrong_policy_uses_certified_backend(chamfer: bool, absent: bool) {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let guid = owner
        .files
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    let edge =
        SemanticReferenceName::new(guid, 1, 2, "sketch-prism", "start-edge-curve", &[1, 101])
            .unwrap()
            .encode()
            .unwrap();
    let text: String = edge.iter().map(|byte| format!("{byte:02x}")).collect();
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let alternate = match policy {
        oblikovati_martensite::PrismResolutionPolicy::ModelRelative {
            reference,
            reference_size,
            minimum_size,
        } => oblikovati_martensite::PrismResolutionPolicy::ModelRelative {
            reference,
            reference_size: reference_size * 2.,
            minimum_size,
        },
        _ => panic!("desktop modelling policy: expected model-relative bounds"),
    };
    let mut replay = if absent {
        PartReplayService::default()
    } else {
        PartReplayService::with_policy(alternate).unwrap()
    };
    replay.update_part(&mut owner.files, document).unwrap();
    let backend = NoBodyReplay(CertifiedFilletAccess::new(policy).unwrap());
    let host = FileHost::new(&mut owner.files, &mut owner.guids, &owner.version)
        .with_fillet(&backend)
        .with_part_update(&mut replay);
    let client = Client::new(host);
    let result = if chamfer {
        client
            .features()
            .add_feature(&oblikovati_api::wire::featureargs::Chamfer {
                edge_refs: Some(vec![text.clone()]),
                distance: "1 mm".into(),
                ..Default::default()
            })
    } else {
        client
            .features()
            .add_feature(&oblikovati_api::wire::featureargs::Fillet {
                edge_refs: Some(vec![text.clone()]),
                radius: "1 mm".into(),
                ..Default::default()
            })
    };
    let error = result.unwrap_err().to_string();
    assert!(
        error.contains("unexpected independent body replay"),
        "chamfer={chamfer}, absent={absent}: {error}"
    );
    drop(client);
    let certified = CertifiedFilletAccess::new(policy).unwrap();
    let host = FileHost::new(&mut owner.files, &mut owner.guids, &owner.version)
        .with_fillet(&certified)
        .with_part_update(&mut replay);
    let client = Client::new(host);
    if chamfer {
        client
            .features()
            .add_feature(&oblikovati_api::wire::featureargs::Chamfer {
                edge_refs: Some(vec![text]),
                distance: "1 mm".into(),
                ..Default::default()
            })
            .unwrap();
    } else {
        client
            .features()
            .add_feature(&oblikovati_api::wire::featureargs::Fillet {
                edge_refs: Some(vec![text]),
                radius: "1 mm".into(),
                ..Default::default()
            })
            .unwrap();
    }
}
