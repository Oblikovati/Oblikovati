// SPDX-License-Identifier: GPL-2.0-only
//! Reproduce the published cube tolerance check against the tagged kernel.
use martensite_tkbrep::{brep::BRepTool, topexp::Explorer, topods};
use martensite_tkernel::{message::ProgressRange, ncollection::List, standard::Failure};
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tkoffset::{
    brep_offset::MakeOffsetOptions, brep_offset_api::BRepOffsetAPIMakeThickSolid,
};
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakeBox;
use martensite_tktopalgo::brep_builder_api::BRepBuilderAPIMakeShape;

fn maximum_vertex_tolerance(shape: &topods::Shape) -> Result<f64, Failure> {
    let mut vertices = Explorer::new(shape, ShapeEnum::Vertex, ShapeEnum::Shape)?;
    let mut maximum: f64 = 0.;
    while vertices.more() {
        maximum = maximum.max(BRepTool::vertex_tolerance(&topods::vertex(
            vertices.current(),
        )?)?);
        vertices.next()?;
    }
    Ok(maximum)
}

fn main() -> Result<(), Failure> {
    let cube = BRepPrimAPIMakeBox::from_sizes(20., 20., 20.)?
        .shape()?
        .clone();
    let opening = Explorer::new(&cube, ShapeEnum::Face, ShapeEnum::Shape)?
        .current()
        .clone();
    let mut openings = List::default();
    openings.append(opening);
    let mut shell = BRepOffsetAPIMakeThickSolid::new();
    shell.make_thick_solid_by_join(
        &cube,
        &openings,
        -1.,
        1e-7,
        MakeOffsetOptions::default(),
        &ProgressRange::default(),
    )?;
    let source = maximum_vertex_tolerance(&cube)?;
    let result = maximum_vertex_tolerance(shell.shape()?)?;
    println!("source maximum vertex tolerance {source:.17e} mm");
    println!("result maximum vertex tolerance {result:.17e} mm");
    assert!(
        result <= source,
        "shell result increased the maximum vertex tolerance"
    );
    Ok(())
}
