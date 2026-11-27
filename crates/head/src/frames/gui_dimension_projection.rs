// SPDX-License-Identifier: GPL-2.0-only
//! Batch same-plane annotation points without losing local projection errors (#4035).
use crate::{HeadDimensionAnnotation, HeadWindow};

type Pixel = Result<[i32; 2], String>;
pub(super) struct AnnotationPixels {
    pub(super) position: Pixel,
    pub(super) line: Option<Result<[[i32; 2]; 2], String>>,
    pub(super) witnesses: Option<[[i32; 2]; 2]>,
}

/// Project against one stable drawable; example: `project_consistent(window, &annotations)?`.
pub(super) fn project_consistent(
    window: &mut dyn HeadWindow,
    annotations: &[HeadDimensionAnnotation],
) -> Result<(Vec<AnnotationPixels>, bool), String> {
    if annotations.is_empty() {
        return Ok((Vec::new(), false));
    }
    let projected = project(window, annotations);
    let failed = projected
        .iter()
        .any(|item| item.position.is_err() || item.line.as_ref().is_some_and(Result::is_err));
    if failed && window.synchronize_projection_extent()? {
        return Ok((Vec::new(), true));
    }
    Ok((projected, false))
}

pub(super) fn project(
    window: &dyn HeadWindow,
    annotations: &[HeadDimensionAnnotation],
) -> Vec<AnnotationPixels> {
    let mut output = Vec::with_capacity(annotations.len());
    let mut start = 0;
    while start < annotations.len() {
        let end = same_plane_end(annotations, start);
        output.extend(project_group(window, &annotations[start..end]));
        start = end;
    }
    output
}
fn same_plane_end(annotations: &[HeadDimensionAnnotation], start: usize) -> usize {
    let plane = annotations[start].plane;
    annotations[start..]
        .iter()
        .position(|item| !item.plane.same_recipe(plane))
        .map_or(annotations.len(), |offset| start + offset)
}
fn project_group(
    window: &dyn HeadWindow,
    group: &[HeadDimensionAnnotation],
) -> Vec<AnnotationPixels> {
    let points = annotation_points(group);
    let mut pixels = project_points(window, group[0].plane, &points).into_iter();
    group
        .iter()
        .map(|annotation| {
            let position = next_pixel(&mut pixels);
            let line = annotation.line.map(|_| next_segment(&mut pixels));
            let witnesses = annotation
                .witnesses
                .and_then(|_| next_segment(&mut pixels).ok());
            AnnotationPixels {
                position,
                line,
                witnesses,
            }
        })
        .collect()
}
fn annotation_points(group: &[HeadDimensionAnnotation]) -> Vec<[f64; 2]> {
    group
        .iter()
        .flat_map(|annotation| {
            std::iter::once(annotation.position)
                .chain(annotation.line.into_iter().flatten())
                .chain(annotation.witnesses.into_iter().flatten())
        })
        .collect()
}

fn project_points(
    window: &dyn HeadWindow,
    plane: oblikovati_model::sketch::SketchPlane,
    points: &[[f64; 2]],
) -> Vec<Pixel> {
    let failure = match window.sketch_pixel_results(plane, points) {
        Ok(pixels) if pixels.len() == points.len() => return pixels,
        Ok(pixels) => format!(
            "dimension projections {}: expected {} pixels",
            pixels.len(),
            points.len()
        ),
        Err(cause) => format!("dimension projections for {} points: {cause}", points.len()),
    };
    points.iter().map(|_| Err(failure.clone())).collect()
}

fn next_segment(pixels: &mut impl Iterator<Item = Pixel>) -> Result<[[i32; 2]; 2], String> {
    let from = next_pixel(pixels);
    let to = next_pixel(pixels);
    from.and_then(|from| to.map(|to| [from, to]))
}

fn next_pixel(pixels: &mut impl Iterator<Item = Pixel>) -> Pixel {
    pixels
        .next()
        .unwrap_or_else(|| Err("missing dimension projection: expected one pixel per point".into()))
}

#[cfg(test)]
#[path = "../../tests/gui_dimension_projection.rs"]
mod tests;
