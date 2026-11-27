// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::prism_test_fixture as fixture;
use oblikovati_martensite::{PrismResolution, retain_certified_named_sketch_prism};
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId,
        fillet::FilletDefinition,
        part::PartFeatureDefinition,
        work_geometry::DatumLength,
        work_plane::{WorkPlaneFaceReference, WorkPlaneReference, WorkPlaneResolver},
    },
    identity::{RecipeIdAllocator, ReferenceEnvelope},
};

#[test]
fn certified_continuity_resolves_a_datum_after_a_fillet() {
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let mut snapshot = fixture::selected_prism_document();
    fixture::solve_selected_sketches(&mut snapshot);
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    let mut history = DocumentEditHistory::new(snapshot, ids).unwrap();
    let resolution = PrismResolution {
        version: 1,
        weld: 1e-6,
        sew: 1e-7,
        angular: 1e-9,
        display_chordal: 1e-3,
        display_angular: 0.1,
        maximum_tolerance: 1e-5,
    };
    let prism =
        retain_certified_named_sketch_prism(&history, document, FeatureId(1), resolution).unwrap();
    let body = prism
        .body_reference_key(&history, document)
        .unwrap()
        .to_vec();
    let edge = prism
        .subentity_reference_keys(&history, document)
        .unwrap()
        .into_iter()
        .find(|key| ReferenceEnvelope::decode(key).unwrap().kind == 2)
        .unwrap();
    let face = prism
        .face_key_for_role(
            &history,
            document,
            oblikovati_martensite::PrismFaceRole::End,
        )
        .unwrap();
    append_fillet_and_datum(&mut history, body, edge, face);
    let token = history.begin("Origin sketch").unwrap();
    let origin = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_document_sketch(
                &oblikovati_model::history::DocumentSketchCreation {
                    name: "Origin".into(),
                    placement: oblikovati_model::document_sketch::SketchPlacement::OriginPlane(
                        oblikovati_model::feature::work_plane::WorkPlaneOrigin::XY,
                    ),
                    orientation: None,
                    project_origin: false,
                },
                ids,
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let resolver = PartSketchContinuity::new(
        document,
        Some(PrismResolutionPolicy::Fixed(resolution)),
        Default::default(),
    );
    let datums = resolver.datums(&history).unwrap();
    assert!(datums.plane(FeatureId(3)).is_ok());
    let sketch = history.snapshot().sketches.sketches()[0].id();
    let before = resolver
        .replay_count
        .load(std::sync::atomic::Ordering::Relaxed);
    assert!(resolver.plane(&history, sketch).is_ok());
    assert!(resolver.plane(&history, origin).is_ok());
    let results = resolver.resolve(&history, &[sketch, origin, DocumentSketchId(u64::MAX)]);
    assert!(results[0].is_ok() && results[1].is_ok() && results[2].is_err());
    assert_eq!(
        resolver
            .replay_count
            .load(std::sync::atomic::Ordering::Relaxed),
        before
    );
    let token = history.begin("Sketch on face datum").unwrap();
    let dependent = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_document_sketch_with_datums(
                &oblikovati_model::history::DocumentSketchCreation {
                    name: "Datum sketch".into(),
                    placement: oblikovati_model::document_sketch::SketchPlacement::WorkPlane(
                        FeatureId(3),
                    ),
                    orientation: None,
                    project_origin: false,
                },
                ids,
                &datums,
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert!(
        history
            .snapshot()
            .sketch_plane_needs_work_geometry(dependent)
            .unwrap()
    );
    let results = resolver.resolve(&history, &[sketch, dependent, DocumentSketchId(u64::MAX)]);
    assert!(results[0].is_ok() && results[1].is_ok() && results[2].is_err());
    assert_eq!(
        resolver
            .replay_count
            .load(std::sync::atomic::Ordering::Relaxed),
        before + 1
    );

    let token = history.begin("Work axis").unwrap();
    let axis = history
        .edit(&token, |snapshot, ids| {
            let id = ids.allocate_feature().map_err(|error| error.to_string())?;
            let line =
                oblikovati_model::feature::work_axis::WorkAxisLine::new([0., 0., 0.], [1., 0., 0.])
                    .unwrap();
            snapshot
                .work_geometry
                .append(AuthoredFeature::work_axis(id, "Axis", line))
                .map_err(|error| error.to_string())?;
            Ok::<_, String>(id)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let axis_datums = resolver.datums(&history).unwrap();
    let token = history.begin("Feature-oriented sketch").unwrap();
    let oriented = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_document_sketch_with_datums(
                &oblikovati_model::history::DocumentSketchCreation {
                    name: "Oriented".into(),
                    placement: oblikovati_model::document_sketch::SketchPlacement::Fixed(
                        oblikovati_model::sketch::SketchPlane::xy(),
                    ),
                    orientation: Some(
                        oblikovati_model::document_sketch::SketchOrientationDefinition {
                            reference:
                                oblikovati_model::document_sketch::SketchAxisReference::Feature(
                                    axis,
                                ),
                            sketch_axis: oblikovati_model::sketch::SketchFrameAxis::X,
                            reverse: false,
                            origin: None,
                        },
                    ),
                    project_origin: false,
                },
                ids,
                &axis_datums,
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert!(
        history
            .snapshot()
            .sketch_plane_needs_work_geometry(oriented)
            .unwrap()
    );
    let before_axis = resolver
        .replay_count
        .load(std::sync::atomic::Ordering::Relaxed);
    assert!(resolver.resolve(&history, &[oriented])[0].is_ok());
    assert_eq!(
        resolver
            .replay_count
            .load(std::sync::atomic::Ordering::Relaxed),
        before_axis + 1
    );
}

fn append_fillet_and_datum(
    history: &mut DocumentEditHistory,
    body: Vec<u8>,
    edge: Vec<u8>,
    face: Vec<u8>,
) {
    let definition = FilletDefinition::new(body, vec![edge], DatumLength::Constant(0.1)).unwrap();
    let reads = definition.reads();
    let datum = PartFeatureDefinition::Offset {
        base: WorkPlaneReference::Face(WorkPlaneFaceReference::new(face).unwrap()),
        distance: DatumLength::Constant(0.),
    };
    let token = history.begin("Fillet and datum").unwrap();
    history
        .edit(&token, |snapshot, ids| {
            let fillet_id = ids.allocate_feature().map_err(|cause| cause.to_string())?;
            snapshot
                .work_geometry
                .append(AuthoredFeature::new(
                    fillet_id,
                    "Fillet",
                    PartFeatureDefinition::Fillet(Box::new(definition)),
                    reads,
                ))
                .map_err(|cause| cause.to_string())?;
            let datum_id = ids.allocate_feature().map_err(|cause| cause.to_string())?;
            snapshot
                .work_geometry
                .append(AuthoredFeature::datum(datum_id, "Plane", datum))
                .map_err(|cause| cause.to_string())?;
            Ok::<(), String>(())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
}
