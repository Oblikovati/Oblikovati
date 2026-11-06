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
            Ok(pixel) => labels.push(GuiDimensionLabel {
                pixel,
                text: annotation.text,
                role: annotation.role,
                edit: annotation.edit,
            }),
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
/// A half-side of the displayed origin planes, in working lengths: readable at the default view.
const ORIGIN_PLANE_REACH: f64 = 6.;
/// The planes as the camera sees them; a window that cannot project shows none.
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
