// SPDX-License-Identifier: GPL-2.0-only
//! Unit tests for extended Work Plane definitions (Program v68, Source v58, #3918).

use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, FeatureSequence,
        work_axis::WorkAxisLine,
        work_geometry::{DatumAngle, DatumLength, WorkGeometryDefinition},
        work_plane::{
            WorkPlaneEdgeReference, WorkPlaneExtendedDefinition, WorkPlaneOrigin,
            WorkPlaneReference, WorkPlaneResolver, WorkPlaneVertexReference,
        },
    },
    parameter::ParameterCollection,
    units::UnitsOfMeasure,
};

#[test]
fn midplane_between_two_parallel_planes_evaluates_and_round_trips_v68_v58() {
    let params = ParameterCollection::new(UnitsOfMeasure::default());
    let mut seq = FeatureSequence::default();
    let offset = WorkGeometryDefinition::Offset {
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        distance: DatumLength::Constant(4.0),
    };
    seq.append(AuthoredFeature::datum(FeatureId(1), "Work Plane1", offset))
        .unwrap();
    let mid = WorkGeometryDefinition::WorkPlaneExtended(Box::new(
        WorkPlaneExtendedDefinition::MidplaneTwoPlanes {
            first: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
            second: WorkPlaneReference::Feature(FeatureId(1)),
        },
    ));
    seq.append(AuthoredFeature::datum(FeatureId(2), "Work Plane2", mid))
        .unwrap();

    let recipe = seq.work_plane_recipe().unwrap();
    assert_eq!(recipe.version, 68);
    assert_eq!(recipe.features[1].definition.version, 58);

    let json = serde_json::to_string(&recipe).unwrap();
    let decoded = serde_json::from_str(&json).unwrap();
    let restored = FeatureSequence::from_work_plane_recipe(&decoded).unwrap();
    assert!(seq.same_work_plane_recipe(&restored));

    let evaluated = restored.evaluate_work_planes(&params).unwrap();
    let plane = evaluated.plane(FeatureId(2)).unwrap();
    assert!((plane.recipe().origin[2] - 2.0).abs() < 1e-12);
    assert!((plane.normal()[2].abs() - 1.0).abs() < 1e-12);
}

#[test]
fn midplane_rejects_coincident_planes_and_supports_angle_bisector() {
    let params = ParameterCollection::new(UnitsOfMeasure::default());
    let mut coincident = FeatureSequence::default();
    let bad = WorkGeometryDefinition::WorkPlaneExtended(Box::new(
        WorkPlaneExtendedDefinition::MidplaneTwoPlanes {
            first: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
            second: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        },
    ));
    coincident
        .append(AuthoredFeature::datum(FeatureId(1), "Bad", bad))
        .unwrap();
    let eval = coincident.evaluate_work_planes(&params).unwrap();
    assert!(eval.plane(FeatureId(1)).is_err());

    let mut angled = FeatureSequence::default();
    let bisector = WorkGeometryDefinition::WorkPlaneExtended(Box::new(
        WorkPlaneExtendedDefinition::MidplaneTwoPlanes {
            first: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
            second: WorkPlaneReference::Origin(WorkPlaneOrigin::XZ),
        },
    ));
    angled
        .append(AuthoredFeature::datum(FeatureId(1), "Bisector", bisector))
        .unwrap();
    let eval = angled.evaluate_work_planes(&params).unwrap();
    let plane = eval.plane(FeatureId(1)).unwrap();
    let n = plane.normal();
    assert!((n[0]).abs() < 1e-12);
    assert!((n[1].abs() - std::f64::consts::FRAC_1_SQRT_2).abs() < 1e-12);
    assert!((n[2].abs() - std::f64::consts::FRAC_1_SQRT_2).abs() < 1e-12);
}

