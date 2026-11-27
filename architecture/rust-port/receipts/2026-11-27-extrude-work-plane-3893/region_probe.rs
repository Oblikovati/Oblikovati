// SPDX-License-Identifier: GPL-2.0-only
use martensite_tkbo::profile_extrusion::{
    DisplayTolerance, ProfileCurve, ProfileCurveId, ProfilePlane, Resolution, SketchProfile,
    ToleranceCap, build_planar_regions,
};

fn main() {
    for radius in [0.25, 0.3] {
        let diagonal = 2.0_f64.sqrt() * 2.0 * radius;
        let resolution = Resolution::new(
            1e-9 * diagonal,
            1e-4 * diagonal,
            1e-9,
            DisplayTolerance {
                chordal: 0.05,
                angular: 0.5,
            },
            ToleranceCap::Refuse(1e-4 * diagonal),
        );
        for (origin, center) in [([0., 0., 0.], [0., 0.]), ([0., 0., -1.2], [-2., -1.25])] {
            let profile = SketchProfile {
                plane: ProfilePlane {
                    origin,
                    ..ProfilePlane::world_xy()
                },
                vertices: vec![],
                curves: vec![ProfileCurve::Circle {
                    id: ProfileCurveId(2),
                    center,
                    radius,
                }],
            };
            match build_planar_regions(&profile, resolution) {
                Ok(regions) => {
                    assert_eq!(regions.regions().len(), 1);
                    let area = regions.regions()[0].area();
                    assert!(area.encloses(std::f64::consts::PI * radius * radius));
                    println!("radius={radius} origin={origin:?} center={center:?}: certified");
                }
                Err(cause) => {
                    println!("radius={radius} origin={origin:?} center={center:?}: {cause}");
                }
            }
        }
    }
}
