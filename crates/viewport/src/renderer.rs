// SPDX-License-Identifier: GPL-2.0-only
use crate::{ViewportCamera, ViewportScene};
use martensite_vulkan::{Graphic3dRenderDriver, Graphic3dRenderView};
use rustycad_tkernel::{
    quantity::{Color, ColorSpace},
    standard::Handle,
};
use rustycad_tkmath::gp::{Direction3, Point3};
use rustycad_tkservice::{
    aspect::AspectWindow,
    graphic3d::{
        Graphic3dCamera, Graphic3dCameraProjection, Graphic3dStructure, Graphic3dStructureNetwork,
        Graphic3dStructureSource,
    },
};
use rustycad_tkv3d::v3d::{V3dView, V3dViewer};
#[path = "renderer/navigation.rs"]
mod navigation;
#[path = "renderer/status_text.rs"]
mod status_text;
pub(crate) struct SavedViewCamera {
    pub state: ViewportCamera,
    members: Graphic3dCamera,
}
pub(crate) struct RenderView {
    pub viewer: Handle<V3dViewer>,
    pub view: Handle<V3dView>,
    pub renderer: Handle<Graphic3dRenderView>,
    structures: Vec<Handle<Graphic3dStructure>>,
    status: status_text::StatusTextOverlay,
}
impl RenderView {
    pub fn new(
        driver: &Handle<Graphic3dRenderDriver>,
        window: &Handle<dyn AspectWindow>,
        extent: [u32; 2],
    ) -> Result<Self, String> {
        let viewer = V3dViewer::new(&driver.cast()).map_err(|error| error.to_string())?;
        let view = viewer.create_view().map_err(|error| error.to_string())?;
        view.set_immediate_update(false);
        let renderer = view
            .get()
            .ok_or("viewport creation: expected a non-null kernel view")?
            .view()
            .cast();
        let mut result = Self {
            viewer,
            view,
            renderer,
            structures: Vec::new(),
            status: Default::default(),
        };
        result.configure(window)?;
        result.camera(ViewportCamera::default(), extent)?;
        Ok(result)
    }
    fn configure(&self, window: &Handle<dyn AspectWindow>) -> Result<(), String> {
        self.view
            .set_window(window, 0)
            .map_err(|error| error.to_string())?;
        self.view
            .set_auto_z_fit_mode(false, 1.)
            .map_err(|error| error.to_string())
    }
    pub fn camera(&mut self, input: ViewportCamera, extent: [u32; 2]) -> Result<(), String> {
        input.validate()?;
        let camera = Handle::new(Graphic3dCamera::default());
        configure_orientation(&camera, input)?;
        configure_projection(&camera, input, extent)?;
        self.view
            .set_camera(&camera)
            .map_err(|error| error.to_string())
    }
    pub fn submit(&mut self, scene: &ViewportScene) -> Result<(), String> {
        scene.validate()?;
        let structures = crate::scene::build_scene(&self.viewer, &scene.triangles)?;
        let color = Color::new(
            scene.background[0],
            scene.background[1],
            scene.background[2],
            ColorSpace::Rgb,
        )
        .map_err(|error| error.to_string())?;
        for old in self.structures.drain(..) {
            old.remove();
        }
        self.view.set_background_color(color);
        for structure in &structures {
            structure.display();
        }
        self.structures = structures;
        Ok(())
    }
    pub(crate) fn saved_camera(&self) -> SavedViewCamera {
        SavedViewCamera {
            state: self.camera_state(),
            members: self.view.camera().copy_members(),
        }
    }
    pub(crate) fn restore_camera(
        &self,
        saved: &SavedViewCamera,
        extent: [u32; 2],
    ) -> Result<(), String> {
        saved.state.validate()?;
        let camera = Handle::new(saved.members.copy_members());
        camera.set_aspect(f64::from(extent[0]) / f64::from(extent[1]));
        self.view
            .set_camera(&camera)
            .map_err(|cause| cause.to_string())
    }
    pub fn resized(&self, extent: [u32; 2]) {
        self.view
            .camera()
            .set_aspect(f64::from(extent[0]) / f64::from(extent[1]));
        self.view.must_be_resized();
    }
}
impl Drop for RenderView {
    fn drop(&mut self) {
        for structure in self.structures.drain(..) {
            structure.remove();
        }
        self.status.clear();
        self.view.remove();
    }
}

fn configure_orientation(
    camera: &Handle<Graphic3dCamera>,
    input: ViewportCamera,
) -> Result<(), String> {
    let (direction, up) = camera_directions(input)?;
    let [x, y, z] = input.eye;
    let eye = Point3::new(x, y, z);
    let [x, y, z] = input.target;
    let target = Point3::new(x, y, z);
    // v0.1.2 set_center ignores equal-distance target changes; update both points together.
    camera
        .set_eye_and_center(&eye, &target)
        .map_err(|error| error.to_string())?;
    camera.set_direction_from_eye(&direction);
    camera.set_up(&up);
    Ok(())
}
fn camera_directions(input: ViewportCamera) -> Result<(Direction3, Direction3), String> {
    let [x, y, z]: [f64; 3] = std::array::from_fn(|axis| input.target[axis] - input.eye[axis]);
    let direction = Direction3::new(x, y, z).map_err(|error| {
        format!("viewport camera {input:?}: expected usable view direction: {error}")
    })?;
    let [x, y, z] = input.up;
    let up = Direction3::new(x, y, z).map_err(|error| {
        format!("viewport camera {input:?}: expected usable up direction: {error}")
    })?;
    direction.crossed(up).map_err(|error| {
        format!("viewport camera {input:?}: expected independent viewing/up directions: {error}")
    })?;
    Ok((direction, up))
}
fn configure_projection(
    camera: &Handle<Graphic3dCamera>,
    input: ViewportCamera,
    extent: [u32; 2],
) -> Result<(), String> {
    camera.set_projection_type(Graphic3dCameraProjection::Orthographic);
    camera.set_scale(input.scale);
    camera.set_aspect(f64::from(extent[0]) / f64::from(extent[1]));
    camera
        .set_z_range(input.near, input.far)
        .map_err(|error| error.to_string())
}

impl RenderView {
    pub(crate) fn status_text(&mut self, text: &str) -> Result<bool, String> {
        self.status.submit(&self.viewer, text)
    }
}
