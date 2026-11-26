// SPDX-License-Identifier: GPL-2.0-only
//! Builder qualification for the straight Sweep and two-section Loft fixture geometry.
use martensite_tkbrep::{
    topexp::Explorer,
    topods::{Edge, Shape},
};
use martensite_tkernel::standard::Failure;
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tkgeombase::gprop::GProps;
use martensite_tkmath::gp::{Axis2, Circle3, Direction3, Point3};
use martensite_tkoffset::brep_offset_api::{BRepOffsetAPIMakePipe, BRepOffsetAPIThruSections};
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakeSweep;
use martensite_tktopalgo::{
    brep_builder_api::{
        BRepBuilderAPIMakeEdge, BRepBuilderAPIMakeFace, BRepBuilderAPIMakeShape,
        BRepBuilderAPIMakeWire,
    },
    brep_check::BRepCheckAnalyzer,
    brep_gprop::BRepGProp,
};
use std::f64::consts::PI;

#[test]
fn sweep_along_the_fixture_line_returns_a_cylinder_with_history() -> Result<(), Failure> {
    let spine =
        BRepBuilderAPIMakeEdge::from_points(Point3::default(), Point3::new(0., 0., 3.))?.edge()?;
    let path = BRepBuilderAPIMakeWire::from_edge(&spine)?.wire()?;
    let section_edge = circle(0., 0.5)?;
    let section_wire = BRepBuilderAPIMakeWire::from_edge(&section_edge)?.wire()?;
    let section = BRepBuilderAPIMakeFace::from_wire(&section_wire, true)?.face()?;
    let mut sweep = BRepOffsetAPIMakePipe::new(&path, section.as_ref())?;
    let solid = sweep.shape()?.clone();
    let wall = sweep.generated_by(spine.as_ref(), section_edge.as_ref())?;
    let boundaries = [sweep.first_shape()?, sweep.last_shape()?, wall];
    verify_solid(&solid, 0.75 * PI, &boundaries)
}

#[test]
fn loft_between_the_fixture_sections_returns_a_frustum_with_history() -> Result<(), Failure> {
    for ruled in [true, false] {
        verify_loft(ruled)?;
    }
    Ok(())
}

fn verify_loft(ruled: bool) -> Result<(), Failure> {
    let mut loft = BRepOffsetAPIThruSections::new(true, ruled, 1e-6);
    loft.set_mutable_input(false);
    let sections = [circle(0., 1.)?, circle(3., 0.5)?];
    for section in &sections {
        loft.add_wire(&BRepBuilderAPIMakeWire::from_edge(section)?.wire()?);
    }
    let solid = loft.shape()?.clone();
    let wall = loft_wall(&mut loft, &sections[0])?;
    assert!(wall.is_same(&loft_wall(&mut loft, &sections[1])?));
    let boundaries = [
        loft.first_shape().as_ref().clone(),
        loft.last_shape().as_ref().clone(),
        wall,
    ];
    verify_solid(&solid, 1.75 * PI, &boundaries)
}

fn loft_wall(loft: &mut BRepOffsetAPIThruSections, section: &Edge) -> Result<Shape, Failure> {
    let history = loft.generated(section.as_ref())?;
    assert_eq!(history.len(), 1);
    Ok(history.first().unwrap().clone())
}

fn circle(z: f64, radius: f64) -> Result<Edge, Failure> {
    let frame = Axis2::new(Point3::new(0., 0., z), Direction3::Z)?;
    BRepBuilderAPIMakeEdge::from_circ(Circle3::new(frame, radius)?)?.edge()
}

fn verify_solid(shape: &Shape, volume: f64, boundaries: &[Shape; 3]) -> Result<(), Failure> {
    assert_eq!(shape.shape_type()?, ShapeEnum::Solid);
    assert!(BRepCheckAnalyzer::new(shape, true, false, true)?.is_valid()?);
    let faces: Vec<_> = Explorer::new(shape, ShapeEnum::Face, ShapeEnum::Shape)?.collect();
    assert_eq!(faces.len(), boundaries.len());
    for face in &faces {
        assert_eq!(
            boundaries
                .iter()
                .filter(|named| named.is_same(face))
                .count(),
            1
        );
    }
    let mut properties = GProps::default();
    let achieved = BRepGProp::volume_properties_eps(shape, &mut properties, 1e-9, false, false)?;
    assert!(achieved.is_finite() && achieved.abs() <= 1e-9, "{achieved}");
    assert!((properties.mass() - volume).abs() < 1e-8, "{properties:?}");
    Ok(())
}