#[test]
fn extended_three_points_parallel_normal_angle_and_coplanar_edges_evaluate() {
    let params = ParameterCollection::new(UnitsOfMeasure::default());
    let mut seq = FeatureSequence::default();
    seq.append(AuthoredFeature::datum(
        FeatureId(1),
        "P1",
        WorkGeometryDefinition::Point([
            DatumLength::Constant(2.0),
            DatumLength::Constant(0.0),
            DatumLength::Constant(3.0),
        ]),
    ))
    .unwrap();
    seq.append(AuthoredFeature::datum(
        FeatureId(2),
        "P2",
        WorkGeometryDefinition::Point([
            DatumLength::Constant(0.0),
            DatumLength::Constant(4.0),
            DatumLength::Constant(3.0),
        ]),
    ))
    .unwrap();
    seq.append(AuthoredFeature::datum(
        FeatureId(3),
        "P3",
        WorkGeometryDefinition::Point([
            DatumLength::Constant(0.0),
            DatumLength::Constant(0.0),
            DatumLength::Constant(3.0),
        ]),
    ))
    .unwrap();
    seq.append(AuthoredFeature::datum(
        FeatureId(4),
        "ThreePts",
        WorkGeometryDefinition::WorkPlaneExtended(Box::new(
            WorkPlaneExtendedDefinition::ThreePoints {
                points: [
                    WorkPlaneVertexReference::WorkPoint(FeatureId(3)),
                    WorkPlaneVertexReference::WorkPoint(FeatureId(1)),
                    WorkPlaneVertexReference::WorkPoint(FeatureId(2)),
                ],
            },
        )),
    ))
    .unwrap();
    seq.append(AuthoredFeature::datum(
        FeatureId(5),
        "Parallel",
        WorkGeometryDefinition::WorkPlaneExtended(Box::new(
            WorkPlaneExtendedDefinition::ParallelThroughPoint {
                base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
                point: WorkPlaneVertexReference::WorkPoint(FeatureId(1)),
            },
        )),
    ))
    .unwrap();
    seq.append(AuthoredFeature::datum(
        FeatureId(6),
        "NormalAxis",
        WorkGeometryDefinition::WorkPlaneExtended(Box::new(
            WorkPlaneExtendedDefinition::NormalToAxisThroughPoint {
                axis: WorkPlaneEdgeReference::OriginZ,
                point: WorkPlaneVertexReference::WorkPoint(FeatureId(1)),
            },
        )),
    ))
    .unwrap();
    seq.append(AuthoredFeature::datum(
        FeatureId(7),
        "Angled45",
        WorkGeometryDefinition::WorkPlaneExtended(Box::new(
            WorkPlaneExtendedDefinition::AngleToPlaneAroundEdge {
                base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
                edge: WorkPlaneEdgeReference::OriginX,
                angle: DatumAngle::constant(std::f64::consts::FRAC_PI_4).unwrap(),
            },
        )),
    ))
    .unwrap();
    seq.append(AuthoredFeature::datum(
        FeatureId(8),
        "AxisA",
        WorkGeometryDefinition::Axis(WorkAxisLine::new([0.0, 0.0, 5.0], [1.0, 0.0, 0.0]).unwrap()),
    ))
    .unwrap();
    seq.append(AuthoredFeature::datum(
        FeatureId(9),
        "AxisB",
        WorkGeometryDefinition::Axis(WorkAxisLine::new([0.0, 0.0, 5.0], [0.0, 1.0, 0.0]).unwrap()),
    ))
    .unwrap();
    seq.append(AuthoredFeature::datum(
        FeatureId(10),
        "Coplanar",
        WorkGeometryDefinition::WorkPlaneExtended(Box::new(
            WorkPlaneExtendedDefinition::TwoCoplanarEdges {
                first: WorkPlaneEdgeReference::WorkAxis(FeatureId(8)),
                second: WorkPlaneEdgeReference::WorkAxis(FeatureId(9)),
            },
        )),
    ))
    .unwrap();

    // Flip normal + toggle visibility on extended plane
    seq.flip_plane_normal(FeatureId(5)).unwrap();
    seq.set_datum_visible(FeatureId(5), false).unwrap();

    let evaluated = seq.evaluate_work_planes(&params).unwrap();
    assert!((evaluated.plane(FeatureId(4)).unwrap().recipe().origin[2] - 3.0).abs() < 1e-12);
    assert!((evaluated.plane(FeatureId(5)).unwrap().normal()[2] + 1.0).abs() < 1e-12);
    assert!(!seq.get(FeatureId(5)).unwrap().is_visible());
    assert!((evaluated.plane(FeatureId(6)).unwrap().normal()[2].abs() - 1.0).abs() < 1e-12);
    let angled = evaluated.plane(FeatureId(7)).unwrap();
    assert!((angled.normal()[1].abs() - std::f64::consts::FRAC_1_SQRT_2).abs() < 1e-12);
    assert!((angled.normal()[2].abs() - std::f64::consts::FRAC_1_SQRT_2).abs() < 1e-12);
    let coplanar = evaluated.plane(FeatureId(10)).unwrap();
    assert!((coplanar.recipe().origin[2] - 5.0).abs() < 1e-12);
    assert!((coplanar.normal()[2].abs() - 1.0).abs() < 1e-12);
}

