// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{WORK_PLANE_OFFSET, WORK_REF_XY_PLANE};
use oblikovati_api::wire::{ResolveSketchReferenceArgs, SketchEntityInfo};
use oblikovati_model::{feature::FeatureId, history::RecipeReservations, sketch::SketchPlane};
#[path = "entities/spline.rs"]
mod spline;
pub(super) fn rows(files: &DiscoveryWorkspace) -> Vec<SketchEntityInfo> {
    Client::new(FileQueryHost::new(files))
        .sketch()
        .entities(&SketchArgs::default())
        .unwrap()
        .entities
        .unwrap()
}
pub(super) fn install_mixed(files: &mut DiscoveryWorkspace, document: u64) {
    files
        .commands(document)
        .unwrap()
        .run("Mixed primitives", author_mixed)
        .unwrap();
}
fn author_mixed(
    snapshot: &mut DocumentSnapshot,
    reservations: &mut RecipeReservations<'_>,
) -> Result<(), String> {
    let sketch = reservations
        .allocate_sketch()
        .map_err(|cause| cause.to_string())?;
    let ids: Vec<SketchEntityId> = (0..6)
        .map(|_| reservations.allocate_sketch_entity().unwrap())
        .collect();
    let geometry = mixed_geometry(ids.try_into().unwrap());
    snapshot
        .sketches
        .add(sketch, "Mixed", SketchPlane::xy(), geometry)
        .map_err(|cause| cause.to_string())
}
fn mixed_geometry(ids: [SketchEntityId; 6]) -> Sketch2D {
    let mut geometry = Sketch2D::default();
    geometry.add_point(ids[0], [0.0; 2]).unwrap();
    geometry.add_point(ids[1], [20.0, 0.0]).unwrap();
    geometry.add_line(ids[2], [ids[0], ids[1]]).unwrap();
    geometry.add_circle(ids[3], ids[0], 20.0).unwrap();
    geometry.add_point(ids[4], [0.0, 20.0]).unwrap();
    geometry
        .add_arc(ids[5], ids[0], [ids[1], ids[4]], true)
        .unwrap();
    geometry
}
pub(super) fn configured_scaled(scale: f64) -> (DiscoveryWorkspace, u64, QueryGuids) {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    let document = files
        .create(
            Path::new("/fixtures/sketch.obk"),
            scaled_package(scale),
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    (files, document, source)
}
fn scaled_package(scale: f64) -> DocumentPackage {
    let contents = format!(
        "schemaVersion: 2\ndocumentType: 1\nmodel:\n  units: {{workingScale: '{scale}'}}\n  parameters: [{{name: width, kind: user, expression: '1 cm'}}]\n"
    );
    DocumentPackage::decode(contents.as_bytes()).unwrap()
}
pub(super) fn verify_keys(files: &DiscoveryWorkspace, entities: &[SketchEntityInfo]) {
    let client = Client::new(FileQueryHost::new(files));
    for entity in entities {
        let found = client
            .sketch()
            .resolve_reference(&ResolveSketchReferenceArgs {
                reference_key: entity.reference_key.clone(),
            })
            .unwrap();
        assert!(found.found);
        assert_eq!(
            (found.kind.as_str(), found.sketch_index, found.entity_id),
            ("sketchEntity", 0, entity.id)
        );
    }
}
#[test]
fn typed_entities_expose_all_owned_shapes_in_centimetres_and_authored_order() {
    let (mut files, document, _) = configured_scaled(0.1);
    install_mixed(&mut files, document);
    let entities = rows(&files);
    assert_eq!(
        entities
            .iter()
            .map(|row| row.kind.as_str())
            .collect::<Vec<_>>(),
        ["point", "point", "line", "circle", "point", "arc"]
    );
    assert_eq!(
        entities.iter().map(|row| row.index).collect::<Vec<_>>(),
        [0, 1, 2, 3, 4, 5]
    );
    assert_wire_shapes(&entities);
    verify_keys(&files, &entities);
}
fn assert_wire_shapes(entities: &[SketchEntityInfo]) {
    assert_eq!(
        entities[2].points,
        Some(vec![Some(vec![0.0; 2]), Some(vec![2.0, 0.0])])
    );
    assert_eq!((entities[3].radius, entities[5].radius), (2.0, 2.0));
    assert_eq!(
        entities[5].points,
        Some(vec![
            Some(vec![0.0; 2]),
            Some(vec![2.0, 0.0]),
            Some(vec![0.0, 2.0])
        ])
    );
    assert!(
        entities
            .iter()
            .all(|row| row.moveable_status == "freeToMove" && !row.construction && !row.reference)
    );
}
#[test]
fn queries_use_the_current_owner_without_geometry_history_or_dirty_mutation() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    files.save(document, "test", &mut source).unwrap();
    let before = owner_revision(&files, document);
    let saved = owned_geometry(&files, document).clone();
    let entities = rows(&files);
    assert!(entities.iter().all(|row| row.moveable_status == "fixed"));
    let reply = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .entities(&SketchArgs::default())
        .unwrap();
    assert_eq!(reply.entities.unwrap(), entities);
    assert_eq!(before, owner_revision(&files, document));
    assert!(saved.same_evaluated_state(owned_geometry(&files, document)));
}
#[test]
fn save_reopen_and_copy_retain_shapes_and_local_ids_with_the_right_namespace() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    let original = rows(&files);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(rows(&files), original);
    let path = Path::new("/fixtures/entity-copy.obk");
    activate_copy(&mut files, document, &mut source, path);
    let copied = rows(&files);
    for (first, second) in original.iter().zip(&copied) {
        assert_eq!(
            (first.id, &first.points, first.radius),
            (second.id, &second.points, second.radius)
        );
        assert_ne!(first.reference_key, second.reference_key);
    }
    verify_keys(&files, &copied);
}
pub(super) fn activate_copy(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    path: &Path,
) {
    files
        .save_copy(document, path, None, "test", source)
        .unwrap();
    let _opened = files
        .open(
            path,
            Default::default(),
            "test",
            source,
            OpenActivation::Activate,
        )
        .unwrap();
}
#[test]
fn projected_origin_and_empty_assembly_sketches_report_real_results() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        let (mut files, document, mut source) = configured_files(kind, Default::default());
        Client::new(FileHost::new(&mut files, &mut source, "test"))
            .sketch()
            .create(&Default::default())
            .unwrap();
        let before = owner_revision(&files, document);
        let entities = rows(&files);
        if kind == DOCUMENT_PART {
            assert_eq!(entities.len(), 1);
            assert_eq!(entities[0].moveable_status, "fixed");
            assert_eq!(entities[0].points, Some(vec![Some(vec![0.0; 2])]));
        } else {
            assert!(entities.is_empty());
        }
        assert_eq!(before, owner_revision(&files, document));
    }
}
#[test]
fn injected_invalid_rank_policy_declines_instead_of_publishing_unknown_status() {
    let options = SolveOptions {
        rank_tolerance: 0.0,
        ..Default::default()
    };
    let (files, document, _, _) = files_with_circle(options);
    let before = owner_revision(&files, document);
    expect_api_decline(
        Client::new(FileQueryHost::new(&files))
            .sketch()
            .entities(&SketchArgs::default()),
        "rank tolerance 0.0",
    );
    assert_eq!(before, owner_revision(&files, document));
}
#[test]
fn malformed_requests_and_wrong_or_absent_owners_decline() {
    let empty = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    expect_api_decline(
        Client::new(FileQueryHost::new(&empty))
            .sketch()
            .entities(&SketchArgs::default()),
        "active part",
    );
    for kind in [DOCUMENT_DRAWING, DOCUMENT_PRESENTATION] {
        let (files, _, _) = configured_files(kind, Default::default());
        expect_api_decline(
            Client::new(FileQueryHost::new(&files))
                .sketch()
                .entities(&SketchArgs::default()),
            "kind",
        );
    }
}
#[test]
fn invalid_selected_indices_and_malformed_json_decline() {
    let (files, _, _, _) = files_with_circle(Default::default());
    expect_api_decline(
        FileQueryHost::new(&files).call(METHOD_SKETCH_ENTITIES, b"{"),
        METHOD_SKETCH_ENTITIES,
    );
    for index in [-1, 1, i64::MAX] {
        expect_api_decline(
            Client::new(FileQueryHost::new(&files))
                .sketch()
                .entities(&SketchArgs {
                    sketch_index: index,
                }),
            &index.to_string(),
        );
    }
}

