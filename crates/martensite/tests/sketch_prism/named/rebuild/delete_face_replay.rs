// SPDX-License-Identifier: GPL-2.0-only
//! Real-kernel verification of Part Modify Delete Face (open-shell conversion and defeaturing heal, #3918).
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        AuthoredFeature,
        delete_face::{DeleteFaceDefinition, DeleteFaceRecipe},
        fillet::FilletDefinition,
        sketch_prism::{SketchDirectedCurve, SketchPrismSelection},
        work_plane::WorkPlaneOrigin,
    },
    identity::ReferenceEnvelope,
    sketch::{Sketch2D, SketchEntityId},
};

fn cylinder_document() -> (DocumentSnapshot, FeatureId) {
    let mut snapshot = selected_prism_document();
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(201), [1.0, 1.5]).unwrap();
    geometry
        .add_circle(SketchEntityId(202), SketchEntityId(201), 0.5)
        .unwrap();
    let sketch = DocumentSketchId(2);
    snapshot
        .sketches
        .add_on_origin_plane(sketch, "CylinderSketch", WorkPlaneOrigin::XY, geometry)
        .unwrap();
    let selection = SketchPrismSelection {
        outer: vec![SketchDirectedCurve {
            curve: SketchEntityId(202),
            reversed: false,
        }],
        holes: vec![],
    };
    let definition = SketchPrismDefinition::new(
        sketch,
        selection,
        [0.0, 0.0, 2.0].map(DatumLength::Constant),
    )
    .unwrap();
    let reads = definition.reads(&snapshot).unwrap();
    let cyl_id = FeatureId(2);
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            cyl_id,
            "Extrusion2",
            PartFeatureDefinition::SketchPrism(Box::new(definition)),
            reads,
        ))
        .unwrap();
    solve_selected_sketches(&mut snapshot);
    (snapshot, cyl_id)
}

#[test]
fn unhealed_delete_face_converts_cylinder_solid_to_two_face_surface_body() {
    let (doc_snapshot, cyl_id) = cylinder_document();
    let mut history = owner(doc_snapshot);
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let before = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let cyl = before.body(cyl_id).expect("expected cylinder solid");
    assert!(!cyl.is_surface());
    let body_key = cyl
        .body_reference_key(&history, document())
        .unwrap()
        .to_vec();
    let faces = cyl.face_reference_keys(&history, document()).unwrap();
    assert_eq!(
        faces.len(),
        3,
        "cylinder should have 3 faces before deletion"
    );
    let lower_cap = faces
        .into_iter()
        .find(|face| {
            cyl.face_topology(&history, document(), face)
                .unwrap()
                .surface
                == PrismSurfaceKind::Plane
                && cyl.face_bounds_center(&history, document(), face).unwrap()[2].abs() < 1e-6
        })
        .expect("expected lower circular cap at Z = 0");

    let def = DeleteFaceDefinition::from_recipe(DeleteFaceRecipe {
        version: 1,
        body: body_key.clone(),
        faces: vec![lower_cap],
        heal_remaining_faces: false,
        select_lump_or_void: false,
    })
    .unwrap();
    let token = history.begin("Delete Face").unwrap();
    let id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_delete_face("Delete Face1", def, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let after = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        after
            .health()
            .iter()
            .all(|record| record.status == FeatureEvaluationStatus::Healthy),
        "health: {:?}",
        after.health()
    );
    let surface = after.body(id).expect("expected open surface body");
    assert!(
        surface.is_surface(),
        "deleting a solid face without healing must convert the body to a surface body"
    );
    let summary = surface
        .topology_summary(&history, document(), &body_key)
        .unwrap();
    assert!(
        !summary.solid,
        "open shell must not be classified as a solid"
    );
    assert_eq!(summary.faces, 2, "cylinder face count should drop 3 -> 2");
}

#[test]
fn healed_delete_face_fails_authentically_on_unhealable_cylinder_end_cap() {
    let (doc_snapshot, cyl_id) = cylinder_document();
    let mut history = owner(doc_snapshot);
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let before = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let cyl = before.body(cyl_id).expect("expected cylinder solid");
    let body_key = cyl
        .body_reference_key(&history, document())
        .unwrap()
        .to_vec();
    let lower_cap = cyl
        .face_reference_keys(&history, document())
        .unwrap()
        .into_iter()
        .find(|face| {
            cyl.face_topology(&history, document(), face)
                .unwrap()
                .surface
                == PrismSurfaceKind::Plane
                && cyl.face_bounds_center(&history, document(), face).unwrap()[2].abs() < 1e-6
        })
        .expect("expected lower circular cap");

    let def = DeleteFaceDefinition::from_recipe(DeleteFaceRecipe {
        version: 1,
        body: body_key,
        faces: vec![lower_cap],
        heal_remaining_faces: true,
        select_lump_or_void: false,
    })
    .unwrap();
    let token = history.begin("Delete Face").unwrap();
    let id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_delete_face("Delete Face1", def, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let after = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(after.body(id).is_none());
    assert!(
        after.body(cyl_id).is_some(),
        "failed delete face must preserve source solid body"
    );
    let status = format!(
        "{:?}",
        after
            .health()
            .iter()
            .find(|record| record.feature == id)
            .expect("expected Delete Face health record")
            .status
    );
    assert!(
        status.contains("Create DelFace feature failed"),
        "expected authentic defeaturing failure, saw {status}"
    );
}

#[test]
fn healed_delete_face_removes_fillet_face_and_restores_cube_volume() {
    let mut history = owner(cube_document());
    let edge = certified(&history)
        .subentity_reference_keys(&history, document())
        .unwrap()
        .into_iter()
        .find(|key| ReferenceEnvelope::decode(key).unwrap().kind == 2)
        .unwrap();
    let fillet =
        FilletDefinition::new(key("body", &[1]), vec![edge], DatumLength::Constant(0.1)).unwrap();
    let token = history.begin("Fillet").unwrap();
    let fillet_id = history.create_fillet(&token, "Fillet1", fillet).unwrap();
    history.commit(&token, false).unwrap();

    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let filleted = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let body = filleted.body(fillet_id).expect("expected filleted cube");
    let blend_face = body
        .face_reference_keys(&history, document())
        .unwrap()
        .into_iter()
        .find(|face| {
            body.face_topology(&history, document(), face)
                .unwrap()
                .surface
                == PrismSurfaceKind::Cylinder
        })
        .expect("expected cylindrical fillet face");

    let def = DeleteFaceDefinition::from_recipe(DeleteFaceRecipe {
        version: 1,
        body: key("body", &[1]),
        faces: vec![blend_face],
        heal_remaining_faces: true,
        select_lump_or_void: false,
    })
    .unwrap();
    let token = history.begin("Delete Face").unwrap();
    let delete_id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_delete_face("Delete Face1", def, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let healed = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        healed
            .health()
            .iter()
            .all(|record| record.status == FeatureEvaluationStatus::Healthy),
        "health: {:?}",
        healed.health()
    );
    let result = healed.body(delete_id).expect("expected healed solid");
    assert!(!result.is_surface(), "healed solid must remain a solid");
    let vol = result.volume(&history, document()).unwrap();
    assert!(
        (vol - 1.0).abs() < 1e-6,
        "healed removal of fillet should restore unit cube volume 1.0, got {vol}"
    );
}
