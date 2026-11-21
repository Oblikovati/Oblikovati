// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_model::sketch::{SketchFrameAlignment, SketchFrameAxis, SketchPlane};

#[test]
fn origin_only_edit_preserves_existing_axes_and_local_geometry() {
    let initial = SketchPlane::xy()
        .with_orientation([1., 0., 0.], SketchFrameAxis::Y, false, None)
        .unwrap();
    let changed = initial.with_frame_edit(Some([2., 3., 0.]), None).unwrap();
    assert_eq!(
        changed.recipe().x_axis.map(f64::to_bits),
        initial.recipe().x_axis.map(f64::to_bits)
    );
    assert_eq!(
        changed.recipe().y_axis.map(f64::to_bits),
        initial.recipe().y_axis.map(f64::to_bits)
    );
    assert_eq!(changed.to_model([1., 2.]).unwrap(), [4., 2., 0.]);
}

#[test]
fn axis_only_edit_preserves_origin_and_supports_reverse() {
    let initial = SketchPlane::xy()
        .with_frame_edit(Some([2., 3., 0.]), None)
        .unwrap();
    let aligned = initial
        .with_frame_edit(
            None,
            Some(SketchFrameAlignment {
                direction: [1., 0., 0.],
                axis: SketchFrameAxis::Y,
                reverse: false,
            }),
        )
        .unwrap();
    let reversed = initial
        .with_frame_edit(
            None,
            Some(SketchFrameAlignment {
                direction: [1., 0., 0.],
                axis: SketchFrameAxis::Y,
                reverse: true,
            }),
        )
        .unwrap();
    assert_eq!(aligned.recipe().origin, [2., 3., 0.]);
    assert_eq!(aligned.recipe().y_axis, [1., 0., 0.]);
    assert_eq!(reversed.recipe().y_axis, [-1., -0., -0.]);
    assert_eq!(aligned.to_model([1., 2.]).unwrap(), [4., 2., 0.]);
}

#[test]
fn frame_edit_refuses_invalid_origin_and_degenerate_axis() {
    assert!(
        SketchPlane::xy()
            .with_frame_edit(Some([f64::NAN, 0., 0.]), None)
            .is_err()
    );
    assert!(
        SketchPlane::xy()
            .with_frame_edit(
                None,
                Some(SketchFrameAlignment {
                    direction: [0., 0., 1.],
                    axis: SketchFrameAxis::X,
                    reverse: false
                })
            )
            .is_err()
    );
}

#[test]
fn authored_origin_edit_keeps_base_axes_and_round_trips_with_local_geometry() {
    use oblikovati_model::document_sketch::{
        DocumentSketchId, DocumentSketches, SketchFrameEditDefinition, SketchFrameOrigin,
    };
    use oblikovati_model::sketch::{Sketch2D, SketchEntityId};
    let sketch = DocumentSketchId(7);
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(8), [1., 2.]).unwrap();
    let mut sketches = DocumentSketches::default();
    let base = SketchPlane::xy()
        .with_orientation([1., 0., 0.], SketchFrameAxis::Y, false, None)
        .unwrap();
    sketches
        .add(sketch, "Edited", base, geometry.clone())
        .unwrap();
    sketches
        .set_frame_edit(
            sketch,
            Some(SketchFrameEditDefinition {
                origin: Some(SketchFrameOrigin::Position([2., 3., 0.])),
                alignment: None,
            }),
        )
        .unwrap();
    let saved = sketches.recipe();
    assert_eq!(saved.version, 15);
    let mut old_version = saved.clone();
    old_version.version = 14;
    assert!(DocumentSketches::from_recipe(&old_version).is_err());
    let restored = DocumentSketches::from_recipe(&saved).unwrap();
    assert!(sketches.same_recipe(&restored));
    let placed = restored.get(sketch).unwrap();
    assert!(placed.frame_edit().is_some());
    assert!(placed.geometry().same_recipe(&geometry));
    assert_eq!(
        placed.plane(None).unwrap().to_model([1., 2.]).unwrap(),
        [4., 2., 0.]
    );
}

