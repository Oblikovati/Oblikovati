// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::{
    SketchPathMember, SketchPlacementBinding, SketchPlacementGeometry, SketchPlacementRecipe,
    SketchPlacementRelation,
};
use serde::Deserialize;
#[derive(Deserialize)]
#[serde(rename_all = "PascalCase")]
struct GoRecipe {
    #[serde(default)]
    chain: Vec<[f64; 2]>,
    #[serde(default)]
    closed: bool,
    sides: usize,
    inscribed: bool,
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
    major_axis: [f64; 2],
    minor_radius: f64,
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
    assert_eq!(fixtures.len(), 103);
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
    for (actual, expected) in actual.members.iter().zip(&fixture.entities) {
        if expected.kind == "ellipse" {
            ellipses::verify_frame(actual.geometry, expected);
        } else if fixture.kind == "polygon" && expected.kind == "circle" {
            polygons::verify_circle(actual.geometry, expected);
        } else {
            assert_eq!(actual.geometry, expected_geometry(expected));
        }
        assert_eq!(actual.construction, expected.construction);
    }
}
fn expected_geometry(entity: &GoEntity) -> SketchPlacementGeometry {
    if entity.kind == "point" {
        return SketchPlacementGeometry::Point(entity.points[0]);
    }
    if entity.kind == "circle" {
        return SketchPlacementGeometry::Circle {
            center: entity.points[0],
            radius: entity.radius,
        };
    }
    SketchPlacementGeometry::Path(if entity.kind == "line" {
        SketchPathMember::Line([entity.points[0], entity.points[1]])
    } else {
        SketchPathMember::Arc {
            center: entity.points[0],
            endpoints: [entity.points[1], entity.points[2]],
            counter_clockwise: entity.ccw,
        }
    })
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
            "pointOnCircle" => SketchPlacementRelation::PointOnCircle {
                point: b.points.as_ref().unwrap()[0],
                circle: b.entities[0],
            },
            "equalLength" => SketchPlacementRelation::EqualLength([b.entities[0], b.entities[1]]),
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
            "ellipseRadius" => {
                SketchPlacementBinding::EllipseRadius(usize::try_from(b.entity).unwrap())
            }
            "steeringLength" => SketchPlacementBinding::SteeringLength,
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

#[path = "placement_recipe/ellipses.rs"]
mod ellipses;

#[path = "placement_recipe/polygons.rs"]
mod polygons;

#[path = "placement_recipe/slot_centerline.rs"]
mod slot_centerline;

#[path = "placement_recipe/points.rs"]
mod points;

#[path = "placement_recipe/chains.rs"]
mod chains;
