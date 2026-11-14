// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchEllipse2D;
#[test]
fn analytic_curve_conversion_keeps_positions_sweep_and_authored_order() {
    let mut sketch = Sketch2D::default();
    for (id, point) in [(1, [0., 0.]), (2, [2., 0.]), (3, [0., 2.])] {
        sketch.add_point(SketchEntityId(id), point).unwrap();
    }
    sketch
        .add_circle(SketchEntityId(11), SketchEntityId(1), 2.)
        .unwrap();
    sketch
        .add_arc(
            SketchEntityId(12),
            SketchEntityId(1),
            [SketchEntityId(2), SketchEntityId(3)],
            false,
        )
        .unwrap();
    sketch
        .add_ellipse(
            SketchEllipse2D::new(SketchEntityId(13), SketchEntityId(1), [1., 0.], [3., 1.])
                .unwrap(),
        )
        .unwrap();
    let (input, _) = profile(&sketch, SketchPlane::xy()).unwrap();
    assert!(matches!(
        input.curves[0],
        ProfileCurve::Circle {
            id: ProfileCurveId(11),
            center: [0., 0.],
            radius: 2.
        }
    ));
    assert!(matches!(
        input.curves[1],
        ProfileCurve::Arc {
            id: ProfileCurveId(12),
            start: ProfileVertexId(2),
            end: ProfileVertexId(3),
            sense: ArcSense::Clockwise,
            ..
        }
    ));
    assert!(matches!(
        input.curves[2],
        ProfileCurve::Ellipse {
            id: ProfileCurveId(13),
            major_axis: [3., 0.],
            minor_radius: 1.,
            ..
        }
    ));
}
