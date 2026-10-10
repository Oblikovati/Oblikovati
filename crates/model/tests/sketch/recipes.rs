// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{
    AngleTurn, CircleContact, LineContactSide, LineRelation, SketchRecipe, TangentDistanceContact,
    TangentDistanceOptions, TangentLean,
};
fn rich_sketch() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, xy) in [
        (1, [0.0, 0.0]),
        (2, [4.0, 0.0]),
        (3, [0.0, 5.0]),
        (4, [0.0, 10.0]),
        (5, [2.0, 3.0]),
    ] {
        sketch.add_point(entity(id), xy).unwrap();
    }
    sketch.add_line(entity(10), [entity(1), entity(2)]).unwrap();
    sketch.add_line(entity(11), [entity(2), entity(5)]).unwrap();
    sketch.add_circle(entity(12), entity(5), 2.0).unwrap();
    sketch.add_circle(entity(13), entity(3), 5.0).unwrap();
    sketch
        .add_arc(entity(14), entity(3), [entity(1), entity(4)], false)
        .unwrap();
    sketch
}
fn constraints() -> Vec<PlanarConstraint> {
    vec![
        PlanarConstraint::Coincident {
            points: [entity(1), entity(2)],
        },
        PlanarConstraint::Horizontal {
            points: [entity(1), entity(2)],
        },
        PlanarConstraint::Vertical {
            points: [entity(1), entity(2)],
        },
        PlanarConstraint::Fix {
            point: entity(1),
            position: [0.0, -0.0],
        },
        PlanarConstraint::ArcMidpoint {
            point: entity(5),
            arc: entity(14),
        },
        PlanarConstraint::Midpoint {
            point: entity(5),
            line: entity(10),
        },
        PlanarConstraint::PointOnLine {
            point: entity(5),
            line: entity(10),
        },
        PlanarConstraint::Offset {
            point: entity(5),
            line: entity(10),
            parameter: 91,
            side: LineContactSide::Right,
            linear_diameter: true,
        },
        PlanarConstraint::Angle {
            lines: [entity(10), entity(11)],
            parameter: 92,
            turn: AngleTurn::Clockwise,
        },
        PlanarConstraint::ThreePointAngle {
            vertex: entity(2),
            points: [entity(1), entity(5)],
            parameter: 92,
            turn: AngleTurn::CounterClockwise,
        },
        PlanarConstraint::Lines {
            lines: [entity(10), entity(11)],
            relation: LineRelation::Parallel,
        },
        PlanarConstraint::PointOnCircularCurve {
            point: entity(5),
            curve: entity(14),
        },
        PlanarConstraint::Concentric {
            curves: [entity(12), entity(14)],
        },
        PlanarConstraint::EqualRadius {
            curves: [entity(12), entity(14)],
        },
        PlanarConstraint::Diameter {
            curve: entity(12),
            parameter: 91,
        },
        PlanarConstraint::ArcLength {
            arc: entity(14),
            parameter: 91,
        },
        PlanarConstraint::Radius {
            curve: entity(14),
            parameter: 91,
        },
        PlanarConstraint::LineTangent {
            line: entity(10),
            curve: entity(14),
            side: LineContactSide::Left,
        },
        PlanarConstraint::TangentDistance {
            line: entity(10),
            curve: entity(14),
            parameter: 91,
            options: TangentDistanceOptions {
                far_side: false,
                linear_diameter: true,
            },
            contact: TangentDistanceContact {
                side: LineContactSide::Left,
                lean: Some(TangentLean::Negative),
            },
        },
        PlanarConstraint::CircularTangent {
            curves: [entity(12), entity(14)],
            contact: CircleContact::SecondContainsFirst,
        },
        PlanarConstraint::Distance {
            points: [entity(1), entity(2)],
            parameter: 91,
            measure: DistanceMeasure::Aligned,
        },
    ]
}
#[test]
fn every_authored_relation_round_trips_without_solver_or_structural_records() {
    let mut original = rich_sketch();
    for (index, value) in constraints().into_iter().enumerate() {
        relation(&mut original, index as u64 + 1, value);
    }
    let encoded = serde_json::to_string(&original.recipe()).unwrap();
    for forbidden in ["gradient", "circularity", "numerical", "projection"] {
        assert!(!encoded.contains(forbidden));
    }
    let restored: SketchRecipe = serde_json::from_str(&encoded).unwrap();
    assert_eq!(Sketch2D::from_recipe(&restored).unwrap(), original);
    assert_eq!(serde_json::to_string(&restored).unwrap(), encoded);
    assert_eq!(restored.constraints.len(), 21);
    assert!(!restored.arcs[0].counter_clockwise);
    assert_eq!(
        restored.lines[0].endpoints[1],
        restored.lines[1].endpoints[0]
    );
}
#[test]
fn recipe_restore_rejects_invalid_versions_identities_values_references_and_contact_shape() {
    let original = rich_sketch().recipe();
    let mut candidates = Vec::new();
    let mut recipe = original.clone();
    recipe.version = 99;
    candidates.push((recipe, "99"));
    let mut recipe = original.clone();
    recipe.points[1].id = entity(1);
    candidates.push((recipe, "unique"));
    let mut recipe = original.clone();
    recipe.points[0].id = entity(0);
    candidates.push((recipe, "nonzero"));
    let mut recipe = original.clone();
    recipe.circles[0].id = entity(10);
    candidates.push((recipe, "unique"));
    let mut recipe = original.clone();
    recipe.points[0].position[0] = f64::NAN;
    candidates.push((recipe, "finite"));
    let mut recipe = original.clone();
    recipe.circles[0].radius = -1.0;
    candidates.push((recipe, "positive"));
    let mut recipe = original.clone();
    recipe.lines[0].endpoints[1] = entity(999);
    candidates.push((recipe, "999"));
    let mut recipe = original.clone();
    recipe.arcs[0].end = entity(999);
    candidates.push((recipe, "999"));
    let mut recipe = original.clone();
    recipe.circles[0].center = entity(999);
    candidates.push((recipe, "999"));
    let mut sketch = rich_sketch();
    relation(
        &mut sketch,
        1,
        PlanarConstraint::Radius {
            curve: entity(12),
            parameter: 91,
        },
    );
    let mut recipe = sketch.recipe();
    recipe.constraints.push(recipe.constraints[0].clone());
    candidates.push((recipe, "unique"));
    let mut recipe = sketch.recipe();
    recipe.constraints[0].relation = PlanarConstraint::Radius {
        curve: entity(999),
        parameter: 91,
    };
    candidates.push((recipe, "999"));
    let mut recipe = sketch.recipe();
    recipe.constraints[0].relation = PlanarConstraint::TangentDistance {
        line: entity(10),
        curve: entity(14),
        parameter: 91,
        options: Default::default(),
        contact: TangentDistanceContact {
            side: LineContactSide::Left,
            lean: None,
        },
    };
    candidates.push((recipe, "lean"));
    for (recipe, expected) in candidates {
        assert!(
            Sketch2D::from_recipe(&recipe)
                .unwrap_err()
                .to_string()
                .contains(expected)
        );
    }
    assert_eq!(rich_sketch().recipe(), original);
}
#[test]
fn strict_records_refuse_unknown_fields_missing_fields_and_unknown_variants() {
    let encoded = serde_json::to_value(rich_sketch().recipe()).unwrap();
    for pointer in ["", "/points/0", "/lines/0", "/circles/0", "/arcs/0"] {
        let mut value = encoded.clone();
        value
            .pointer_mut(pointer)
            .unwrap()
            .as_object_mut()
            .unwrap()
            .insert("unsupported".into(), serde_json::Value::Bool(true));
        assert!(
            serde_json::from_value::<SketchRecipe>(value).is_err(),
            "{pointer}"
        );
    }
    let mut value = encoded.clone();
    value.as_object_mut().unwrap().remove("version");
    assert!(serde_json::from_value::<SketchRecipe>(value).is_err());
    let mut sketch = rich_sketch();
    relation(
        &mut sketch,
        1,
        PlanarConstraint::Radius {
            curve: entity(12),
            parameter: 91,
        },
    );
    let mut value = serde_json::to_value(sketch.recipe()).unwrap();
    value["constraints"][0]["relation"] = serde_json::json!({"unsupported":{}});
    assert!(serde_json::from_value::<SketchRecipe>(value).is_err());
    let mut value = serde_json::to_value(sketch.recipe()).unwrap();
    value["constraints"][0]["relation"]["radius"]["unsupported"] = true.into();
    assert!(serde_json::from_value::<SketchRecipe>(value).is_err());
}
#[test]
fn restored_parametric_rectangle_solves_and_updates_with_zero_dof() {
    let original = rectangle();
    let encoded = serde_json::to_string(&original.recipe()).unwrap();
    let recipe: SketchRecipe = serde_json::from_str(&encoded).unwrap();
    let mut sketch = Sketch2D::from_recipe(&recipe).unwrap();
    let mut parameters = parameters();
    for width in [5.0, 8.0] {
        parameters
            .set_expression(1, &format!("{width} cm"))
            .unwrap();
        let result = sketch
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(result.numerical().termination, SolveTermination::Converged);
        assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
        assert_eq!(result.candidate().constraints(), original.constraints());
        let encoded = serde_json::to_string(&result.candidate().recipe()).unwrap();
        let restored: SketchRecipe = serde_json::from_str(&encoded).unwrap();
        sketch = Sketch2D::from_recipe(&restored).unwrap();
        assert_point(&sketch, 3, [width, width / 2.0]);
    }
}
#[test]
fn recipes_preserve_float_bits_and_editable_draft_geometry() {
    let mut sketch = Sketch2D::default();
    let values = [
        -0.0,
        f64::from_bits(1),
        1.2345678901234567,
        f64::MAX,
        1e-200,
        1e200,
    ];
    for (index, value) in values.iter().enumerate() {
        sketch
            .add_point(entity(index as u64 + 1), [*value, *value])
            .unwrap();
    }
    sketch.add_line(entity(10), [entity(1), entity(1)]).unwrap();
    sketch
        .add_arc(entity(11), entity(1), [entity(1), entity(1)], true)
        .unwrap();
    let encoded = serde_json::to_string(&sketch.recipe()).unwrap();
    let recipe: SketchRecipe = serde_json::from_str(&encoded).unwrap();
    let restored = Sketch2D::from_recipe(&recipe).unwrap();
    for (actual, expected) in restored.points().iter().zip(values) {
        assert_eq!(actual.position[0].to_bits(), expected.to_bits());
    }
    assert_eq!(restored, sketch);
    let empty = Sketch2D::default();
    assert_eq!(Sketch2D::from_recipe(&empty.recipe()).unwrap(), empty);
}
#[test]
fn fixed_schema_fixture_preserves_the_documented_names_and_field_order() {
    let fixture = include_str!("../fixtures/rust-sketch-recipe-v1.json").trim();
    let recipe: SketchRecipe = serde_json::from_str(fixture).unwrap();
    let restored = Sketch2D::from_recipe(&recipe).unwrap();
    assert_eq!(restored.point(entity(3)).unwrap().position, [2.0, -3.0]);
    assert_eq!(restored.lines()[0].endpoints, [entity(1), entity(2)]);
    assert_eq!(
        restored.constraints()[0].relation,
        PlanarConstraint::Offset {
            point: entity(3),
            line: entity(4),
            parameter: 7,
            side: LineContactSide::Right,
            linear_diameter: true,
        }
    );
    assert_eq!(serde_json::to_string(&restored.recipe()).unwrap(), fixture);
}
