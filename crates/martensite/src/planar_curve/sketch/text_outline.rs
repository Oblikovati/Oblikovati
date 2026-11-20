// SPDX-License-Identifier: GPL-2.0-only
//! Sketch text display samples analytic glyph edges; the model keeps only the text recipe.
use super::{SketchSamplingOptionsV1, sampling};
use martensite_tkbrep::{brep_adaptor::BRepAdaptorCurve, topexp::Explorer, topods};
use martensite_tkernel::ncollection::String8;
use martensite_tkg3d::adaptor3d::Adaptor3dCurve;
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tkgeombase::gcpnts::QuasiUniformDeflection;
use martensite_tkmath::{bnd::BoundingBox3, geom_abs::Shape};
use martensite_tkservice::{
    font::{FontFontAspect, FontStrictLevel},
    graphic3d::{
        Graphic3dHorizontalTextAlignment as AlignX, Graphic3dVerticalTextAlignment as AlignY,
    },
};
use martensite_tktopalgo::brep_bnd_lib::BRepBndLib;
use martensite_tkv3d::std_prs::{StdPrsBRepFont, StdPrsBRepFontTextOptions};
use oblikovati_api::types::SketchTextStyle;

/// Derive independent glyph edge paths for preview or committed display.
/// Example: `sample_sketch_text(&style, anchor, options)?`.
pub fn sample_sketch_text(
    style: &SketchTextStyle,
    anchor: [f64; 2],
    options: SketchSamplingOptionsV1,
) -> Result<Vec<Vec<[f64; 2]>>, String> {
    sampling::require_sampling_options(options)?;
    let size = working_em(style, anchor, options.working_scale)?;
    if style.content.trim().is_empty() {
        return Ok(Vec::new());
    }
    let shape = glyph_shape(style, size)?;
    let shift = vertical_shift(&shape, &style.v_align.0)?;
    let placement = TextPlacement {
        anchor,
        shift,
        rotation: style.rotation,
    };
    sample_glyph_edges(&shape, placement, options)
}

fn working_em(style: &SketchTextStyle, anchor: [f64; 2], scale: f64) -> Result<f64, String> {
    let em = if style.font_size == 0. {
        style.height
    } else {
        style.font_size
    };
    let size = em / scale;
    if scale > 0.
        && scale.is_finite()
        && size > 0.
        && size.is_finite()
        && style.rotation.is_finite()
        && anchor.iter().all(|value| value.is_finite())
    {
        return Ok(size);
    }
    Err(format!(
        "sketch text size {size:?}, scale {scale:?}, anchor {anchor:?}, rotation {:?}: expected positive finite size and scale and a finite placement",
        style.rotation
    ))
}

fn glyph_shape(style: &SketchTextStyle, size: f64) -> Result<topods::Shape, String> {
    let (family, search) = if style.family.is_empty() {
        ("sans-serif", FontStrictLevel::Any)
    } else {
        (style.family.as_str(), FontStrictLevel::Aliases)
    };
    let font = StdPrsBRepFont::from_name(
        &native_string(family)?,
        FontFontAspect::Regular,
        size,
        search,
    );
    if !font.is_valid() {
        return Err(format!(
            "sketch text font {family:?}: expected an available font family"
        ));
    }
    let options = StdPrsBRepFontTextOptions {
        horizontal_alignment: horizontal_alignment(&style.h_align.0)?,
        vertical_alignment: AlignY::Topfirstline,
        ..Default::default()
    };
    let shape = font.render_text_with(&native_string(&style.content)?, &options);
    if shape.is_null() {
        return Err(format!(
            "sketch text in font {family:?}: expected renderable glyph outlines"
        ));
    }
    Ok(shape)
}

fn native_string(value: &str) -> Result<String8, String> {
    let length = i32::try_from(value.len()).map_err(|_| {
        format!(
            "sketch text byte count {}: expected an i32 length",
            value.len()
        )
    })?;
    if value.contains('\0') {
        return Err("sketch text contains NUL: expected a UTF-8 string without NUL".into());
    }
    Ok(String8::from_unicode_units(value.as_bytes(), length))
}

fn horizontal_alignment(value: &str) -> Result<AlignX, String> {
    match value {
        "" | "left" => Ok(AlignX::Left),
        "center" => Ok(AlignX::Center),
        "right" => Ok(AlignX::Right),
        _ => Err(format!(
            "text alignment {value:?}: expected left, center or right"
        )),
    }
}