struct FrameDatumResolver {
    point: Option<[f64; 3]>,
}
impl oblikovati_model::document_sketch::SketchAxisResolver for FrameDatumResolver {
    fn axis_direction(
        &self,
        feature: oblikovati_model::feature::FeatureId,
    ) -> Result<[f64; 3], oblikovati_model::sketch::SketchError> {
        Err(oblikovati_model::sketch::SketchError::Definition(format!(
            "axis {feature:?}: expected no axis lookup"
        )))
    }
    fn frame_origin(
        &self,
        reference: oblikovati_model::feature::work_geometry::WorkPointReference,
    ) -> Result<[f64; 3], oblikovati_model::sketch::SketchError> {
        self.point.ok_or_else(|| {
            oblikovati_model::sketch::SketchError::Definition(format!(
                "datum {reference:?}: expected a current point"
            ))
        })
    }
}

#[test]
fn datum_origin_is_resolved_on_each_read_and_declines_when_unavailable() {
    use oblikovati_model::{
        document_sketch::{
            DocumentSketchId, DocumentSketches, SketchFrameEditDefinition, SketchFrameOrigin,
        },
        feature::{FeatureId, FeatureReadKey, work_geometry::WorkPointReference},
        sketch::Sketch2D,
    };
    let id = DocumentSketchId(1);
    let mut sketches = DocumentSketches::default();
    sketches
        .add(id, "Referenced", SketchPlane::xy(), Sketch2D::default())
        .unwrap();
    let origin = SketchFrameOrigin::Datum(WorkPointReference::Feature(FeatureId(9)));
    sketches
        .set_frame_edit(
            id,
            Some(SketchFrameEditDefinition {
                origin: Some(origin),
                alignment: None,
            }),
        )
        .unwrap();
    assert!(sketches.requires_part_dependency_recompute());
    let placed = sketches.get(id).unwrap();
    assert!(
        placed
            .read_footprint()
            .keys()
            .contains(&FeatureReadKey::Feature(FeatureId(9)))
    );
    for position in [[1., 2., 0.], [3., 4., 0.]] {
        let frame = placed
            .plane_with_axes(
                None,
                &FrameDatumResolver {
                    point: Some(position),
                },
            )
            .unwrap();
        assert_eq!(frame.recipe().origin, position);
    }
    assert!(
        placed
            .plane_with_axes(None, &FrameDatumResolver { point: None })
            .is_err()
    );
}

#[test]
fn invalid_frame_edits_leave_the_authored_recipe_unchanged() {
    use oblikovati_model::document_sketch::{
        DocumentSketchId, DocumentSketches, SketchAxisReference, SketchFrameAxisEdit,
        SketchFrameEditDefinition, SketchFrameOrigin,
    };
    use oblikovati_model::feature::{FeatureId, work_geometry::WorkPointReference};
    let id = DocumentSketchId(1);
    let mut sketches = DocumentSketches::default();
    sketches
        .add(id, "Invalid", SketchPlane::xy(), Default::default())
        .unwrap();
    let before = sketches.recipe();
    for edit in [
        SketchFrameEditDefinition {
            origin: None,
            alignment: None,
        },
        SketchFrameEditDefinition {
            origin: Some(SketchFrameOrigin::Position([f64::NAN, 0., 0.])),
            alignment: None,
        },
        SketchFrameEditDefinition {
            origin: Some(SketchFrameOrigin::Datum(WorkPointReference::Feature(
                FeatureId(0),
            ))),
            alignment: None,
        },
        SketchFrameEditDefinition {
            origin: None,
            alignment: Some(SketchFrameAxisEdit {
                reference: SketchAxisReference::Feature(FeatureId(0)),
                axis: SketchFrameAxis::X,
                reverse: false,
            }),
        },
    ] {
        assert!(sketches.set_frame_edit(id, Some(edit)).is_err());
        assert_eq!(sketches.recipe(), before);
    }
}

