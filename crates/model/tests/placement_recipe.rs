// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::{
    SketchPathMember, SketchPlacementBinding, SketchPlacementGeometry, SketchPlacementRecipe,
    SketchPlacementRelation,
};
use serde::Deserialize;
#[derive(Deserialize)]
#[serde(rename_all = "PascalCase")]
struct GoRecipe {
    kind: String,
    locked: Option<Vec<GoLockedArc>>,
    solved_points: Option<Vec<[f64; 2]>>,
    curved: bool,
    input: [[f64; 2]; 3],
    width: f64,
    #[serde(rename = "CCW")]
    ccw: bool,
    points: Vec<[f64; 2]>,
    entities: Vec<GoEntity>,
    relations: Vec<GoRelation>,
    fields: Vec<GoField>,
    #[serde(rename = "DOF")]
    dof: usize,
}
#[derive(Deserialize)]
#[serde(rename_all = "PascalCase")]
struct GoEntity {
    radius: f64,
    kind: String,
    points: Vec<usize>,
    #[serde(rename = "CCW")]
    ccw: bool,
    construction: bool,
}
#[derive(Deserialize)]
#[serde(rename_all = "PascalCase")]
struct GoRelation {
    kind: String,
    entities: Vec<usize>,
    points: Option<Vec<usize>>,
}
#[derive(Deserialize)]
#[serde(rename_all = "PascalCase")]
struct GoField {
    label: String,
    value: f64,
    witness: [[f64; 2]; 2],
    binding: String,
    points: [usize; 2],
    entity: i64,
    orientation: String,
}
#[test]
fn constrained_slot_graphs_and_field_bindings_match_the_independent_go_recipes() {
    let fixtures: Vec<GoRecipe> =
        serde_json::from_str(include_str!("placement_recipe/go-recipes.json")).unwrap();
    assert_eq!(fixtures.len(), 47);
    for fixture in fixtures
        .into_iter()
        .filter(|fixture| fixture.kind.is_empty())
    {
        let [center, start, end] = fixture.input;
        let actual = if fixture.curved {
            SketchPlacementRecipe::arc_slot(center, [start, end], fixture.width, fixture.ccw)
        } else {
            SketchPlacementRecipe::straight_slot([center, start], fixture.width)
        }
        .unwrap();
        assert_eq!(fixture.dof, if fixture.curved { 6 } else { 5 });
        assert_eq!(actual.points.len(), fixture.points.len());
        for (a, b) in actual
            .points
            .iter()
            .flatten()
            .zip(fixture.points.iter().flatten())
        {
            close(*a, *b);
        }
        verify_entities(&actual, &fixture);
        verify_relations(&actual, &fixture);
        verify_fields(&actual, &fixture);
    }
}
fn close(actual: f64, expected: f64) {
    // Shared slot seed arithmetic can differ in rounding order from Go's GUI recipe formula.
    let bound = 64.0 * f64::EPSILON * expected.abs().max(1.0);
    assert!(
        (actual - expected).abs() <= bound,
        "{actual} != {expected}, bound {bound}"
    );
}
fn verify_entities(actual: &SketchPlacementRecipe, fixture: &GoRecipe) {
    assert_eq!(actual.members.len(), fixture.entities.len());
    for (a, b) in actual.members.iter().zip(&fixture.entities) {
        let expected = if b.kind == "circle" {
            SketchPlacementGeometry::Circle {
                center: b.points[0],
                radius: b.radius,
            }
        } else {
            SketchPlacementGeometry::Path(if b.kind == "line" {
                SketchPathMember::Line([b.points[0], b.points[1]])
            } else {
                SketchPathMember::Arc {
                    center: b.points[0],
                    endpoints: [b.points[1], b.points[2]],
                    counter_clockwise: b.ccw,
                }
            })
        };
        assert_eq!(a.geometry, expected);
        assert_eq!(a.construction, b.construction);
    }
}
fn verify_relations(actual: &SketchPlacementRecipe, fixture: &GoRecipe) {
    assert_eq!(actual.relations.len(), fixture.relations.len());
    for (a, b) in actual.relations.iter().zip(&fixture.relations) {
        let expected = match b.kind.as_str() {
            "tangent" => SketchPlacementRelation::LineTangent {
                line: b.entities[0],
                curve: b.entities[1],
            },
            "circularTangent" => {
                SketchPlacementRelation::CircularTangent([b.entities[0], b.entities[1]])
            }
            "equalRadius" => SketchPlacementRelation::EqualRadius([b.entities[0], b.entities[1]]),
            "horizontalLine" => SketchPlacementRelation::LineHorizontal(b.entities[0]),
            "verticalLine" => SketchPlacementRelation::LineVertical(b.entities[0]),
            "perpendicular" => {
                SketchPlacementRelation::Perpendicular([b.entities[0], b.entities[1]])
            }
            "midpoint" => SketchPlacementRelation::Midpoint {
                point: b.points.as_ref().unwrap()[0],
                line: b.entities[0],
            },
            _ => panic!("unknown relation {}", b.kind),
        };
        assert_eq!(*a, expected);
    }
}
fn verify_fields(actual: &SketchPlacementRecipe, fixture: &GoRecipe) {
    assert_eq!(actual.fields.len(), fixture.fields.len());
    for (a, b) in actual.fields.iter().zip(&fixture.fields) {
        assert_eq!(a.label, b.label);
        close(a.value, b.value);
        for (a, b) in a.witness.iter().flatten().zip(b.witness.iter().flatten()) {
            close(*a, *b);
        }
        let expected = match b.binding.as_str() {
            "distance" => match b.orientation.as_str() {
                "horizontal" => SketchPlacementBinding::HorizontalDistance(b.points),
                "vertical" => SketchPlacementBinding::VerticalDistance(b.points),
                "" | "aligned" => SketchPlacementBinding::Distance(b.points),
                value => panic!("unknown orientation {value}"),
            },
            "radius" => SketchPlacementBinding::Radius(usize::try_from(b.entity).unwrap()),
            "diameter" => SketchPlacementBinding::Diameter(usize::try_from(b.entity).unwrap()),
            "steering" => SketchPlacementBinding::SteeringAngle,
            _ => panic!("unknown binding {}", b.binding),
        };
        assert_eq!(a.binding, expected);
        assert_eq!(
            a.binding.dimension(),
            if b.binding == "steering" {
                oblikovati_model::quantity::Dimension::Angle
            } else {
                oblikovati_model::quantity::Dimension::Length
            }
        );
    }
}

#[path = "placement_recipe/rectangles.rs"]
mod rectangles;

#[path = "placement_recipe/circles.rs"]
mod circles;

#[path = "placement_recipe/arcs.rs"]
mod arcs;

#[derive(Deserialize)]
#[serde(rename_all = "PascalCase")]
struct GoLockedArc {
    radius: f64,
    points: Vec<[f64; 2]>,
    #[serde(rename = "DOF")]
    dof: usize,
}
