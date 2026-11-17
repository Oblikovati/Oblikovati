// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::{
    part::PartFeatureDefinition, sketch_prism::SketchPrismDefinition, work_geometry::DatumLength,
};
use oblikovati_persistence::package::DocumentPackage;
fn holed_document(rotation: usize) -> DocumentSnapshot {
    let mut snapshot = selected_prism_document();
    add_fixed_sketch_hole(&mut snapshot);
    let mut selection = selected_sketch_boundary();
    selection.outer.rotate_left(rotation);
    selection.holes.push(selected_sketch_line_ring(111));
    let definition =
        SketchPrismDefinition::new(SKETCH, selection, [1., 2., -3.].map(DatumLength::Constant))
            .unwrap();
    let reads = definition.reads(&snapshot).unwrap();
    snapshot
        .work_geometry
        .replace(
            FeatureId(1),
            PartFeatureDefinition::SketchPrism(Box::new(definition)),
            reads,
        )
        .unwrap();
    solve_selected_sketches(&mut snapshot);
    snapshot
}
#[test]
fn hole_and_oblique_negative_sweep_names_ignore_traversal_order() {
    let first = owner(holed_document(0));
    let second = owner(holed_document(2));
    let left = named(&first);
    let right = named(&second);
    let keys = left.reference_keys(&first, document()).unwrap();
    assert_eq!(keys, right.reference_keys(&second, document()).unwrap());
    assert_eq!(keys.len(), 11);
    for key in keys
        .into_iter()
        .filter(|key| key != &super::key("body", &[1]))
    {
        let a = left.face_report(&first, document(), &key).unwrap();
        let b = right.face_report(&second, document(), &key).unwrap();
        assert_eq!(a.role, b.role);
        assert_analytic_scalar(a.area, b.area);
        assert_eq!(a.loops, b.loops);
    }
    for role in ["start", "end"] {
        let face = left
            .face_report(&first, document(), &key(role, &[1]))
            .unwrap();
        assert_analytic_scalar(face.area, 10.);
        assert_eq!(face.loops, 2);
        let expected_z = if role == "start" { 0. } else { -3. };
        assert_centroid([0., 0., face.centroid[2]], [0., 0., expected_z]);
    }
}
fn package(snapshot: &DocumentSnapshot) -> DocumentPackage {
    let high_water = snapshot
        .work_geometry
        .features()
        .iter()
        .map(|feature| feature.id().0)
        .max()
        .unwrap_or(0);
    DocumentPackage::decode(b"schemaVersion: 3\ndocumentType: 1\n")
        .unwrap()
        .with_parameters(&snapshot.parameters, snapshot.parameters.units().clone())
        .unwrap()
        .with_sketches(&snapshot.sketches, 1, 104)
        .unwrap()
        .with_work_geometry(&snapshot.work_geometry, high_water)
        .unwrap()
}
fn restored_owner(snapshot: &DocumentSnapshot) -> DocumentEditHistory {
    let bytes = package(snapshot).encode().unwrap();
    let reopened = DocumentPackage::decode(&bytes).unwrap();
    let mut snapshot = DocumentSnapshot {
        parameters: reopened.restore_document_parameters().unwrap().parameters,
        sketches: reopened.restore_sketches().unwrap().sketches,
        work_geometry: reopened.restore_work_geometry().unwrap().program,
        assets: Default::default(),
        model_states: None,
        assembly: None,
        drawing: None,
        body_names: Default::default(),
        design_views: Default::default(),
        sketch_settings: None,
        properties: Default::default(),
        attributes: Default::default(),
        attachments: Default::default(),
        interests: Default::default(),
        resources: Default::default(),
    };
    solve_selected_sketches(&mut snapshot);
    owner(snapshot)
}
#[test]
fn saved_recipes_rebuild_the_same_keys_under_a_new_history_owner() {
    let original = owner(selected_prism_document());
    let before = named(&original);
    let keys = before.reference_keys(&original, document()).unwrap();
    let restored = restored_owner(original.snapshot());
    let after = named(&restored);
    assert_eq!(keys, after.reference_keys(&restored, document()).unwrap());
    assert_faces(&after, &restored, 2.);
    assert!(before.reference_keys(&restored, document()).is_err());
}
fn certified_resolution() -> oblikovati_martensite::PrismResolution {
    oblikovati_martensite::PrismResolution {
        version: 1,
        weld: 1e-6,
        sew: 1e-7,
        angular: 1e-9,
        display_chordal: 1e-3,
        display_angular: 0.1,
        maximum_tolerance: 1e-5,
    }
}
fn certified(history: &DocumentEditHistory) -> NamedPrismSnapshot {
    oblikovati_martensite::retain_certified_named_sketch_prism(
        history,
        document(),
        FeatureId(1),
        certified_resolution(),
    )
    .unwrap()
}
fn cube_document() -> DocumentSnapshot {
    use oblikovati_model::sketch::{PlanarConstraint, Sketch2D};
    let mut snapshot = selected_prism_document();
    let mut recipe = snapshot.sketches.get(SKETCH).unwrap().geometry().recipe();
    for constraint in &mut recipe.constraints {
        if let PlanarConstraint::Distance { parameter, .. } = &mut constraint.relation {
            *parameter = 1;
        }
    }
    let geometry = Sketch2D::from_recipe(&recipe).unwrap();
    snapshot
        .sketches
        .replace_geometry(SKETCH, geometry)
        .unwrap();
    snapshot.parameters.set_expression(1, "10 mm").unwrap();
    solve_selected_sketches(&mut snapshot);
    let definition = SketchPrismDefinition::new(
        SKETCH,
        selected_sketch_boundary(),
        [
            DatumLength::Constant(0.),
            DatumLength::Constant(0.),
            DatumLength::Parameter(1),
        ],
    )
    .unwrap();
    let reads = definition.reads(&snapshot).unwrap();
    snapshot
        .work_geometry
        .replace(
            FeatureId(1),
            PartFeatureDefinition::SketchPrism(Box::new(definition)),
            reads,
        )
        .unwrap();
    snapshot
}
fn assert_cube(history: &DocumentEditHistory, size_cm: f64, keys: &[Vec<u8>]) {
    let cube = certified(history);
    assert_eq!(cube.reference_keys(history, document()).unwrap(), keys);
    let body_key = key("body", &[1]);
    let report = cube.body_report(history, document(), &body_key).unwrap();
    assert_analytic_scalar(report.properties.volume, size_cm.powi(3));
    assert_eq!(
        (
            report.topology.faces,
            report.topology.edges,
            report.topology.vertices
        ),
        (6, 12, 8)
    );
    for face in keys.iter().filter(|candidate| **candidate != body_key) {
        assert_analytic_scalar(
            cube.face_report(history, document(), face).unwrap().area,
            size_cm.powi(2),
        );
    }
}
fn subentity_signatures(
    history: &DocumentEditHistory,
) -> Vec<(
    oblikovati_martensite::PrismSubentityRole,
    oblikovati_martensite::PrismHistorySource,
)> {
    let body = certified(history);
    let history = &body.certification().unwrap().subentities;
    assert_eq!((history.edges.len(), history.vertices.len()), (12, 8));
    history
        .edges
        .iter()
        .chain(&history.vertices)
        .map(|record| (record.role, record.source))
        .collect()
}
fn subentity_keys(history: &DocumentEditHistory) -> Vec<Vec<u8>> {
    let body = certified(history);
    let keys = body.subentity_reference_keys(history, document()).unwrap();
    assert_eq!(keys.len(), 20);
    for key in &keys {
        let provenance = body.subentity_provenance(history, document(), key).unwrap();
        assert!(provenance.point.iter().all(|value| value.is_finite()));
    }
    keys
}