#[test]
fn authored_origin_preserves_signed_zero_in_recipe_comparisons() {
    use oblikovati_model::document_sketch::{
        DocumentSketchId, DocumentSketches, SketchFrameEditDefinition, SketchFrameOrigin,
    };
    let id = DocumentSketchId(1);
    let mut sketches = DocumentSketches::default();
    sketches
        .add(id, "Exact", SketchPlane::xy(), Default::default())
        .unwrap();
    let edit = |x| {
        Some(SketchFrameEditDefinition {
            origin: Some(SketchFrameOrigin::Position([x, 0., 0.])),
            alignment: None,
        })
    };
    sketches.set_frame_edit(id, edit(0.)).unwrap();
    let positive = sketches.clone();
    sketches.set_frame_edit(id, edit(-0.)).unwrap();
    assert!(!positive.same_recipe(&sketches));
}

#[test]
fn clearing_a_frame_edit_restores_the_base_recipe() {
    use oblikovati_model::document_sketch::{
        DocumentSketchId, DocumentSketches, SketchFrameEditDefinition, SketchFrameOrigin,
    };
    let id = DocumentSketchId(1);
    let mut sketches = DocumentSketches::default();
    sketches
        .add(id, "Reset", SketchPlane::xy(), Default::default())
        .unwrap();
    let before = sketches.clone();
    sketches
        .set_frame_edit(
            id,
            Some(SketchFrameEditDefinition {
                origin: Some(SketchFrameOrigin::Position([2., 3., 0.])),
                alignment: None,
            }),
        )
        .unwrap();
    sketches.set_frame_edit(id, None).unwrap();
    assert!(before.same_recipe(&sketches));
    assert!(sketches.get(id).unwrap().frame_edit().is_none());
}

#[test]
fn partial_frame_edits_preserve_previous_authored_aspects() {
    use oblikovati_model::document_sketch::{
        DocumentSketchId, DocumentSketches, SketchAxisReference, SketchFrameAxisEdit,
        SketchFrameEditDefinition, SketchFrameOrigin,
    };
    let id = DocumentSketchId(1);
    let mut sketches = DocumentSketches::default();
    sketches
        .add(id, "Partial", SketchPlane::xy(), Default::default())
        .unwrap();
    sketches
        .patch_frame_edit(
            id,
            SketchFrameEditDefinition {
                origin: Some(SketchFrameOrigin::Position([2., 3., 0.])),
                alignment: None,
            },
        )
        .unwrap();
    sketches
        .patch_frame_edit(
            id,
            SketchFrameEditDefinition {
                origin: None,
                alignment: Some(SketchFrameAxisEdit {
                    reference: SketchAxisReference::OriginX,
                    axis: SketchFrameAxis::Y,
                    reverse: false,
                }),
            },
        )
        .unwrap();
    let frame = sketches.get(id).unwrap().plane(None).unwrap();
    assert_eq!(frame.recipe().origin, [2., 3., 0.]);
    assert_eq!(frame.recipe().y_axis, [1., 0., 0.]);
    let before = sketches.recipe();
    assert!(
        sketches
            .patch_frame_edit(
                id,
                SketchFrameEditDefinition {
                    origin: None,
                    alignment: None
                }
            )
            .is_err()
    );
    assert_eq!(sketches.recipe(), before);
}

