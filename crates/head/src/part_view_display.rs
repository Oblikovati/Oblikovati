// SPDX-License-Identifier: GPL-2.0-only
//! Bridge part-view display transitions to the live viewport (#4118).
use oblikovati_api::types::{
    FRONT_VIEW_ORIENTATION, ISO_TOP_RIGHT_VIEW_ORIENTATION, PartDesignViewCamera,
    RIGHT_VIEW_ORIENTATION, TOP_VIEW_ORIENTATION, ViewOrientationTypeEnum,
};
use oblikovati_automation::PartViewDisplayAccess;
use oblikovati_gui::{GuiCubeFace, GuiOrientCommand};

pub(crate) struct WindowPartViewDisplay<'a>(pub(crate) &'a mut dyn crate::HeadWindow);
impl PartViewDisplayAccess for WindowPartViewDisplay<'_> {
    fn camera(&mut self) -> Result<PartDesignViewCamera, String> {
        Ok(self.0.view_camera()?.part_view_camera())
    }
    fn orient(
        &mut self,
        camera: &PartDesignViewCamera,
        orientation: ViewOrientationTypeEnum,
    ) -> Result<PartDesignViewCamera, String> {
        let camera = oblikovati_viewport::ViewportCamera::from_part_view_camera(camera)?;
        if orientation.0 == 0 {
            return Ok(camera.part_view_camera());
        }
        let command = if orientation == FRONT_VIEW_ORIENTATION {
            GuiOrientCommand::Face(GuiCubeFace::Front)
        } else if orientation == TOP_VIEW_ORIENTATION {
            GuiOrientCommand::Face(GuiCubeFace::Top)
        } else if orientation == RIGHT_VIEW_ORIENTATION {
            GuiOrientCommand::Face(GuiCubeFace::Right)
        } else if orientation == ISO_TOP_RIGHT_VIEW_ORIENTATION {
            return Ok(isometric(&camera).part_view_camera());
        } else {
            return Err(format!(
                "part view orientation {orientation:?}: expected a supported standard orientation"
            ));
        };
        Ok(crate::view_orientation::oriented(&camera, command).part_view_camera())
    }
    fn apply(&mut self, camera: &PartDesignViewCamera) -> Result<(), String> {
        let camera = oblikovati_viewport::ViewportCamera::from_part_view_camera(camera)?;
        self.0.set_view_camera(camera).map(|_| ())
    }
}
fn isometric(camera: &oblikovati_viewport::ViewportCamera) -> oblikovati_viewport::ViewportCamera {
    let front =
        crate::view_orientation::oriented(camera, GuiOrientCommand::Face(GuiCubeFace::Front));
    crate::view_orientation::oriented(
        &front,
        GuiOrientCommand::Corner {
            right: true,
            up: true,
        },
    )
}