#[path = "fillet_replay.rs"]
mod fillet_replay;

#[test]
fn one_size_parameter_drives_certified_cube_replay_and_saved_names() {
    let mut history = owner(cube_document());
    let keys = certified(&history)
        .reference_keys(&history, document())
        .unwrap();
    assert_cube(&history, 1., &keys);
    let subentities = subentity_signatures(&history);
    let lower_keys = subentity_keys(&history);
    let retained = certified(&history);
    let token = history.begin("Size").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(1, "20 mm")?;
            solve_selected_sketches(snapshot);
            Ok::<(), oblikovati_model::parameter::ParameterError<u64>>(())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert_cube(&history, 2., &keys);
    assert_eq!(subentity_signatures(&history), subentities);
    assert_eq!(subentity_keys(&history), lower_keys);
    assert!(
        retained
            .subentity_provenance(&history, document(), &lower_keys[0])
            .is_err()
    );
    history.undo().unwrap();
    assert_cube(&history, 1., &keys);
    assert_eq!(subentity_signatures(&history), subentities);
    assert_eq!(subentity_keys(&history), lower_keys);
    history.redo().unwrap();
    assert_cube(&history, 2., &keys);
    assert_eq!(subentity_signatures(&history), subentities);
    assert_eq!(subentity_keys(&history), lower_keys);
    let reopened = restored_owner(history.snapshot());
    assert_cube(&reopened, 2., &keys);
    assert_eq!(subentity_signatures(&reopened), subentities);
    assert_eq!(subentity_keys(&reopened), lower_keys);
}

