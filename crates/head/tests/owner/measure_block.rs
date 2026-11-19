// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId, parameter::ParameterCollection, units::UnitsOfMeasure,
};

fn fixture() -> (DocumentSnapshot, crate::HeadSketchEdit, SketchEntityId) {
    let mut snapshot = DocumentSnapshot {
        work_geometry: Default::default(),
        sketches: Default::default(),
        attributes: Default::default(),
        attachments: Default::default(),
        interests: Default::default(),
        resources: Default::default(),
        assets: Default::default(),
        model_states: None,
        assembly: None,
        drawing: None,
        body_names: Default::default(),
        design_views: Default::default(),
        part_design_views: None,
        sketch_settings: None,
        parameters: ParameterCollection::new(UnitsOfMeasure::default()),
        properties: Default::default(),
    };
    let mut geometry = Sketch2D::default();
    for (id, point) in [(1, [0., 0.]), (2, [1., 0.])] {
        geometry.add_point(SketchEntityId(id), point).unwrap();
    }
    geometry
        .add_line(SketchEntityId(3), [SketchEntityId(1), SketchEntityId(2)])
        .unwrap();
    let sketch = DocumentSketchId(1);
    snapshot
        .sketches
        .add(sketch, "Sketch", SketchPlane::xy(), geometry)
        .unwrap();
    snapshot
        .sketches
        .capture_block_definition("Rib", sketch, &[SketchEntityId(3)])
        .unwrap();
    let member = snapshot.sketches.block_definitions()[0].geometry().lines()[0].id;
    for (id, position, rotation, scale) in [
        (10, [0., 2.], std::f64::consts::FRAC_PI_2, 2.),
        (11, [3., 0.], 0., -1.),
    ] {
        snapshot
            .sketches
            .insert_block_instance(
                sketch,
                SketchBlockInstance::new(SketchEntityId(id), "Rib", position, rotation, scale)
                    .unwrap(),
            )
            .unwrap();
    }
    (
        snapshot,
        crate::HeadSketchEdit {
            document: 1,
            sketch,
            plane: SketchPlane::xy(),
            look_at_plane: true,
        },
        member,
    )
}

#[test]
fn block_and_loose_curves_use_placed_analytic_geometry_without_edits() {
    let (snapshot, edit, member) = fixture();
    let before = snapshot.sketches.clone();
    let picks = [
        MeasurePick::Curve(SketchEntityId(3)),
        MeasurePick::BlockCurve {
            instance: SketchEntityId(10),
            member,
        },
    ];
    let rows = rows(&snapshot, edit, &picks).unwrap();
    assert!(rows.contains(&"Length 10.000 mm".to_owned()), "{rows:?}");
    assert!(rows.contains(&"Length 20.000 mm".to_owned()), "{rows:?}");
    assert!(rows.contains(&"Angle 90.00 deg".to_owned()), "{rows:?}");
    assert!(
        rows.contains(&"Minimum distance 20.000 mm".to_owned()),
        "{rows:?}"
    );
    assert!(snapshot.sketches.same_recipe(&before));
}

#[test]
fn two_instances_keep_distinct_member_identities_and_scale() {
    let (snapshot, edit, member) = fixture();
    let picks = [
        MeasurePick::BlockCurve {
            instance: SketchEntityId(10),
            member,
        },
        MeasurePick::BlockCurve {
            instance: SketchEntityId(11),
            member,
        },
    ];
    let rows = rows(&snapshot, edit, &picks).unwrap();
    assert!(rows.contains(&"Length 20.000 mm".to_owned()), "{rows:?}");
    assert!(rows.contains(&"Length 10.000 mm".to_owned()), "{rows:?}");
    assert!(rows.contains(&"Angle 90.00 deg".to_owned()), "{rows:?}");
    assert!(
        rows.contains(&"Minimum distance 28.284 mm".to_owned()),
        "{rows:?}"
    );
}

#[test]
fn transformed_circle_reports_analytic_radius_center_and_length() {
    let (mut snapshot, edit, _) = fixture();
    let mut geometry = snapshot
        .sketches
        .get(edit.sketch)
        .unwrap()
        .geometry()
        .clone();
    geometry.add_point(SketchEntityId(4), [1., 0.]).unwrap();
    geometry
        .add_circle(SketchEntityId(5), SketchEntityId(4), 1.)
        .unwrap();
    snapshot
        .sketches
        .replace_geometry(edit.sketch, geometry)
        .unwrap();
    snapshot
        .sketches
        .capture_block_definition("Circle", edit.sketch, &[SketchEntityId(5)])
        .unwrap();
    let member = snapshot.sketches.block_definitions()[1]
        .geometry()
        .circles()[0]
        .id;
    let block = SketchBlockInstance::new(
        SketchEntityId(12),
        "Circle",
        [3., 1.],
        std::f64::consts::FRAC_PI_2,
        -2.,
    )
    .unwrap();
    snapshot
        .sketches
        .insert_block_instance(edit.sketch, block)
        .unwrap();
    let found = rows(
        &snapshot,
        edit,
        &[MeasurePick::BlockCurve {
            instance: SketchEntityId(12),
            member,
        }],
    )
    .unwrap();
    assert!(found.contains(&"Radius 20.000 mm".to_owned()), "{found:?}");
    assert!(found.contains(&"Length 125.664 mm".to_owned()), "{found:?}");
    assert!(
        found.contains(&"Center X 30.000 Y -10.000 Z 0.000 mm".to_owned()),
        "{found:?}"
    );
}
