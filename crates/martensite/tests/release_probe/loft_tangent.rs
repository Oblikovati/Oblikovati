// SPDX-License-Identifier: GPL-2.0-only
//! Qualify the native boundary history for a circular Loft takeoff condition.
use martensite_tkbrep::{
    brep::BRepTool,
    topexp::{self, Explorer},
    topods::{Builder, Shape, Shell},
};
use martensite_tkernel::{
    ncollection::Array1,
    standard::{Failure, Handle},
};
use martensite_tkg3d::{
    geom::{GeomBezierCurve, GeomCurve},
    top_abs::ShapeEnum,
};
use martensite_tkgeombase::gprop::GProps;
use martensite_tkmath::gp::{Axis1, Direction3, Point3};
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakeRevol;
use martensite_tktopalgo::{
    brep_builder_api::{BRepBuilderAPIMakeEdge, BRepBuilderAPIMakeShape},
    brep_check::BRepCheckAnalyzer,
    brep_gprop::BRepGProp,
};

#[test]
fn circular_loft_end_tangency_keeps_both_generated_trims() -> Result<(), Failure> {
    let axial = measure_conditioned_surface(90., 2.)?;
    let angled = measure_conditioned_surface(45., 2.)?;
    let frustum = 1.5 * std::f64::consts::PI * 9.25_f64.sqrt();
    assert!((axial - frustum).abs() > 0.1, "{axial} vs {frustum}");
    assert!((angled - axial).abs() > 0.1, "{angled} vs {axial}");
    Ok(())
}

fn measure_conditioned_surface(angle_degrees: f64, weight: f64) -> Result<f64, Failure> {
    let (sine, cosine) = angle_degrees.to_radians().sin_cos();
    let poles = [
        Point3::new(1., 0., 0.),
        Point3::new(1. + weight * cosine, 0., weight * sine),
        Point3::new(2. / 3., 0., 2.),
        Point3::new(0.5, 0., 3.),
    ];
    let poles = Array1::from_shared(&poles, 1);
    let curve: Handle<dyn GeomCurve> = Handle::new(GeomBezierCurve::new(&poles)?).into_upcast();
    let edge = BRepBuilderAPIMakeEdge::from_curve(&curve)?.edge()?;
    let axis = Axis1::new(Point3::default(), Direction3::Z);
    let mut revolved = BRepPrimAPIMakeRevol::full(edge.as_ref(), axis, true)?;
    let face = revolved.shape()?.clone();
    assert_eq!(face.shape_type()?, ShapeEnum::Face);
    let shell = one_face_shell(&face)?;
    assert!(BRepCheckAnalyzer::new(&shell, true, false, true)?.is_valid()?);
    assert!(!BRepTool::is_closed(&shell)?);
    let edges: Vec<_> = Explorer::new(&shell, ShapeEnum::Edge, ShapeEnum::Shape)?.collect();
    let tips = [
        topexp::first_vertex(&edge, true),
        topexp::last_vertex(&edge, true),
    ];
    for tip in tips {
        let history = revolved.generated(tip.as_ref())?;
        assert_eq!(history.len(), 1);
        let trim = history.first().unwrap();
        assert_eq!(trim.shape_type()?, ShapeEnum::Edge);
        assert_eq!(edges.iter().filter(|edge| edge.is_same(trim)).count(), 1);
    }
    let mut properties = GProps::default();
    let achieved = BRepGProp::surface_properties_eps(&shell, &mut properties, 1e-9, false)?;
    assert!(achieved.is_finite() && achieved.abs() <= 1e-9, "{achieved}");
    Ok(properties.mass())
}

fn one_face_shell(face: &Shape) -> Result<Shape, Failure> {
    let mut shell = Shell::default();
    let builder = Builder;
    builder.make_shell(&mut shell);
    let mut shape = shell.into_shape();
    builder.add(&mut shape, face)?;
    Ok(shape)
}