#[test]
fn certified_replay_declines_an_unknown_resolution_version() {
    let history = owner(cube_document());
    let mut resolution = certified_resolution();
    resolution.version = 2;
    let error = oblikovati_martensite::retain_certified_named_sketch_prism(
        &history,
        document(),
        FeatureId(1),
        resolution,
    )
    .err()
    .unwrap();
    assert!(error.contains("version 2"), "{error}");
}
#[test]
fn certified_replay_keeps_named_faces_through_undo_redo_and_reopen() {
    let mut history = owner(selected_prism_document());
    let first = certified(&history);
    let keys = first.reference_keys(&history, document()).unwrap();
    assert_faces(&first, &history, 2.);
    assert!(first.certification().unwrap().achieved_tolerance.vertex <= 1e-5);
    let token = history.begin("Depth").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(3, "3 cm")
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let changed = certified(&history);
    assert_eq!(changed.reference_keys(&history, document()).unwrap(), keys);
    assert_faces(&changed, &history, 3.);
    history.undo().unwrap();
    assert_faces(&certified(&history), &history, 2.);
    history.redo().unwrap();
    assert_faces(&certified(&history), &history, 3.);
    let reopened = restored_owner(history.snapshot());
    let restored = certified(&reopened);
    assert_eq!(
        restored.reference_keys(&reopened, document()).unwrap(),
        keys
    );
    assert_faces(&restored, &reopened, 3.);
}

#[path = "rebuild_projection.rs"]
mod projection;

#[test]
fn named_prism_display_mesh_covers_each_analytic_face_once() {
    check_named_prism_mesh(owner(cube_document()), 6);
    check_named_prism_mesh(owner(holed_document(0)), 10);
}

fn check_named_prism_mesh(history: DocumentEditHistory, expected_faces: usize) {
    let mut evaluator = oblikovati_martensite::CertifiedNamedSolidEvaluator::new(
        &history,
        document(),
        certified_resolution(),
    );
    let replay = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    let solid = replay.body(FeatureId(1)).unwrap();
    let options = oblikovati_martensite::PrismMeshOptions {
        chord_error: 0.01,
        angular_error: 0.2,
    };
    let meshes = solid
        .named_face_meshes(&history, document(), options)
        .unwrap();
    let keys = solid.face_reference_keys(&history, document()).unwrap();
    assert_eq!(meshes.len(), expected_faces);
    assert_eq!(
        meshes
            .iter()
            .map(|mesh| mesh.key.clone())
            .collect::<Vec<_>>(),
        keys
    );
    use oblikovati_martensite::BodyPresentationSource;
    let ordinary = canonical_triangles(solid.body_triangles(options).unwrap());
    let named = canonical_triangles(
        meshes
            .iter()
            .flat_map(|mesh| mesh.triangles.iter().copied()),
    );
    assert_eq!(
        named, ordinary,
        "named mesh must preserve the complete original display"
    );
    for mesh in meshes {
        let area: f64 = mesh.triangles.iter().map(triangle_area).sum();
        let exact = solid.face_area(&history, document(), &mesh.key).unwrap();
        assert!(
            (area - exact).abs() < 1e-10,
            "mesh face {:?}: area {area}, analytic {exact}",
            mesh.key
        );
    }
}

fn triangle_area([a, b, c]: &[[f64; 3]; 3]) -> f64 {
    let u = std::array::from_fn::<_, 3, _>(|i| b[i] - a[i]);
    let v = std::array::from_fn::<_, 3, _>(|i| c[i] - a[i]);
    let cross = [
        u[1] * v[2] - u[2] * v[1],
        u[2] * v[0] - u[0] * v[2],
        u[0] * v[1] - u[1] * v[0],
    ];
    cross[0].hypot(cross[1]).hypot(cross[2]) / 2.
}

fn canonical_triangles(triangles: impl IntoIterator<Item = [[f64; 3]; 3]>) -> Vec<[[u64; 3]; 3]> {
    let mut result: Vec<_> = triangles
        .into_iter()
        .map(|triangle| triangle.map(|point| point.map(f64::to_bits)))
        .collect();
    result.sort();
    result
}
