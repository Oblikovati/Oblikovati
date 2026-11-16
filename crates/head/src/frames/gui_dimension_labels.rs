// SPDX-License-Identifier: GPL-2.0-only
use super::{HeadWindow, HeadWork};
use oblikovati_gui::GuiDimensionLabel;

pub(super) fn projected_dimension_labels(
    window: &mut dyn HeadWindow,
    owner: &dyn HeadWork,
) -> Result<Vec<GuiDimensionLabel>, String> {
    let annotations = owner.sketch_dimension_annotations()?;
    let mut declines = annotations.declines;
    let mut labels = Vec::new();
    for annotation in annotations.labels {
        match window.sketch_pixel(annotation.plane, annotation.position) {
            Ok(pixel) => {
                let line = match annotation
                    .line
                    .map(|[from, to]| {
                        Ok::<_, String>([
                            window.sketch_pixel(annotation.plane, from)?,
                            window.sketch_pixel(annotation.plane, to)?,
                        ])
                    })
                    .transpose()
                {
                    Ok(line) => line,
                    Err(cause) => {
                        declines.push(format!("dimension line {:?}: {cause}", annotation.text));
                        None
                    }
                };
                labels.push(GuiDimensionLabel {
                    pixel,
                    line,
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
    Ok(labels)
}
/// Marks the camera cannot place are left out: a mark is a reading aid, never a modelling input.
pub(super) fn projected_constraint_glyphs(
    window: &dyn HeadWindow,
    owner: &dyn HeadWork,
) -> Vec<oblikovati_gui::GuiConstraintGlyph> {
    owner
        .sketch_constraint_marks()
        .into_iter()
        .filter_map(|mark| {
            let pixel = window.sketch_pixel(mark.plane, mark.at).ok()?;
            Some(oblikovati_gui::GuiConstraintGlyph {
                sketch: mark.sketch,
                constraint: mark.constraint,
                related: mark.related,
                pixel,
                text: mark.text,
            })
        })
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
/// A half-side of the displayed origin planes, in working lengths: readable at the default view.
const ORIGIN_PLANE_REACH: f64 = 6.;
/// The planes as the camera sees them; a window that cannot project shows none.
/// The origin's axes in pixels, with the rows that name them.
pub(super) fn origin_axis_lines(
    window: &dyn HeadWindow,
    commands: &oblikovati_gui::GuiCommandState,
) -> Vec<oblikovati_gui::GuiOriginAxis> {
    if commands.document.is_none()
        || commands.authoring != oblikovati_gui::GuiAuthoringContext::Part
    {
        return Vec::new();
    }
    const R: f64 = ORIGIN_PLANE_REACH;
    [
        ("X Axis", [[-R, 0., 0.], [R, 0., 0.]]),
        ("Y Axis", [[0., -R, 0.], [0., R, 0.]]),
        ("Z Axis", [[0., 0., -R], [0., 0., R]]),
    ]
    .into_iter()
    .filter_map(|(label, [from, to])| {
        Some(oblikovati_gui::GuiOriginAxis {
            label,
            ends: [window.model_pixel(from).ok()?, window.model_pixel(to).ok()?],
        })
    })
    .collect()
}
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
    let planes = [
        (Xy, SketchPlane::xy()),
        (Xz, SketchPlane::xz()),
        (Yz, SketchPlane::yz()),
    ];
    planes
        .into_iter()
        .filter_map(|(kind, plane)| {
            let [a, b, c, d] =
                [[-1., -1.], [1., -1.], [1., 1.], [-1., 1.]].map(|[u, v]: [f64; 2]| {
                    window.sketch_pixel(plane, [u * ORIGIN_PLANE_REACH, v * ORIGIN_PLANE_REACH])
                });
            Some(oblikovati_gui::GuiOriginPlane {
                kind,
                corners: [a.ok()?, b.ok()?, c.ok()?, d.ok()?],
            })
        })
        .collect()
}