#[test]
fn two_coplanar_edges_rejects_skew_lines() {
    let params = ParameterCollection::new(UnitsOfMeasure::default());
    let mut seq = FeatureSequence::default();
    seq.append(AuthoredFeature::datum(
        FeatureId(1),
        "A",
        WorkGeometryDefinition::Axis(WorkAxisLine::new([0.0, 0.0, 0.0], [1.0, 0.0, 0.0]).unwrap()),
    ))
    .unwrap();
    seq.append(AuthoredFeature::datum(
        FeatureId(2),
        "B",
        WorkGeometryDefinition::Axis(WorkAxisLine::new([0.0, 0.0, 2.0], [0.0, 1.0, 0.0]).unwrap()),
    ))
    .unwrap();
    seq.append(AuthoredFeature::datum(
        FeatureId(3),
        "Skew",
        WorkGeometryDefinition::WorkPlaneExtended(Box::new(
            WorkPlaneExtendedDefinition::TwoCoplanarEdges {
                first: WorkPlaneEdgeReference::WorkAxis(FeatureId(1)),
                second: WorkPlaneEdgeReference::WorkAxis(FeatureId(2)),
            },
        )),
    ))
    .unwrap();
    let eval = seq.evaluate_work_planes(&params).unwrap();
    assert!(eval.plane(FeatureId(3)).is_err());
}

fn semantic_key(context: u64, kind: u32, role: &str) -> Vec<u8> {
    oblikovati_model::identity::SemanticReferenceName::new(
        "11111111-2222-3333-4444-555555555555".parse().unwrap(),
        context,
        kind,
        "sketch-prism",
        role,
        &[1, 101],
    )
    .unwrap()
    .encode()
    .unwrap()
}

struct StubExtendedResolver {
    torus_plane: oblikovati_model::sketch::SketchPlane,
    tangent_plane: oblikovati_model::sketch::SketchPlane,
}

impl WorkPlaneResolver for StubExtendedResolver {
    fn plane(
        &self,
        _id: FeatureId,
    ) -> Result<
        oblikovati_model::sketch::SketchPlane,
        oblikovati_model::feature::work_plane::WorkPlaneError,
    > {
        Ok(oblikovati_model::sketch::SketchPlane::xy())
    }
    fn axis(
        &self,
        _id: FeatureId,
    ) -> Result<WorkAxisLine, oblikovati_model::feature::work_plane::WorkPlaneError> {
        Ok(WorkAxisLine::new([0.0, 0.0, 0.0], [0.0, 0.0, 1.0]).unwrap())
    }
    fn point(
        &self,
        _id: FeatureId,
    ) -> Result<[f64; 3], oblikovati_model::feature::work_plane::WorkPlaneError> {
        Ok([1.0, 2.0, 3.0])
    }
}

