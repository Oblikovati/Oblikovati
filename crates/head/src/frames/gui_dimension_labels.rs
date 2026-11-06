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
