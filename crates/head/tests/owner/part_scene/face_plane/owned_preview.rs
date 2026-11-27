// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn face_datum_preview_uses_owned_checkpoints_without_committing() {
    use oblikovati_automation::PartUpdateAccess;
    use oblikovati_model::feature::work_plane::{
        WorkPlaneFaceReference, WorkPlaneReference, WorkPlaneResolver,
    };
    use oblikovati_model::identity::ReferenceEnvelope;
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_policy(policy).unwrap();
    let document = open_prism(&mut owner, false);
    owner.take_scene().unwrap().unwrap();
    let reference = published_keys(&owner, document, FeatureId(1))
        .into_iter()
        .filter(|key| ReferenceEnvelope::decode(key).unwrap().kind == 1)
        .map(|key| WorkPlaneFaceReference::new(key).unwrap())
        .find(|face| {
            owner
                .part_scenes
                .resolve_planar_face(&mut owner.files, document, face)
                .is_ok_and(|plane| {
                    (plane.recipe().origin[2] - 2.).abs() < 1e-9 && plane.normal()[2] > 0.99
                })
        })
        .unwrap();
    owner
        .files
        .ensure_owned_checkpoints(document, Some(policy), &mut 0)
        .unwrap();
    let workspace = owner.files.workspace();
    let session = workspace.session(document).unwrap();
    let generation = session.generation();
    session
        .preview_projected(
            |candidate, ids| {
                let id = ids.allocate_feature().unwrap();
                let plane = oblikovati_model::feature::work_plane::WorkPlaneDefinition::Offset {
                    base: WorkPlaneReference::Face(reference),
                    distance: DatumLength::Constant(0.5),
                };
                candidate
                    .work_geometry
                    .append(AuthoredFeature::datum(id, "Preview", plane))?;
                Ok::<_, oblikovati_model::feature::FeatureSequenceError>(id)
            },
            |candidate, id| {
                let owned = workspace
                    .preview_owned_part(document, Some(policy), candidate)
                    .unwrap();
                assert!(owned.used_checkpoint);
                let datums = oblikovati_automation::preview_part_datums(
                    &owner.files,
                    document,
                    candidate,
                    Some(policy),
                )
                .unwrap();
                assert!((datums.plane(id).unwrap().recipe().origin[2] - 2.5).abs() < 1e-9);
                assert_eq!(
                    datums.plane(id).unwrap().recipe().origin,
                    owned
                        .evaluation
                        .datum_evaluation()
                        .plane(id)
                        .unwrap()
                        .recipe()
                        .origin
                );
                Ok::<_, oblikovati_model::feature::FeatureSequenceError>(())
            },
        )
        .unwrap();
    assert_eq!(
        workspace.session(document).unwrap().generation(),
        generation
    );
}