impl oblikovati_model::feature::work_plane::WorkPlaneExtendedResolver for StubExtendedResolver {
    fn topology_vertex(
        &self,
        _reference: &oblikovati_model::feature::work_plane::WorkPlaneTopologyRef,
    ) -> Result<[f64; 3], oblikovati_model::feature::work_plane::WorkPlaneError> {
        Ok([2.0, 0.0, 0.0])
    }
    fn topology_edge(
        &self,
        _reference: &oblikovati_model::feature::work_plane::WorkPlaneTopologyRef,
    ) -> Result<WorkAxisLine, oblikovati_model::feature::work_plane::WorkPlaneError> {
        Ok(WorkAxisLine::new([2.0, 0.0, 0.0], [0.0, 0.0, 1.0]).unwrap())
    }
    fn torus_midplane(
        &self,
        _surface: &oblikovati_model::feature::work_plane::WorkPlaneSurfaceReference,
    ) -> Result<
        oblikovati_model::sketch::SketchPlane,
        oblikovati_model::feature::work_plane::WorkPlaneError,
    > {
        Ok(self.torus_plane)
    }
    fn tangent_through_edge(
        &self,
        _surface: &oblikovati_model::feature::work_plane::WorkPlaneSurfaceReference,
        _line: WorkAxisLine,
    ) -> Result<
        oblikovati_model::sketch::SketchPlane,
        oblikovati_model::feature::work_plane::WorkPlaneError,
    > {
        Ok(self.tangent_plane)
    }
    fn tangent_through_point(
        &self,
        _surface: &oblikovati_model::feature::work_plane::WorkPlaneSurfaceReference,
        _point: [f64; 3],
    ) -> Result<
        oblikovati_model::sketch::SketchPlane,
        oblikovati_model::feature::work_plane::WorkPlaneError,
    > {
        Ok(self.tangent_plane)
    }
    fn tangent_parallel_to_plane(
        &self,
        _surface: &oblikovati_model::feature::work_plane::WorkPlaneSurfaceReference,
        _plane: oblikovati_model::sketch::SketchPlane,
    ) -> Result<
        oblikovati_model::sketch::SketchPlane,
        oblikovati_model::feature::work_plane::WorkPlaneError,
    > {
        Ok(self.tangent_plane)
    }
}

