// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{
    OwnedSketchGeometry, SketchArc2D, SketchPointInput, SketchReadIndex,
};

type Point = [f64; 2];
type Segment = [Point; 2];

pub(super) fn single(
    index: &SketchReadIndex<'_>,
    id: SketchEntityId,
    distance: f64,
) -> Result<(SketchPrimitiveCreation, &'static str), Error> {
    let geometry = index.entity(id).map_err(api_error)?;
    Ok(match geometry {
        OwnedSketchGeometry::Line(line) => {
            let endpoints = offset_line(positions(index, line.endpoints)?, distance)?;
            (
                SketchPrimitiveCreation::Line {
                    endpoints: endpoints.map(SketchPointInput::New),
                    construction: false,
                    centerline: false,
                },
                "line",
            )
        }
        OwnedSketchGeometry::Circle(circle) => (
            SketchPrimitiveCreation::Circle {
                center: SketchPointInput::New(point(index, circle.center)?),
                radius: require_radius(circle.radius + distance)?,
                construction: false,
            },
            "circle",
        ),
        OwnedSketchGeometry::Arc(arc) => (offset_arc(index, arc, distance)?, "arc"),
        _ => {
            return Err(Error(format!(
                "sketch offset entity {}: expected a line, circle, or arc",
                id.0
            )));
        }
    })
}

fn offset_arc(
    index: &SketchReadIndex<'_>,
    arc: SketchArc2D,
    distance: f64,
) -> Result<SketchPrimitiveCreation, Error> {
    let center = point(index, arc.center)?;
    let endpoints = positions(index, [arc.start, arc.end])?;
    let radial = endpoints.map(|end| difference(end, center));
    let radius = radial[0][0].hypot(radial[0][1]);
    require_radius(radius)?;
    let scale = require_radius(radius + distance)? / radius;
    Ok(SketchPrimitiveCreation::Arc {
        center: SketchPointInput::New(center),
        endpoints: radial.map(|vector| {
            SketchPointInput::New(std::array::from_fn(|axis| {
                center[axis] + scale * vector[axis]
            }))
        }),
        counter_clockwise: arc.counter_clockwise,
        construction: false,
    })
}

pub(super) fn chain(
    index: &SketchReadIndex<'_>,
    ids: &[u64],
    distance: f64,
) -> Result<SketchPolylineCreation, Error> {
    let mut lines = ids
        .iter()
        .map(
            |&id| match index.entity(SketchEntityId(id)).map_err(api_error)? {
                OwnedSketchGeometry::Line(line) => positions(index, line.endpoints),
                _ => Err(Error(format!(
                    "sketch offset chain entity {id}: expected a line"
                ))),
            },
        )
        .collect::<Result<Vec<_>, _>>()?;
    orient_chain(&mut lines, ids)?;
    let closed = lines
        .first()
        .zip(lines.last())
        .is_some_and(|(first, last)| first[0] == last[1]);
    let offset = lines
        .into_iter()
        .map(|line| offset_line(line, distance))
        .collect::<Result<Vec<_>, _>>()?;
    let points = miter_vertices(&offset, closed)?;
    Ok(SketchPolylineCreation {
        points,
        closed,
        construction: false,
    })
}

fn orient_chain(lines: &mut [Segment], ids: &[u64]) -> Result<(), Error> {
    if let [first, second, ..] = lines
        && !second.contains(&first[1])
        && second.contains(&first[0])
    {
        first.reverse();
    }
    for i in 1..lines.len() {
        let previous = lines[i - 1][1];
        if lines[i][1] == previous {
            lines[i].reverse();
        }
        if lines[i][0] != previous {
            return Err(Error(format!(
                "sketch offset chain {:?}: expected endpoint continuity at entity {}",
                ids, ids[i]
            )));
        }
    }
    Ok(())
}

fn miter_vertices(lines: &[Segment], closed: bool) -> Result<Vec<Point>, Error> {
    let first = lines
        .first()
        .ok_or_else(|| Error("sketch offset chain: expected at least one line".into()))?;
    let last = lines
        .last()
        .ok_or_else(|| Error("sketch offset chain: expected at least one line".into()))?;
    let start = if closed {
        miter(*last, *first)?
    } else {
        first[0]
    };
    let mut points = vec![start];
    for pair in lines.windows(2) {
        points.push(miter(pair[0], pair[1])?);
    }
    if !closed {
        points.push(last[1]);
    }
    Ok(points)
}

fn miter(before: Segment, after: Segment) -> Result<Point, Error> {
    if before[1] == after[0] {
        return Ok(after[0]);
    }
    let a = difference(before[1], before[0]);
    let b = difference(after[1], after[0]);
    let determinant = cross(a, b);
    if determinant == 0. {
        return Err(Error(format!(
            "sketch offset join {before:?}, {after:?}: expected intersecting supports"
        )));
    }
    let t = cross(difference(after[0], before[0]), b) / determinant;
    Ok(std::array::from_fn(|axis| before[0][axis] + t * a[axis]))
}

fn offset_line(line: Segment, distance: f64) -> Result<Segment, Error> {
    let [dx, dy] = difference(line[1], line[0]);
    let length = dx.hypot(dy);
    if !(length.is_finite() && length > 0.) {
        return Err(Error(format!(
            "sketch offset line {line:?}: expected distinct finite points"
        )));
    }
    let normal = [-dy / length, dx / length];
    Ok(line.map(|point| std::array::from_fn(|axis| point[axis] + distance * normal[axis])))
}

fn require_radius(radius: f64) -> Result<f64, Error> {
    if radius.is_finite() && radius > 0. {
        Ok(radius)
    } else {
        Err(Error(format!(
            "sketch offset radius {radius}: expected a positive finite radius"
        )))
    }
}

fn positions(index: &SketchReadIndex<'_>, [a, b]: [SketchEntityId; 2]) -> Result<Segment, Error> {
    Ok([point(index, a)?, point(index, b)?])
}

fn point(index: &SketchReadIndex<'_>, id: SketchEntityId) -> Result<Point, Error> {
    Ok(index.point(id).map_err(api_error)?.position)
}

fn difference(a: Point, b: Point) -> Point {
    [a[0] - b[0], a[1] - b[1]]
}

fn cross(a: Point, b: Point) -> f64 {
    a[0] * b[1] - a[1] * b[0]
}
