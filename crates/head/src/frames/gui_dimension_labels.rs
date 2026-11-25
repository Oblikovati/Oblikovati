// SPDX-License-Identifier: GPL-2.0-only
use super::{HeadWindow, HeadWork};
use oblikovati_gui::GuiDimensionLabel;
#[path = "gui_dimension_projection.rs"]
mod dimension_projection;

/// Project the active sketch axes using the Go grid span (20 cells per side).
pub(super) fn projected_sketch_axes(
    window: &dyn HeadWindow,
    commands: &oblikovati_gui::GuiCommandState,
) -> Vec<[[i32; 2]; 2]> {
    let Some(options) = commands
        .sketch_options
        .as_ref()
        .filter(|options| options.axes_visible)
    else {
        return Vec::new();
    };
    let Some(plane) = commands.active_sketch_plane else {
        return Vec::new();
    };
    let half = options.grid_spacing_cm * 20.;
    if !half.is_finite() || half <= 0. {
        return Vec::new();
    }
    let points = [[-half, 0.], [half, 0.], [0., -half], [0., half]];
    match window.sketch_pixels(plane, &points) {
        Ok(pixels) if pixels.len() == 4 => vec![[pixels[0], pixels[1]], [pixels[2], pixels[3]]],
        _ => Vec::new(),
    }
}

pub(super) fn projected_dimension_labels(
    window: &mut dyn HeadWindow,
    owner: &dyn HeadWork,
) -> Result<(Vec<GuiDimensionLabel>, bool), String> {
    let annotations = owner.sketch_dimension_annotations()?;
    let mut declines = annotations.declines;
    let mut labels = Vec::new();
    let (projections, resized) =
        dimension_projection::project_consistent(window, &annotations.labels)?;
    for (annotation, projection) in annotations.labels.into_iter().zip(projections) {
        match projection.position {
            Ok(pixel) => {
                let line = match projection.line.transpose() {
                    Ok(line) => line,
                    Err(cause) => {
                        declines.push(format!("dimension line {:?}: {cause}", annotation.text));
                        None
                    }
                };
                labels.push(GuiDimensionLabel {
                    pixel,
                    line,
                    witnesses: projection.witnesses,
                    unit: annotation.unit.map(str::to_owned),
                    text: annotation.text,
                    role: annotation.role,
                    edit: annotation.edit,
                });
            }
            Err(cause) => declines.push(format!(
                "dimension label {:?} at {:?}: {cause}",
                annotation.text, annotation.position
            )),
        }
    }
    if !declines.is_empty() {
        window.show_command_notice(crate::DesktopCommandNotice::AuthoringFailed {
            cause: declines.join("; "),
        })?;
    }
    Ok((labels, resized))
}
/// Marks the camera cannot place are left out: a mark is a reading aid, never a modelling input.
pub(super) fn projected_constraint_glyphs(
    window: &dyn HeadWindow,
    owner: &dyn HeadWork,
) -> Vec<oblikovati_gui::GuiConstraintGlyph> {
    project_constraint_marks(window, owner.sketch_constraint_marks())
}
fn project_constraint_marks(
    window: &dyn HeadWindow,
    marks: Vec<crate::HeadConstraintMark>,
) -> Vec<oblikovati_gui::GuiConstraintGlyph> {
    let pixels = project_mark_pixels(window, &marks);
    marks
        .into_iter()
        .zip(pixels)
        .filter_map(|(mark, pixel)| {
            Some(oblikovati_gui::GuiConstraintGlyph {
                sketch: mark.sketch,
                constraint: mark.constraint,
                related: mark.related,
                pixel: pixel?,
                text: mark.text,
            })
        })
        .collect()
}
fn project_mark_pixels(
    window: &dyn HeadWindow,
    marks: &[crate::HeadConstraintMark],
) -> Vec<Option<[i32; 2]>> {
    let Some(first) = marks.first() else {
        return Vec::new();
    };
    if marks.iter().all(|mark| mark.plane.same_recipe(first.plane)) {
        let points: Vec<_> = marks.iter().map(|mark| mark.at).collect();
        if let Ok(pixels) = window.sketch_pixels(first.plane, &points)
            && pixels.len() == marks.len()
        {
            return pixels.into_iter().map(Some).collect();
        }
    }
    marks
        .iter()
        .map(|mark| window.sketch_pixel(mark.plane, mark.at).ok())
        .collect()
}
/// How far the triad's axes reach on screen, in pixels: a mark, not a measure.
const TRIAD_REACH: i32 = 28;
/// The chosen row's outline in pixels; a segment a window cannot project shows nothing.
pub(super) fn projected_browser_outline(
    window: &dyn HeadWindow,
    owner: &mut dyn HeadWork,
) -> Vec<[[i32; 2]; 2]> {
    owner
        .selected_browser_outline()
        .into_iter()
        .filter_map(|[from, to]| {
            Some([window.model_pixel(from).ok()?, window.model_pixel(to).ok()?])
        })
        .collect()
}
pub(super) fn projected_selected_edges(
    window: &dyn HeadWindow,
    owner: &mut dyn HeadWork,
) -> Vec<Vec<[[i32; 2]; 2]>> {
    owner
        .selected_edge_outlines()
        .into_iter()
        .map(|edge| {
            edge.into_iter()
                .filter_map(|[from, to]| {
                    Some([window.model_pixel(from).ok()?, window.model_pixel(to).ok()?])
                })
                .collect()
        })
        .collect()
}
/// The toolbar's triad at its sketch position; a window that cannot project shows none.
pub(super) fn project_precise_triad(
    window: &dyn HeadWindow,
    owner: &dyn HeadWork,
    commands: &mut oblikovati_gui::GuiCommandState,
) {
    let Some(target) = commands.precise_target.as_mut() else {
        return;
    };
    let Some((plane, at)) = owner.precise_triad_position() else {
        return;
    };
    target.triad = window.sketch_pixel(plane, at).ok().map(triad_pixels);
    target.origin_triad = window.sketch_pixel(plane, [0., 0.]).ok().map(triad_pixels);
}
fn triad_pixels(origin: [i32; 2]) -> [[i32; 2]; 3] {
    [
        origin,
        [origin[0] + TRIAD_REACH, origin[1]],
        [origin[0], origin[1] - TRIAD_REACH],
    ]
}
/// The fixed half-size of an origin datum, in document lengths.
const ORIGIN_PLANE_REACH: f64 = 6.;
/// Leave visual breathing room around the scene when a datum follows its bounds.
const ORIGIN_AUTO_MARGIN: f64 = 1.1;

