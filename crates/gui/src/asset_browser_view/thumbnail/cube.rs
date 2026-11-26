// SPDX-License-Identifier: GPL-2.0-only
//! Orthographic cube sample with face-local material coordinates.
use super::{PreviewLight, sampled_surface::PreviewSurface};
use crate::GuiAssetChoice;
use imgui::DrawListMut;

struct Face {
    corners: [[f32; 2]; 4],
    facing: f32,
    shade: f32,
}

const FACES: [Face; 3] = [
    Face {
        corners: [[-0.84, -0.51], [0.18, -0.93], [0.86, -0.58], [-0.17, -0.13]],
        facing: 0.37,
        shade: 1.02,
    },
    Face {
        corners: [[-0.84, -0.51], [-0.17, -0.13], [-0.17, 0.90], [-0.84, 0.52]],
        facing: 0.54,
        shade: 0.58,
    },
    Face {
        corners: [[-0.17, -0.13], [0.86, -0.58], [0.86, 0.45], [-0.17, 0.90]],
        facing: 0.76,
        shade: 0.78,
    },
];

pub(crate) fn paint_cube_preview(
    draw: &DrawListMut<'_>,
    center: [f32; 2],
    radius: f32,
    asset: &GuiAssetChoice,
    light: PreviewLight,
) {
    let Some(finish) = asset.appearance_preview else {
        return;
    };
    let material = PreviewSurface::from_asset(asset);
    let gloss = 1. - f32::from(finish.roughness) / 255.;
    let highlight = f32::from(finish.highlights) / 255.;
    for face in &FACES {
        let paint = |[u, v]: [f32; 2]| {
            let facing = face.facing;
            let metallic = 1. + f32::from(finish.metalness) / 255. * facing * 0.22;
            let key = [0.5 + light.offset[0] * 0.7, 0.5 + light.offset[1] * 0.7];
            let distance = (u - key[0]).powi(2) + (v - key[1]).powi(2);
            let shine = (1. - distance * (4. + gloss * 21.)).max(0.).powi(2)
                * highlight
                * gloss
                * face.shade
                * 0.36;
            let base = material.color([2. * u - 1., 2. * v - 1.], face.shade * metallic, facing);
            light.illuminate(base.map(|value| value + (1. - value) * shine))
        };
        paint_face(draw, center, radius, face, paint);
    }
}

fn paint_face(
    draw: &DrawListMut<'_>,
    center: [f32; 2],
    radius: f32,
    face: &Face,
    paint: impl Fn([f32; 2]) -> [f32; 3],
) {
    // Paint the full face first to cover the antialiased joins between samples.
    quad(
        draw,
        face.corners.map(|at| project(center, radius, at)),
        paint([0.5; 2]),
    );
    const CELLS: u8 = 14;
    let step = 1. / f32::from(CELLS);
    for y in 0..CELLS {
        for x in 0..CELLS {
            let [u, v] = [f32::from(x) * step, f32::from(y) * step];
            let corners = [[u, v], [u + step, v], [u + step, v + step], [u, v + step]];
            let points = corners.map(|uv| project(center, radius, face.point(uv)));
            quad(draw, points, paint([u + step / 2., v + step / 2.]));
        }
    }
}

fn quad(draw: &DrawListMut<'_>, points: [[f32; 2]; 4], [r, g, b]: [f32; 3]) {
    draw.add_polyline(points.to_vec(), [r, g, b, 1.])
        .filled(true)
        .build();
}

fn project(center: [f32; 2], radius: f32, point: [f32; 2]) -> [f32; 2] {
    std::array::from_fn(|axis| center[axis] + radius * point[axis])
}

impl Face {
    fn point(&self, [u, v]: [f32; 2]) -> [f32; 2] {
        let [a, b, c, d] = self.corners;
        std::array::from_fn(|i| {
            let top = a[i] + (b[i] - a[i]) * u;
            let bottom = d[i] + (c[i] - d[i]) * u;
            top + (bottom - top) * v
        })
    }
}