#[test]
fn torus_and_tangent_extended_constructors_evaluate_and_roundtrip_v68_v58() {
    use oblikovati_model::feature::work_plane::{
        WorkPlaneFaceReference, WorkPlaneSurfaceReference, WorkPlaneTopologyRef,
    };
    let params = ParameterCollection::new(UnitsOfMeasure::default());
    let face = WorkPlaneFaceReference::new(semantic_key(10, 1, "cyl")).unwrap();
    let edge = WorkPlaneTopologyRef::edge(semantic_key(10, 2, "edge")).unwrap();
    let vertex = WorkPlaneTopologyRef::vertex(semantic_key(10, 3, "vtx")).unwrap();
    let surface = WorkPlaneSurfaceReference { face };

    let defs = [
        WorkPlaneExtendedDefinition::MidplaneOfTorus {
            surface: surface.clone(),
        },
        WorkPlaneExtendedDefinition::TangentThroughEdge {
            surface: surface.clone(),
            edge: WorkPlaneEdgeReference::TopologyEdge(edge),
        },
        WorkPlaneExtendedDefinition::TangentThroughPoint {
            surface: surface.clone(),
            point: WorkPlaneVertexReference::TopologyVertex(vertex),
        },
        WorkPlaneExtendedDefinition::TangentParallelToPlane {
            surface,
            plane: WorkPlaneReference::Origin(WorkPlaneOrigin::YZ),
        },
    ];

    let mut seq = FeatureSequence::default();
    seq.append(AuthoredFeature::datum(
        FeatureId(10),
        "SourceDatum",
        WorkGeometryDefinition::Offset {
            base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
            distance: DatumLength::Constant(1.0),
        },
    ))
    .unwrap();
    for (idx, def) in defs.iter().enumerate() {
        assert!(def.requires_topology());
        let name = format!("WP{}", idx + 1);
        seq.append(AuthoredFeature::datum(
            FeatureId((idx + 11) as u64),
            &name,
            WorkGeometryDefinition::WorkPlaneExtended(Box::new(def.clone())),
        ))
        .unwrap();
    }

    let recipe = seq.work_plane_recipe().unwrap();
    assert_eq!(recipe.version, 68);
    assert!(
        recipe.features[1..]
            .iter()
            .all(|f| f.definition.version == 58)
    );
    let json = serde_json::to_string(&recipe).unwrap();
    let decoded = serde_json::from_str(&json).unwrap();
    let restored = FeatureSequence::from_work_plane_recipe(&decoded).unwrap();
    assert!(seq.same_work_plane_recipe(&restored));

    let resolver = StubExtendedResolver {
        torus_plane: oblikovati_model::sketch::SketchPlane::xy(),
        tangent_plane: oblikovati_model::sketch::SketchPlane::yz(),
    };
    let torus = defs[0].evaluate(&params, &resolver).unwrap();
    assert!((torus.normal()[2].abs() - 1.0).abs() < 1e-12);
    for def in &defs[1..] {
        let tan = def.evaluate(&params, &resolver).unwrap();
        assert!((tan.normal()[0].abs() - 1.0).abs() < 1e-12);
    }
}

#[test]
fn extended_work_plane_grounding_redefinition_and_downstream_reference() {
    use oblikovati_model::feature::work_geometry::WorkPlaneDisplaySettings;
    let params = ParameterCollection::new(UnitsOfMeasure::default());
    let mut seq = FeatureSequence::default();
    let initial = WorkGeometryDefinition::WorkPlaneExtended(Box::new(
        WorkPlaneExtendedDefinition::ParallelThroughPoint {
            base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
            point: WorkPlaneVertexReference::Origin,
        },
    ));
    seq.append(AuthoredFeature::datum(FeatureId(1), "Work Plane1", initial))
        .unwrap();
    let grounded = WorkPlaneDisplaySettings::default()
        .edited(None, Some(true), None)
        .unwrap();
    seq.set_plane_display(FeatureId(1), grounded).unwrap();
    assert!(
        seq.get(FeatureId(1))
            .unwrap()
            .plane_display_settings()
            .grounded()
    );

    // Downstream offset work plane referencing the extended work plane
    seq.append(AuthoredFeature::datum(
        FeatureId(2),
        "Work Plane2",
        WorkGeometryDefinition::Offset {
            base: WorkPlaneReference::Feature(FeatureId(1)),
            distance: DatumLength::Constant(2.5),
        },
    ))
    .unwrap();
    let eval = seq.evaluate_work_planes(&params).unwrap();
    assert!((eval.plane(FeatureId(2)).unwrap().recipe().origin[2] - 2.5).abs() < 1e-12);

    // Redefine Work Plane1 to an angled plane and verify downstream updates associatively
    let redefined = WorkGeometryDefinition::WorkPlaneExtended(Box::new(
        WorkPlaneExtendedDefinition::AngleToPlaneAroundEdge {
            base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
            edge: WorkPlaneEdgeReference::OriginX,
            angle: DatumAngle::constant(std::f64::consts::FRAC_PI_2).unwrap(),
        },
    ));
    let reads = redefined.reads();
    seq.replace(FeatureId(1), redefined, reads).unwrap();
    let updated = seq.evaluate_work_planes(&params).unwrap();
    let downstream = updated.plane(FeatureId(2)).unwrap();
    assert!((downstream.normal()[1].abs() - 1.0).abs() < 1e-12);
}
