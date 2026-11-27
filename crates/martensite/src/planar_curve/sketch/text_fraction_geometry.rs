// SPDX-License-Identifier: GPL-2.0-only
//! Lay out stacked fraction boxes with Martensite's measured analytic glyphs.
use super::{AlignX, AlignY, native_string, sketch_font};
use martensite_tkbrep::{
    brep::BRepBuilder,
    topods::{Compound, Shape},
};
use martensite_tkmath::gp::Point3;
use martensite_tktopalgo::brep_builder_api::{BRepBuilderAPIMakeFace, BRepBuilderAPIMakePolygon};
use martensite_tkv3d::{
    brep_font::BRepFontBuilderTextPlan as Plan,
    std_prs::{StdPrsBRepFont as Font, StdPrsBRepFontTextOptions as Options},
};
use oblikovati_api::types::{SketchTextFraction, SketchTextStyle};

// Text composition ratios. These define the visible form and do not set a kernel tolerance.
const FRACTION_SCALE: f64 = 0.62;
const FRACTION_PADDING: f64 = 0.10;
const FRACTION_BAR_THICKNESS: f64 = 0.055;
const FRACTION_AXIS_HEIGHT: f64 = 0.42;

pub(super) fn render(
    font: &Font,
    options: &Options,
    style: &SketchTextStyle,
    ranges: &[SketchTextFraction],
    advance: f64,
    width: f32,
) -> Result<Shape, String> {
    let small = sketch_font(style, font.size() * FRACTION_SCALE, width)?;
    let input = fraction_lines(&style.content, ranges)?;
    let builder = BRepBuilder::default();
    let mut geometry = Compound::default();
    builder.make_compound(&mut geometry);
    let drawing = FractionDrawing {
        font,
        small: &small,
        builder: &builder,
        horizontal_padding: font.size() * FRACTION_PADDING * f64::from(width),
    };
    for (index, row) in input.iter().enumerate() {
        let row = drawing.measure(row)?;
        drawing.place_row(
            &mut geometry,
            &row,
            options.horizontal_alignment,
            -(index as f64) * advance,
        )?;
    }
    Ok(geometry.into_shape())
}

enum Run<'a> {
    Text(&'a str),
    Fraction(&'a str, &'a str),
}

fn fraction_lines<'a>(
    content: &'a str,
    ranges: &[SketchTextFraction],
) -> Result<Vec<Vec<Run<'a>>>, String> {
    let offsets: Vec<_> = content
        .char_indices()
        .map(|(offset, _)| offset)
        .chain(std::iter::once(content.len()))
        .collect();
    let mut lines = vec![Vec::new()];
    let mut from = 0;
    for range in ranges {
        let first = offsets[range.start as usize];
        let last = offsets[range.end as usize];
        push_literal(&mut lines, &content[from..first]);
        let (top, bottom) = content[first..last]
            .split_once('/')
            .ok_or_else(|| format!("fraction {range:?}: expected a slash separator"))?;
        lines.last_mut().unwrap().push(Run::Fraction(top, bottom));
        from = last;
    }
    push_literal(&mut lines, &content[from..]);
    Ok(lines)
}

fn push_literal<'a>(lines: &mut Vec<Vec<Run<'a>>>, literal: &'a str) {
    for (index, part) in literal.split('\n').enumerate() {
        if index != 0 {
            lines.push(Vec::new());
        }
        if !part.is_empty() {
            lines.last_mut().unwrap().push(Run::Text(part));
        }
    }
}

enum Measured {
    Literal(Plan),
    Fraction { top: Plan, bottom: Plan, width: f64 },
}
impl Measured {
    fn width(&self) -> f64 {
        match self {
            Self::Literal(plan) => plan.width(),
            Self::Fraction { width, .. } => *width,
        }
    }
}

struct FractionDrawing<'a> {
    font: &'a Font,
    small: &'a Font,
    builder: &'a BRepBuilder,
    horizontal_padding: f64,
}

