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

struct NoBodyReplay(CertifiedFilletAccess);
impl FilletAccess for NoBodyReplay {
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