fn vertical_shift(shape: &topods::Shape, value: &str) -> Result<f64, String> {
    if value.is_empty() || value == "baseline" {
        return Ok(0.);
    }
    let mut bounds = BoundingBox3::default();
    BRepBndLib::add_optimal(shape, &mut bounds, false, false).map_err(|cause| cause.to_string())?;
    let [_, min, _, _, max, _] = bounds.get().map_err(|cause| cause.to_string())?;
    match value {
        "lower" => Ok(-min),
        "middle" => Ok(-min.midpoint(max)),
        "upper" => Ok(-max),
        _ => Err(format!(
            "text alignment {value:?}: expected baseline, lower, middle or upper"
        )),
    }
}

#[derive(Clone, Copy)]
struct TextPlacement {
    anchor: [f64; 2],
    shift: f64,
    rotation: f64,
}

fn sample_glyph_edges(
    shape: &topods::Shape,
    placement: TextPlacement,
    options: SketchSamplingOptionsV1,
) -> Result<Vec<Vec<[f64; 2]>>, String> {
    let mut edges = Explorer::new(shape, ShapeEnum::Edge, ShapeEnum::Shape)
        .map_err(|cause| cause.to_string())?;
    let mut paths = Vec::new();
    while edges.more() {
        let samples = edge_samples(edges.current(), options)?;
        paths.push(place_samples(samples, placement)?);
        Explorer::next(&mut edges).map_err(|cause| cause.to_string())?;
    }
    if paths.is_empty() {
        return Err("sketch text shape has no edges: expected glyph outlines".into());
    }
    Ok(paths)
}

fn edge_samples(
    edge: &topods::Shape,
    options: SketchSamplingOptionsV1,
) -> Result<Vec<[f64; 2]>, String> {
    let edge = topods::edge(edge).map_err(|cause| cause.to_string())?;
    let curve = BRepAdaptorCurve::with_edge(&edge).map_err(|cause| cause.to_string())?;
    let samples = QuasiUniformDeflection::with_curve(
        &curve as &dyn Adaptor3dCurve,
        options.deflection,
        Shape::C1,
    )
    .map_err(|cause| cause.to_string())?;
    sampling::sampled_positions(&samples, options.maximum_curve_points)
}

fn place_samples(
    samples: Vec<[f64; 2]>,
    placement: TextPlacement,
) -> Result<Vec<[f64; 2]>, String> {
    let (sin, cos) = placement.rotation.sin_cos();
    let placed: Vec<_> = samples
        .into_iter()
        .map(|[x, y]| {
            let y = y + placement.shift;
            [
                placement.anchor[0] + x * cos - y * sin,
                placement.anchor[1] + x * sin + y * cos,
            ]
        })
        .collect();
    if placed
        .iter()
        .flatten()
        .any(|coordinate| !coordinate.is_finite())
    {
        return Err(
            "sketch text outline exceeded the working coordinate range: expected finite vertices"
                .into(),
        );
    }
    Ok(placed)
}

/// Find the nearest point on the analytic glyph shape. For example,
/// `pick_sketch_text(&style, anchor, scale, cursor)?` for a canvas context menu.
pub fn pick_sketch_text(
    style: &SketchTextStyle,
    anchor: [f64; 2],
    scale: f64,
    cursor: [f64; 2],
) -> Result<Option<[f64; 2]>, String> {
    let size = working_em(style, anchor, scale)?;
    if style.content.trim().is_empty() {
        return Ok(None);
    }
    let shape = glyph_shape(style, size)?;
    let shift = vertical_shift(&shape, &style.v_align.0)?;
    let (sin, cos) = style.rotation.sin_cos();
    let [x, y] = [cursor[0] - anchor[0], cursor[1] - anchor[1]];
    let local = [x * cos + y * sin, y * cos - x * sin - shift];
    let [x, y] = closest_glyph_point(&shape, local)?;
    let y = y + shift;
    Ok(Some([
        anchor[0] + x * cos - y * sin,
        anchor[1] + x * sin + y * cos,
    ]))
}

fn closest_glyph_point(shape: &topods::Shape, [x, y]: [f64; 2]) -> Result<[f64; 2], String> {
    use martensite_tkmath::gp::Point3;
    use martensite_tktopalgo::{
        brep_builder_api::BRepBuilderAPIMakeVertex, brep_extrema::BRepExtremaDistShapeShape,
    };
    let target: topods::Shape = BRepBuilderAPIMakeVertex::new(Point3::new(x, y, 0.))
        .and_then(|builder| builder.vertex())
        .map_err(|cause| cause.to_string())?
        .into();
    let distance = BRepExtremaDistShapeShape::with_shapes(&target, shape, Default::default())
        .map_err(|cause| cause.to_string())?;
    let point = distance
        .point_on_shape2(1)
        .map_err(|cause| cause.to_string())?;
    Ok([point.x(), point.y()])
}