impl FractionDrawing<'_> {
    fn measure(&self, row: &[Run<'_>]) -> Result<Vec<Measured>, String> {
        row.iter()
            .map(|run| match run {
                Run::Text(text) => plan(self.font, text).map(Measured::Literal),
                Run::Fraction(top, bottom) => {
                    let (top, bottom) = (plan(self.small, top)?, plan(self.small, bottom)?);
                    let width = top.width().max(bottom.width()) + 2. * self.horizontal_padding;
                    Ok(Measured::Fraction { top, bottom, width })
                }
            })
            .collect()
    }

    fn place_row(
        &self,
        geometry: &mut Compound,
        row: &[Measured],
        align: AlignX,
        baseline: f64,
    ) -> Result<(), String> {
        let width: f64 = row.iter().map(Measured::width).sum();
        let mut x = match align {
            AlignX::Left => 0.,
            AlignX::Center => -width * 0.5,
            AlignX::Right => -width,
        };
        for item in row {
            match item {
                Measured::Literal(run) => self.add(geometry, self.font, run, [x, baseline])?,
                Measured::Fraction { top, bottom, width } => {
                    self.place_fraction(geometry, top, bottom, *width, [x, baseline])?;
                }
            }
            x += item.width();
        }
        Ok(())
    }

    fn place_fraction(
        &self,
        geometry: &mut Compound,
        top: &Plan,
        bottom: &Plan,
        width: f64,
        [x, y]: [f64; 2],
    ) -> Result<(), String> {
        let size = self.font.size();
        let bar = y + self.font.ascender() * FRACTION_AXIS_HEIGHT;
        let gap = size * FRACTION_PADDING;
        let upper = [
            x + (width - top.width()) * 0.5,
            bar + gap + self.small.descender().abs(),
        ];
        let lower = [
            x + (width - bottom.width()) * 0.5,
            bar - gap - self.small.ascender(),
        ];
        self.add(geometry, self.small, top, upper)?;
        self.add(geometry, self.small, bottom, lower)?;
        let rule = fraction_bar(x, bar, width, size * FRACTION_BAR_THICKNESS)?;
        self.builder
            .add(geometry, &rule)
            .map_err(|error| error.to_string())
    }

    fn add(
        &self,
        geometry: &mut Compound,
        font: &Font,
        plan: &Plan,
        [x, y]: [f64; 2],
    ) -> Result<(), String> {
        if plan.items().is_empty() {
            return Ok(());
        }
        let mut options = Options::default();
        options.pen.set_location(Point3::new(x, y, 0.));
        let shape = font.render_plan_with(plan, &options);
        if shape.is_null() {
            return Err(format!(
                "fraction layout at [{x}, {y}]: expected native glyph outlines"
            ));
        }
        self.builder
            .add(geometry, &shape)
            .map_err(|error| error.to_string())
    }
}

fn plan(font: &Font, content: &str) -> Result<Plan, String> {
    font.plan_text(&native_string(content)?, AlignX::Left, AlignY::Topfirstline)
        .ok_or_else(|| format!("fraction layout for {content:?}: expected a native text plan"))
}

fn fraction_bar(x: f64, y: f64, width: f64, thickness: f64) -> Result<Shape, String> {
    let y_bottom = y - thickness * 0.5;
    let y_top = y + thickness * 0.5;
    let mut polygon = BRepBuilderAPIMakePolygon::new();
    for [px, py] in [
        [x, y_bottom],
        [x + width, y_bottom],
        [x + width, y_top],
        [x, y_top],
    ] {
        polygon
            .add_point(Point3::new(px, py, 0.))
            .map_err(|error| error.to_string())?;
    }
    polygon.close().map_err(|error| error.to_string())?;
    let wire = polygon.wire().map_err(|error| error.to_string())?;
    BRepBuilderAPIMakeFace::from_wire(&wire, true)
        .and_then(|face| face.face())
        .map(Into::into)
        .map_err(|error| error.to_string())
}
