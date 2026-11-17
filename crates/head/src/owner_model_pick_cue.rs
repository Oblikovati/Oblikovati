// SPDX-License-Identifier: GPL-2.0-only
//! Transient cue tied to an exact selected face and document generation (#4048).
use super::*;
use crate::part_scene::pick_candidates::{PickCandidate, PickTarget};
use oblikovati_viewport::{ViewportEvent, ViewportInput};

pub(super) struct ModelPickCue {
    document: u64,
    generation: u64,
    feature: oblikovati_model::feature::FeatureId,
    key: Vec<u8>,
    position: [f64; 3],
}

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn face_cue_from_hit(
        &self,
        document: u64,
        line: [[f64; 3]; 2],
        hit: Option<&PickCandidate>,
    ) -> Result<Option<ModelPickCue>, String> {
        let Some(PickCandidate {
            target: PickTarget::Face(feature, key),
            line_parameter,
        }) = hit
        else {
            return Ok(None);
        };
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        Ok(Some(ModelPickCue {
            document,
            generation,
            feature: *feature,
            key: key.clone(),
            position: hit_position(line, *line_parameter)?,
        }))
    }
    pub(super) fn gui_model_pick_cue(&self) -> Option<oblikovati_gui::GuiModelPickCue> {
        let cue = self.model_pick_cue.as_ref()?;
        if self.files.workspace().active_document() != Some(cue.document)
            || self.active_sketch_edit().is_some()
            || !self
                .chosen_model_face(cue.document)
                .is_some_and(|face| face.feature == cue.feature && face.key == cue.key)
            || self
                .files
                .workspace()
                .session(cue.document)
                .ok()?
                .generation()
                != cue.generation
        {
            return None;
        }
        Some(oblikovati_gui::GuiModelPickCue {
            pixel: [0, 0],
            position: cue.position,
        })
    }
    pub(super) fn synchronize_model_pick_cue(&mut self) -> bool {
        if self.model_pick_cue.is_some() && self.gui_model_pick_cue().is_none() {
            self.model_pick_cue = None;
            return true;
        }
        false
    }
    pub(super) fn update_model_pick_cue(
        &mut self,
        inputs: &[ViewportEvent],
        window: &dyn crate::HeadWindow,
    ) -> bool {
        let Some(cue) = self.model_pick_cue.as_ref() else {
            return false;
        };
        let retire = inputs.iter().any(|event| match event {
            ViewportEvent::Input(ViewportInput::Motion { position, .. }) => window
                .model_pixel(cue.position)
                .map(|anchor| pointer_left(anchor, *position))
                .unwrap_or(true),
            ViewportEvent::Input(ViewportInput::Button {
                button: oblikovati_viewport::ViewportButton::Left,
                pressed: true,
                position,
                ..
            }) => window
                .model_pixel(cue.position)
                .map(|anchor| pointer_left(anchor, *position))
                .unwrap_or(true),
            ViewportEvent::Input(ViewportInput::Button { pressed: true, .. })
            | ViewportEvent::Input(ViewportInput::Wheel { .. })
            | ViewportEvent::Resized(_)
            | ViewportEvent::Focus(false)
            | ViewportEvent::Mapped(false) => true,
            _ => false,
        });
        if retire {
            self.model_pick_cue = None;
        }
        retire
    }
}

fn pointer_left(anchor: [i32; 2], pointer: [i32; 2]) -> bool {
    let dx = f64::from(anchor[0]) - f64::from(pointer[0]);
    let dy = f64::from(anchor[1]) - f64::from(pointer[1]);
    dx.hypot(dy) > 12.
}
fn hit_position([origin, direction]: [[f64; 3]; 2], distance: f64) -> Result<[f64; 3], String> {
    let length = direction[0].hypot(direction[1]).hypot(direction[2]);
    if !origin.into_iter().chain(direction).all(f64::is_finite)
        || !length.is_finite()
        || length == 0.
        || !distance.is_finite()
        || distance < 0.
    {
        return Err(format!(
            "face ray {origin:?}, direction {direction:?}, distance {distance}: expected finite origin, nonzero finite direction and nonnegative finite distance"
        ));
    }
    let position = std::array::from_fn(|axis| origin[axis] + direction[axis] / length * distance);
    if !position.into_iter().all(f64::is_finite) {
        return Err(format!(
            "face hit {position:?}: expected finite coordinates"
        ));
    }
    Ok(position)
}

#[cfg(test)]
mod tests {
    use super::{hit_position, pointer_left};
    #[test]
    fn face_hit_uses_normalized_ray_and_rejects_invalid_geometry() {
        assert_eq!(
            hit_position([[1., 2., 3.], [0., 0., 2.]], 4.).unwrap(),
            [1., 2., 7.]
        );
        for (line, distance) in [
            ([[0.; 3], [0.; 3]], 1.),
            ([[0.; 3], [1., 0., 0.]], -1.),
            ([[f64::NAN, 0., 0.], [1., 0., 0.]], 1.),
        ] {
            assert!(hit_position(line, distance).is_err());
        }
    }
    #[test]
    fn pointer_tolerance_uses_physical_pixels_without_integer_overflow() {
        assert!(!pointer_left([0, 0], [12, 0]));
        assert!(pointer_left([0, 0], [13, 0]));
        assert!(pointer_left([i32::MIN, 0], [i32::MAX, 0]));
    }
}