fn automatic_origin_reach(bounds: Option<[[f64; 3]; 2]>) -> f64 {
    let Some(bounds) = bounds else {
        return ORIGIN_PLANE_REACH;
    };
    if bounds.iter().flatten().any(|value| !value.is_finite()) {
        return ORIGIN_PLANE_REACH;
    }
    let largest = bounds
        .into_iter()
        .flatten()
        .map(f64::abs)
        .fold(0., f64::max);
    let reach = largest * ORIGIN_AUTO_MARGIN;
    if reach.is_finite() {
        reach.max(ORIGIN_PLANE_REACH)
    } else {
        ORIGIN_PLANE_REACH
    }
}

/// Project the origin axes at fixed and scene-sized reaches.
pub(super) fn origin_axis_lines(
    window: &dyn HeadWindow,
    commands: &oblikovati_gui::GuiCommandState,
) -> Vec<oblikovati_gui::GuiOriginAxis> {
    if commands.document.is_none()
        || commands.authoring != oblikovati_gui::GuiAuthoringContext::Part
    {
        return Vec::new();
    }
    let automatic = automatic_origin_reach(window.presentation_bounds());
    ["X Axis", "Y Axis", "Z Axis"]
        .into_iter()
        .enumerate()
        .filter_map(|(axis, label)| {
            let ends = project_axis(window, axis, ORIGIN_PLANE_REACH)?;
            let automatic_ends = if automatic == ORIGIN_PLANE_REACH {
                ends
            } else {
                project_axis(window, axis, automatic)?
            };
            Some(oblikovati_gui::GuiOriginAxis {
                label,
                ends,
                automatic_ends,
            })
        })
        .collect()
}
fn project_axis(window: &dyn HeadWindow, axis: usize, reach: f64) -> Option<[[i32; 2]; 2]> {
    let mut first = [0.; 3];
    first[axis] = -reach;
    let mut last = [0.; 3];
    last[axis] = reach;
    Some([
        window.model_pixel(first).ok()?,
        window.model_pixel(last).ok()?,
    ])
}
/// Project the origin planes at fixed and scene-sized reaches.
pub(super) fn origin_plane_outlines(
    window: &dyn HeadWindow,
    commands: &oblikovati_gui::GuiCommandState,
) -> Vec<oblikovati_gui::GuiOriginPlane> {
    use oblikovati_gui::GuiOriginPlaneKind::{Xy, Xz, Yz};
    use oblikovati_model::sketch::SketchPlane;
    if commands.document.is_none()
        || commands.authoring != oblikovati_gui::GuiAuthoringContext::Part
    {
        return Vec::new();
    }
    let bounds = window.presentation_bounds();
    let automatic = automatic_origin_reach(bounds);
    [
        (Xy, SketchPlane::xy()),
        (Xz, SketchPlane::xz()),
        (Yz, SketchPlane::yz()),
    ]
    .into_iter()
    .filter_map(|(kind, plane)| {
        let corners = project_origin_plane(window, plane, ORIGIN_PLANE_REACH)?;
        let automatic_corners = if automatic == ORIGIN_PLANE_REACH {
            corners
        } else {
            project_origin_plane(window, plane, automatic)?
        };
        Some(oblikovati_gui::GuiOriginPlane {
            kind,
            corners,
            automatic_corners,
            picker_corners: picker_extent(plane, bounds)
                .and_then(|extent| project_plane_extent(window, plane, extent))
                .unwrap_or(corners),
        })
    })
    .collect()
}
fn project_origin_plane(
    window: &dyn HeadWindow,
    plane: oblikovati_model::sketch::SketchPlane,
    reach: f64,
) -> Option<[[i32; 2]; 4]> {
    project_plane_extent(window, plane, [[-reach; 2], [reach; 2]])
}

