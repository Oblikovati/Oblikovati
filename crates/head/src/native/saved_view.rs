// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::wire::CameraView;
use oblikovati_gui::GuiViewProjection;
use oblikovati_viewport::ViewportCamera;

pub(super) type SketchCameraState = (crate::HeadSketchEdit, Option<ViewportCamera>);

#[derive(Clone)]
pub(super) struct SavedWindowView {
    pub camera: ViewportCamera,
    pub projection: GuiViewProjection,
    pub authored: Option<CameraView>,
    pub sketch: Option<SketchCameraState>,
}

impl SavedWindowView {
    pub(super) fn initial(slot: u64) -> Self {
        let camera = ViewportCamera::default();
        let camera = if slot >= crate::FIRST_EXTRA_WINDOW {
            camera.isometric()
        } else {
            camera
        };
        Self {
            camera,
            projection: GuiViewProjection::default(),
            authored: None,
            sketch: None,
        }
    }

    pub(super) fn wire_frame(&self) -> CameraView {
        self.authored
            .clone()
            .unwrap_or_else(|| crate::camera_automation::wire_frame(self.camera, self.projection))
    }
}

pub(super) struct WireCameraStamp {
    pub camera: ViewportCamera,
    pub projection: GuiViewProjection,
    pub authored: CameraView,
}

impl WireCameraStamp {
    pub(super) fn matches(&self, camera: ViewportCamera, projection: GuiViewProjection) -> bool {
        // Depth fitting and resize may change the internal clipping planes without changing the wire frame.
        self.projection == projection
            && self.camera.eye == camera.eye
            && self.camera.target == camera.target
            && self.camera.up == camera.up
            && self.camera.scale == camera.scale
    }
}
