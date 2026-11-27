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
fn sweep_the_fixture_circle_as_a_wire_returns_an_open_cylindrical_shell() -> Result<(), Failure> {
    let start = Point3::new(-2., -1.25, 2.);
    let end = Point3::new(-2., -1.25, 5.);
    let spine = BRepBuilderAPIMakeEdge::from_points(start, end)?.edge()?;
    let path = BRepBuilderAPIMakeWire::from_edge(&spine)?.wire()?;
    let frame = Axis2::new(start, Direction3::Z)?;
    let section = BRepBuilderAPIMakeEdge::from_circ(Circle3::new(frame, 0.5)?)?.edge()?;
    let profile = BRepBuilderAPIMakeWire::from_edge(&section)?.wire()?;
    let mut sweep = BRepOffsetAPIMakePipe::new(&path, profile.as_ref())?;
    let shell = sweep.shape()?.clone();

    assert_eq!(shell.shape_type()?, ShapeEnum::Shell);
    assert!(BRepCheckAnalyzer::new(&shell, true, false, true)?.is_valid()?);
    let wall = sweep.generated_by(spine.as_ref(), section.as_ref())?;
    let faces = Explorer::new(&shell, ShapeEnum::Face, ShapeEnum::Shape)?.collect::<Vec<_>>();
    assert_eq!(faces.len(), 1);
    assert!(faces[0].is_same(&wall));

    let start_wire = sweep.first_shape()?;
    let end_wire = sweep.last_shape()?;
    verify_surface_boundary(&shell, &start_wire)?;
    verify_surface_boundary(&shell, &end_wire)?;
    verify_surface_area(&shell, 3. * PI)
}

fn verify_surface_boundary(shell: &Shape, boundary: &Shape) -> Result<(), Failure> {
    assert_eq!(boundary.shape_type()?, ShapeEnum::Wire);
    let edges = Explorer::new(boundary, ShapeEnum::Edge, ShapeEnum::Shape)?.collect::<Vec<_>>();
    assert_eq!(edges.len(), 1);
    assert!(
        Explorer::new(shell, ShapeEnum::Edge, ShapeEnum::Shape)?
            .any(|edge| edge.is_same(&edges[0]))
    );
    let mut length = GProps::default();
    BRepGProp::linear_properties(boundary, &mut length, true, false)?;
    assert!((length.mass() - PI).abs() < 1e-8, "{length:?}");
    Ok(())
}

pub(super) fn verify_surface_area(shell: &Shape, area: f64) -> Result<(), Failure> {
    let mut properties = GProps::default();
    let achieved = BRepGProp::surface_properties_eps(shell, &mut properties, 1e-9, false)?;
    assert!(achieved.is_finite() && achieved.abs() <= 1e-9, "{achieved}");
    assert!((properties.mass() - area).abs() < 1e-8, "{properties:?}");
    Ok(())
}

#[test]
fn loft_between_the_fixture_sections_returns_a_frustum_with_history() -> Result<(), Failure> {
    for ruled in [true, false] {
        verify_loft(ruled)?;
    }
    Ok(())
}

#[test]
fn loft_between_the_fixture_sections_retains_both_open_trims() -> Result<(), Failure> {
    let mut loft = BRepOffsetAPIThruSections::new(false, false, 1e-6);
    loft.set_mutable_input(true);
    let sections = [circle(0., 1.)?, circle(3., 0.5)?];
    for section in &sections {
        loft.add_wire(&BRepBuilderAPIMakeWire::from_edge(section)?.wire()?);
    }
    let shell = loft.shape()?.clone();
    assert_eq!(shell.shape_type()?, ShapeEnum::Shell);
    assert!(!martensite_tkbrep::brep::BRepTool::is_closed(&shell)?);
    assert!(BRepCheckAnalyzer::new(&shell, true, false, true)?.is_valid()?);
    let wall = loft_wall(&mut loft, &sections[0])?;
    assert!(wall.is_same(&loft_wall(&mut loft, &sections[1])?));
    let faces: Vec<_> = Explorer::new(&shell, ShapeEnum::Face, ShapeEnum::Shape)?.collect();
    assert_eq!(faces.len(), 1);
    assert!(faces[0].is_same(&wall));
    let edges: Vec<_> = Explorer::new(&shell, ShapeEnum::Edge, ShapeEnum::Shape)?.collect();
    for section in &sections {
        assert!(edges.iter().any(|edge| edge.is_same(section.as_ref())));
    }
    verify_surface_area(&shell, 1.5 * PI * 9.25_f64.sqrt())
}

#[test]
fn loft_through_a_placed_section_preserves_the_caps_and_changes_volume() -> Result<(), Failure> {
    let baseline = placed_loft_volume(0.75)?;
    let expanded = placed_loft_volume(0.75 * 2_f64.sqrt())?;
    assert!(
        (baseline - 7. * PI / 6.).abs() < 1e-8,
        "baseline {baseline}"
    );
    assert!(expanded > baseline * 1.4, "{expanded} <= {baseline}");
    Ok(())
}

fn placed_loft_volume(mid_radius: f64) -> Result<f64, Failure> {
    let mut loft = BRepOffsetAPIThruSections::new(true, false, 1e-6);
    loft.set_mutable_input(false);
    let sections = [circle(0., 1.)?, circle(1., mid_radius)?, circle(2., 0.5)?];
    for section in &sections {
        loft.add_wire(&BRepBuilderAPIMakeWire::from_edge(section)?.wire()?);
    }
    let solid = loft.shape()?.clone();
    let wall = loft_wall(&mut loft, &sections[0])?;
    for section in &sections[1..] {
        assert!(loft_wall(&mut loft, section)?.is_same(&wall));
    }
    let boundaries = [
        loft.first_shape().as_ref().clone(),
        loft.last_shape().as_ref().clone(),
        wall,
    ];
    verify_solid_boundaries(&solid, &boundaries)
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
    let actual = verify_solid_boundaries(shape, boundaries)?;
    assert!((actual - volume).abs() < 1e-8, "{actual}");
    Ok(())
}

fn verify_solid_boundaries(shape: &Shape, boundaries: &[Shape]) -> Result<f64, Failure> {
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
    Ok(properties.mass())
}