fn project_plane_extent(
    window: &dyn HeadWindow,
    plane: oblikovati_model::sketch::SketchPlane,
    [[left, bottom], [right, top]]: [[f64; 2]; 2],
) -> Option<[[i32; 2]; 4]> {
    let outline = [[left, bottom], [right, bottom], [right, top], [left, top]];
    window.sketch_pixels(plane, &outline).ok()?.try_into().ok()
}

fn picker_extent(
    plane: oblikovati_model::sketch::SketchPlane,
    bounds: Option<[[f64; 3]; 2]>,
) -> Option<[[f64; 2]; 2]> {
    let [low, high] = bounds?;
    // The three named origin planes use positive model axes for their local coordinates.
    let minimum = plane.to_sketch(low).ok()?;
    let maximum = plane.to_sketch(high).ok()?;
    let span: [f64; 2] = std::array::from_fn(|i| maximum[i] - minimum[i]);
    let longest = span[0].max(span[1]);
    if longest <= 0. || span.iter().any(|s| !s.is_finite() || *s < 0.) {
        return None;
    }
    let center: [f64; 2] = std::array::from_fn(|i| minimum[i] + span[i] * 0.5);
    let half = span.map(|width| width.max(longest * 0.15) * 0.625);
    Some([
        std::array::from_fn(|i| center[i] - half[i]),
        std::array::from_fn(|i| center[i] + half[i]),
    ])
}

#[cfg(test)]
#[path = "../../tests/constraint_glyph_projection.rs"]
mod tests;

#[cfg(test)]
#[path = "../../tests/origin_projection_tests.rs"]
mod origin_tests;
