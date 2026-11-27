// SPDX-License-Identifier: GPL-2.0-only
//! Verify validation remains authoritative for checkpointed edits.
use super::*;

#[test]
fn injected_validator_still_rejects_a_checkpointed_blend_edit() {
    let (owner, _) = owner();
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let mut owner = owner.with_extrusion_policy(policy).unwrap();
    let document = open_prism(&mut owner, false);
    let edge = counted_edge_key(&owner, document);
    let id = Client::new(owner.document_host())
        .features()
        .add_feature(&oblikovati_api::wire::featureargs::Fillet {
            edge_refs: Some(vec![edge]),
            radius: "1 mm".into(),
            ..Default::default()
        })
        .unwrap()
        .0
        .unwrap()["id"]
        .as_u64()
        .unwrap();
    owner
        .part_scenes
        .ensure_edit_checkpoints(&mut owner.files, document)
        .unwrap();
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let request = oblikovati_api::wire::EditFeatureArgs {
        id,
        scalars: Some(vec![oblikovati_api::wire::ScalarEdit {
            index: 0,
            value: "2 mm".into(),
        }]),
        ..Default::default()
    };
    let workspace = owner.files.workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    let other: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let prefix = session
        .owned_checkpoint_slot(Some(policy))
        .unwrap()
        .candidate_prefix(session)
        .unwrap();
    let replay = workspace
        .owned_checkpoint_replay(document, Some(policy), prefix)
        .unwrap();
    assert!(replay.owns_identity(guid));
    assert!(!replay.owns_identity(other));
    session
        .preview_projected(
            |candidate, _| oblikovati_automation::apply_fillet_radius_edit(candidate, &request),
            |candidate, feature| {
                assert!(
                    replay
                        .prepare_feature(candidate, session.history(), feature)
                        .unwrap()
                        .is_some()
                );
                let error = replay
                    .prepare_feature(candidate, session.history(), FeatureId(u64::MAX))
                    .err()
                    .unwrap();
                assert!(error.to_string().contains("valid current descendant body"));
                let certified = CertifiedFilletAccess::new(policy).unwrap();
                assert!(
                    certified
                        .capture_with_checkpoints(
                            oblikovati_model::feature::part::PartBlendKind::Fillet,
                            session.history(),
                            other,
                            candidate,
                            feature,
                            &replay,
                        )
                        .err()
                        .unwrap()
                        .contains("checkpoint owner identity")
                );
                Ok::<_, oblikovati_api::Error>(())
            },
        )
        .unwrap();
    let backend = RejectCandidate(CertifiedFilletAccess::new(policy).unwrap());
    let host = FileHost::new(&mut owner.files, &mut owner.guids, &owner.version)
        .with_fillet(&backend)
        .with_part_topology(&mut owner.part_scenes);
    let error = Client::new(host)
        .features()
        .edit(&request)
        .unwrap_err()
        .to_string();
    assert!(error.contains("injected candidate refusal"));
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    assert!(
        owner
            .part_scenes
            .cached_checkpoint_prefix(&owner.files, document)
            .unwrap()
            .is_some()
    );
}
