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
use std::cell::Cell;

struct CountedFillet {
    inner: CertifiedFilletAccess,
    body_replays: Cell<u64>,
    validation_replays: Cell<u64>,
}
impl FilletAccess for CountedFillet {
    fn accepts_policy(&self, policy: Option<oblikovati_martensite::PrismResolutionPolicy>) -> bool {
        self.inner.accepts_policy(policy)
    }
    fn body(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        edges: &[Vec<u8>],
    ) -> Result<Vec<u8>, String> {
        self.body_replays.set(self.body_replays.get() + 1);
        self.inner.body(history, document, edges)
    }
    fn validate(
        &self,
        document: DocumentGuid,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
    ) -> Result<usize, String> {
        self.validation_replays
            .set(self.validation_replays.get() + 1);
        self.inner.validate(document, candidate, feature)
    }
}

fn count_fillet_replays(reuse: bool) -> (u64, u64, u64) {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let text = counted_edge_key(&owner, document);
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let backend = CountedFillet {
        inner: CertifiedFilletAccess::new(policy).unwrap(),
        body_replays: Cell::new(0),
        validation_replays: Cell::new(0),
    };
    let mut replay = PartReplayService::with_policy(policy).unwrap();
    replay.update_part(&mut owner.files, document).unwrap();
    let initial = replay.replay_count();
    create_counted_fillet(&mut owner, &backend, &mut replay, text, reuse);
    replay.update_part(&mut owner.files, document).unwrap();
    (
        backend.body_replays.get(),
        backend.validation_replays.get(),
        replay.replay_count() - initial,
    )
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

fn create_counted_fillet(
    owner: &mut TestOwner,
    backend: &CountedFillet,
    replay: &mut PartReplayService,
    text: String,
    reuse: bool,
) {
    let host =
        FileHost::new(&mut owner.files, &mut owner.guids, &owner.version).with_fillet(backend);
    let host = if reuse {
        host.with_part_update(replay)
    } else {
        host
    };
    Client::new(host)
        .features()
        .add_feature(&oblikovati_api::wire::featureargs::Fillet {
            edge_refs: Some(vec![text]),
            radius: "1 mm".into(),
            ..Default::default()
        })
        .unwrap();
}

#[test]
fn fillet_creation_counts_body_candidate_and_published_replays() {
    let before = count_fillet_replays(false);
    let current = count_fillet_replays(true);
    assert_eq!(before, (1, 1, 1));
    assert_eq!(current, (0, 1, 1));
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
