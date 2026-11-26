// SPDX-License-Identifier: GPL-2.0-only
use super::{Viewport, ViewportScene, next_generation};
use crate::scene::MeshDebug;

impl Viewport {
    /// Color the displayed model by face winding; example: `viewport.set_normal_debug(Some(&scene), true)?`.
    pub fn set_normal_debug(
        &mut self,
        scene: Option<&ViewportScene>,
        on: bool,
    ) -> Result<bool, String> {
        let debug = MeshDebug {
            normals: on,
            ..self.live()?.mesh_debug
        };
        self.set_mesh_debug(scene, debug)
    }

    /// Display the model's face or triangle palette; example: `viewport.set_mesh_colors(Some(&scene), true, false)?`.
    pub fn set_mesh_colors(
        &mut self,
        scene: Option<&ViewportScene>,
        on: bool,
        per_triangle: bool,
    ) -> Result<(), String> {
        if on
            && !per_triangle
            && scene.is_some_and(|scene| {
                scene
                    .triangles
                    .iter()
                    .any(|triangle| triangle.face_index.is_none())
            })
        {
            return Err("viewport face colors require a mesh with B-rep face indices; use triangle colors or load the live model".into());
        }
        let debug = MeshDebug {
            colors: on.then_some(per_triangle),
            ..self.live()?.mesh_debug
        };
        self.set_mesh_debug(scene, debug)?;
        Ok(())
    }

    fn set_mesh_debug(
        &mut self,
        scene: Option<&ViewportScene>,
        debug: MeshDebug,
    ) -> Result<bool, String> {
        if self.live()?.mesh_debug == debug {
            return Ok(false);
        }
        if let Some(scene) = scene {
            scene.validate()?;
        }
        let next = next_generation(self.scene)?;
        self.render
            .as_mut()
            .expect("live view")
            .set_mesh_debug(scene, debug)?;
        self.scene = next;
        self.completed = None;
        Ok(true)
    }
}
