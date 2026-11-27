// SPDX-License-Identifier: GPL-2.0-only
//! Qualify the rectangular taper on the released Martensite draft builder.
use martensite_tkbrep::{
    brep::BRepTool, brep_adaptor::BRepAdaptorSurface, topexp::Explorer, topods, topods::Shape,
};
use martensite_tkernel::standard::Failure;
use martensite_tkg3d::{adaptor3d::Adaptor3dSurface, top_abs::ShapeEnum};
use martensite_tkgeombase::gprop::GProps;
use martensite_tkmath::gp::{Direction3, Plane3, Point3};
use martensite_tkoffset::brep_offset_api::BRepOffsetAPIDraftAngle;
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakeBox;
use martensite_tktopalgo::{
    brep_builder_api::BRepBuilderAPIMakeShape, brep_check::BRepCheckAnalyzer, brep_gprop::BRepGProp,
};

#[test]
fn rectangular_extrude_taper_preserves_topology_history() -> Result<(), Failure> {
    for degrees in [5.0_f64, -5.0] {
        check_taper(degrees.to_radians())?;
    }
    Ok(())
}

fn check_taper(angle: f64) -> Result<(), Failure> {
    let (width, depth, height) = (4.0, 3.0, 0.8);
    let source = BRepPrimAPIMakeBox::from_sizes(width, depth, height)?
        .shape()?
        .clone();
    let faces: Vec<_> = entities(&source, ShapeEnum::Face)?;
    let normal = Direction3::Z;
    let neutral = Plane3::new(Point3::default(), normal)?;
    let mut draft = BRepOffsetAPIDraftAngle::with_shape(&source)?;
    for face in &faces {
        let face = topods::face(face)?;
        let plane = BRepAdaptorSurface::with_face(&face, false)?.plane()?;
        if plane.axis().direction().z().abs() < 0.5 {
            draft.add(&face, normal, -angle, &neutral, true)?;
            assert!(draft.add_done()?, "draft face status {:?}", draft.status()?);
        }
    }
    draft.build(&Default::default())?;
    assert!(draft.is_done(), "draft build status {:?}", draft.status()?);
    let result = draft.shape()?.clone();
    assert_eq!(result.shape_type()?, ShapeEnum::Solid);
    assert!(BRepCheckAnalyzer::new(&result, true, false, true)?.is_valid()?);

    let mut properties = GProps::default();
    BRepGProp::volume_properties(&result, &mut properties, true, false, false)?;
    let slope = angle.tan();
    let expected = width * depth * height
        + (width + depth) * slope * height.powi(2)
        + 4.0 / 3.0 * slope.powi(2) * height.powi(3);
    assert!(
        (properties.mass() - expected).abs() < 1e-8,
        "volume {} vs {expected}",
        properties.mass()
    );

    for (kind, count) in [
        (ShapeEnum::Face, 6),
        (ShapeEnum::Edge, 12),
        (ShapeEnum::Vertex, 8),
    ] {
        let originals = entities(&source, kind)?;
        let results = entities(&result, kind)?;
        assert_eq!(originals.len(), count);
        assert_eq!(results.len(), count);
        let images = originals
            .iter()
            .map(|item| draft.modified_shape(item))
            .collect::<Result<Vec<_>, _>>()?;
        assert!(
            results
                .iter()
                .all(|item| images.iter().filter(|image| image.is_same(item)).count() == 1),
            "expected exact native {kind:?} history"
        );
    }

    let vertices = entities(&result, ShapeEnum::Vertex)?;
    let points = vertices
        .iter()
        .map(|vertex| BRepTool::pnt(&topods::vertex(vertex)?))
        .collect::<Result<Vec<_>, _>>()?;
    for (at, extra) in [(0., 0.), (height, height * slope)] {
        let on_cap: Vec<_> = points
            .iter()
            .filter(|point| (point.z() - at).abs() < 1e-9)
            .collect();
        assert_eq!(on_cap.len(), 4);
        let left = on_cap
            .iter()
            .map(|point| point.x())
            .fold(f64::INFINITY, f64::min);
        let right = on_cap
            .iter()
            .map(|point| point.x())
            .fold(f64::NEG_INFINITY, f64::max);
        assert!((left + extra).abs() < 1e-9 && (right - width - extra).abs() < 1e-9);
    }
    Ok(())
}

fn entities(shape: &Shape, kind: ShapeEnum) -> Result<Vec<Shape>, Failure> {
    let mut items = Vec::<Shape>::new();
    for item in Explorer::new(shape, kind, ShapeEnum::Shape)? {
        if !items.iter().any(|current| current.is_same(&item)) {
            items.push(item);
        }
    }
    Ok(items)
}