#[test]
fn dimension_controlled_point_status_is_not_inferred_from_a_pivot_or_fix_on_its_neighbour() {
    let (mut files, document, _) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    install_endpoint_constraints(&mut files, document);
    let entities = rows(&files);
    assert_eq!(entities[0].moveable_status, "fixed");
    assert_eq!(entities[1].moveable_status, "byDimensionChange");
    assert_eq!(entities[2].moveable_status, "fixed");
    assert_eq!(entities[3].moveable_status, "freeToMove");
}
fn install_endpoint_constraints(files: &mut DiscoveryWorkspace, document: u64) {
    files
        .commands(document)
        .unwrap()
        .run("Constrain endpoint", |snapshot, ids| {
            let sketch = snapshot.sketches.sketches()[0].id();
            let mut geometry = snapshot.sketches.get(sketch).unwrap().geometry().clone();
            constrain_endpoint(&mut geometry, sketch, ids);
            snapshot
                .sketches
                .replace_geometry(sketch, geometry)
                .map_err(|cause| cause.to_string())
        })
        .unwrap();
}
fn constrain_endpoint(
    geometry: &mut Sketch2D,
    sketch: DocumentSketchId,
    ids: &mut RecipeReservations<'_>,
) {
    let points = geometry.lines()[0].endpoints;
    for relation in endpoint_relations(points) {
        geometry
            .add_constraint(ids.allocate_sketch_constraint(sketch).unwrap(), relation)
            .unwrap();
    }
}
fn endpoint_relations(points: [SketchEntityId; 2]) -> [PlanarConstraint; 3] {
    use oblikovati_model::sketch::DistanceMeasure;
    [
        PlanarConstraint::Fix {
            point: points[0],
            position: [0.0; 2],
        },
        PlanarConstraint::Horizontal { points },
        PlanarConstraint::Distance {
            points,
            parameter: 1,
            measure: DistanceMeasure::Horizontal,
        },
    ]
}
#[test]
fn unavailable_projection_hosts_decline_without_returning_last_good_coordinates() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create_hosted_sketch(&mut files, &mut source);
    assert_eq!(rows(&files)[0].moveable_status, "fixed");
    files
        .commands(document)
        .unwrap()
        .run("Suppress host", |snapshot, _| {
            snapshot.work_geometry.set_suppressed(FeatureId(1), true)
        })
        .unwrap();
    let before = owner_revision(&files, document);
    expect_api_decline(
        Client::new(FileQueryHost::new(&files))
            .sketch()
            .entities(&SketchArgs::default()),
        "datum FeatureId(1): expected an available current plane",
    );
    assert_eq!(before, owner_revision(&files, document));
}
fn create_hosted_sketch(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    let client = Client::new(FileHost::new(files, source, "test"));
    client
        .work_planes()
        .create(&CreateWorkPlaneArgs {
            kind: WORK_PLANE_OFFSET.0.into_owned(),
            offset: "width".into(),
            refs: Some(vec![WORK_REF_XY_PLANE.into()]),
            ..Default::default()
        })
        .unwrap();
    client
        .sketch()
        .create(&CreateSketchArgs {
            work_plane_index: Some(std::boxed::Box::new(3)),
            ..Default::default()
        })
        .unwrap();
}
#[test]
fn centimetre_overflow_declines_without_mutating_the_owned_sketch() {
    let (mut files, document, _) = configured_scaled(2.0);
    files
        .commands(document)
        .unwrap()
        .run("Large point", author_large_point)
        .unwrap();
    let before = owner_revision(&files, document);
    expect_api_decline(
        Client::new(FileQueryHost::new(&files))
            .sketch()
            .entities(&SketchArgs::default()),
        "finite centimetres",
    );
    assert_eq!(before, owner_revision(&files, document));
}
pub(super) fn author_large_point(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<(), String> {
    let sketch = ids.allocate_sketch().map_err(|cause| cause.to_string())?;
    let point = ids
        .allocate_sketch_entity()
        .map_err(|cause| cause.to_string())?;
    let mut geometry = Sketch2D::default();
    geometry.add_point(point, [f64::MAX, 0.0]).unwrap();
    snapshot
        .sketches
        .add(sketch, "Large", SketchPlane::xy(), geometry)
        .map_err(|cause| cause.to_string())
}

#[test]
fn unhealthy_dimension_inputs_refuse_classification_without_last_good_targets() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&numeric("width", "missing"))
        .unwrap();
    let before = owner_revision(&files, document);
    let cause = Client::new(FileQueryHost::new(&files))
        .sketch()
        .entities(&SketchArgs::default())
        .unwrap_err()
        .0;
    assert!(cause.contains("parameter 1"), "{cause}");
    assert!(cause.contains("healthy"), "{cause}");
    assert_eq!(before, owner_revision(&files, document));
}
