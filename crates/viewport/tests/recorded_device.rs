// SPDX-License-Identifier: GPL-2.0-only
//! Named renderer I/O fake used by viewport state-transition tests.
use super::*;
use martensite_tkernel::standard::WeakHandle;
use martensite_tkservice::graphic3d::Graphic3dCView;
use martensite_vulkan::{
    FrameCapture, Graphic3dRenderView, RenderBackend, RenderError, RenderFrame,
    RenderGuiSurfacePolicy, RenderPipelinePolicies,
};
use std::sync::{Arc, Mutex};
#[derive(Default)]
pub(super) struct RecordedDevice {
    pub(super) frames: Vec<RenderFrame>,
    pub(super) fail: bool,
    pub(super) fail_after_resize: bool,
    pub(super) resize_once: Option<ResizeDuringDraw>,
    pub(super) revision_once: Option<WeakHandle<Graphic3dRenderView>>,
    pub(super) replace_gui_once: Option<RecordedGuiReplacement>,
    pub(super) clear_gui_once: Option<WeakHandle<Graphic3dRenderView>>,
    pub(super) reported_extent: Option<[u32; 2]>,
}
pub(super) struct ResizeDuringDraw {
    pub(super) window: Handle<AspectNeutralWindow>,
    pub(super) extent: [i32; 2],
}
pub(super) struct RecordedGuiReplacement {
    pub(super) target: WeakHandle<Graphic3dRenderView>,
    pub(super) layer: martensite_vulkan::RenderLayer,
    pub(super) extent: [u32; 2],
}
#[derive(Clone, Default)]
pub(super) struct RecordingDevice {
    pub(super) state: Arc<Mutex<RecordedDevice>>,
}
impl RenderBackend for RecordingDevice {
    fn render(&mut self, frame: &RenderFrame) -> Result<FrameCapture, RenderError> {
        let mut state = self.state.lock().unwrap();
        state.begin_recorded_frame(frame)?;
        Ok(state.recorded_capture(frame))
    }
}
impl RecordedDevice {
    fn begin_recorded_frame(&mut self, frame: &RenderFrame) -> Result<(), RenderError> {
        self.frames.push(frame.clone());
        if self.fail {
            return Err(recorded_device_error());
        }
        self.deliver_recorded_surface_change();
        self.deliver_recorded_gui_change();
        if self.fail_after_resize {
            return Err(recorded_device_error());
        }
        Ok(())
    }
    fn deliver_recorded_surface_change(&mut self) {
        if let Some(resize) = self.resize_once.take() {
            resize.window.set_size(resize.extent[0], resize.extent[1]);
        }
        if let Some(target) = self.revision_once.take() {
            target.observe().resized();
        }
    }
    fn deliver_recorded_gui_change(&mut self) {
        if let Some(replacement) = self.replace_gui_once.take() {
            replacement
                .target
                .observe()
                .attach_gui_layer(replacement.extent, replacement.layer)
                .unwrap();
        }
        if let Some(target) = self.clear_gui_once.take() {
            target.observe().clear_gui_layer().unwrap();
        }
    }
    fn recorded_capture(&self, frame: &RenderFrame) -> FrameCapture {
        let [width, height] = self.reported_extent.unwrap_or(frame.extent());
        FrameCapture {
            width,
            height,
            rgba: vec![255; width as usize * height as usize * 4],
            depth: vec![1.; width as usize * height as usize],
        }
    }
}
fn recorded_device_error() -> RenderError {
    RenderError::rejected("recorded device fault", "a successful draw")
}
/// Preserve legacy fixtures; example: `let (runtime, recorder) = runtime();`.
pub(super) fn runtime() -> (ViewportRuntime, RecordingDevice) {
    runtime_with_gui_policy(Default::default())
}
/// Select the publication contract for a fixture; example: `runtime_with_gui_policy(RenderGuiSurfacePolicy::CheckedV2)`.
pub(super) fn runtime_with_gui_policy(
    policy: RenderGuiSurfacePolicy,
) -> (ViewportRuntime, RecordingDevice) {
    let recorder = RecordingDevice::default();
    let driver = recorded_driver(&recorder, policy);
    (
        ViewportRuntime {
            state: Rc::new(RuntimeState {
                driver,
                x11: None,
                next_view: Cell::new(1),
            }),
        },
        recorder,
    )
}
fn recorded_driver(
    recorder: &RecordingDevice,
    policy: RenderGuiSurfacePolicy,
) -> Handle<Graphic3dRenderDriver> {
    Graphic3dRenderDriver::with_backend_and_line_width_configuration(
        recorder.clone(),
        1024,
        Arc::new(overlay_tests::FixedOverlayFont),
        RenderPipelinePolicies {
            gui_surface: policy,
            ..Default::default()
        },
        Default::default(),
    )
    .unwrap()
}