struct NamedFrameOrigin {
    key: oblikovati_model::document_sketch::SketchFrameTopologyOrigin,
    position: Option<[f64; 3]>,
}
impl oblikovati_model::document_sketch::SketchAxisResolver for NamedFrameOrigin {
    fn axis_direction(
        &self,
        feature: oblikovati_model::feature::FeatureId,
    ) -> Result<[f64; 3], oblikovati_model::sketch::SketchError> {
        Err(oblikovati_model::sketch::SketchError::Definition(format!(
            "axis {feature:?}: expected no lookup"
        )))
    }
    fn frame_topology_origin(
        &self,
        reference: &oblikovati_model::document_sketch::SketchFrameTopologyOrigin,
    ) -> Result<[f64; 3], oblikovati_model::sketch::SketchError> {
        if reference != &self.key {
            return Err(oblikovati_model::sketch::SketchError::Definition(
                "unexpected topology key".into(),
            ));
        }
        self.position.ok_or_else(|| {
            oblikovati_model::sketch::SketchError::Definition(
                "expected current named topology".into(),
            )
        })
    }
}

#[test]
fn topology_origin_uses_current_named_source_and_preserves_reference_through_recipe() {
    use oblikovati_model::{
        document_sketch::{
            DocumentSketchId, DocumentSketches, SketchFrameEditDefinition, SketchFrameOrigin,
            SketchFrameTopologyOrigin,
        },
        feature::{FeatureId, FeatureReadKey},
        identity::SemanticReferenceName,
    };
    let key = SemanticReferenceName::new(
        "11111111-2222-3333-4444-555555555555".parse().unwrap(),
        9,
        2,
        "sketch-prism",
        "edge",
        &[1, 101],
    )
    .unwrap()
    .encode()
    .unwrap();
    let reference = SketchFrameTopologyOrigin::new(key).unwrap();
    let id = DocumentSketchId(1);
    let mut sketches = DocumentSketches::default();
    sketches
        .add(id, "Named", SketchPlane::xy(), Default::default())
        .unwrap();
    sketches
        .set_frame_edit(
            id,
            Some(SketchFrameEditDefinition {
                origin: Some(SketchFrameOrigin::Topology(reference.clone())),
                alignment: None,
            }),
        )
        .unwrap();
    let restored = DocumentSketches::from_recipe(&sketches.recipe()).unwrap();
    let placed = restored.get(id).unwrap();
    assert!(
        placed
            .read_footprint()
            .keys()
            .contains(&FeatureReadKey::Feature(FeatureId(9)))
    );
    for position in [[1., 2., 0.], [3., 4., 0.]] {
        let resolver = NamedFrameOrigin {
            key: reference.clone(),
            position: Some(position),
        };
        assert_eq!(
            placed
                .plane_with_axes(None, &resolver)
                .unwrap()
                .recipe()
                .origin,
            position
        );
    }
    assert!(
        placed
            .plane_with_axes(
                None,
                &NamedFrameOrigin {
                    key: reference,
                    position: None
                }
            )
            .is_err()
    );
}

#[test]
fn snapshot_resolves_frame_datum_from_its_own_current_work_geometry() {
    use oblikovati_model::{
        document_sketch::{DocumentSketchId, SketchFrameEditDefinition, SketchFrameOrigin},
        feature::{
            AuthoredFeature, FeatureId,
            work_geometry::{DatumLength, WorkPointReference},
            work_plane::WorkPlaneDefinition,
        },
    };
    let id = DocumentSketchId(1);
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .sketches
        .add(id, "Datum", SketchPlane::xy(), Default::default())
        .unwrap();
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(
            FeatureId(9),
            "Point",
            WorkPlaneDefinition::Point([
                DatumLength::Constant(2.),
                DatumLength::Constant(3.),
                DatumLength::Constant(0.),
            ]),
        ))
        .unwrap();
    snapshot
        .sketches
        .set_frame_edit(
            id,
            Some(SketchFrameEditDefinition {
                origin: Some(SketchFrameOrigin::Datum(WorkPointReference::Feature(
                    FeatureId(9),
                ))),
                alignment: None,
            }),
        )
        .unwrap();
    assert!(snapshot.sketch_plane_needs_work_geometry(id).unwrap());
    assert_eq!(
        snapshot.resolve_sketch_plane(id).unwrap().recipe().origin,
        [2., 3., 0.]
    );
}
