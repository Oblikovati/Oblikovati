#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
#
# Blender oracle driver for M45-F05 PBI-353's OpenPBR perceptual goldens
# (architecture/testing/00-renderer-oracle-pipeline.md, Tier 4). Renders one
# unit sphere lit by one sun lamp through Blender 4.x's Principled BSDF v2 —
# used here as the INTERIM reference (this environment's Blender 4.0.2 has no
# native OpenPBR node yet, and neither MaterialX nor Substance is available to
# script headlessly) — Principled BSDF v2 was itself designed to track
# OpenPBR-adjacent conventions closely (Base Color/Metallic/Roughness/IOR,
# Coat Weight/Roughness/IOR, Sheen Weight/Roughness/Tint for OpenPBR's Fuzz,
# Subsurface Weight/Radius, Transmission Weight all map directly), so this is
# a faithful perceptual reference, not a stand-in with no relation to OpenPBR.
#
# Usage: blender -b -P render_reference.py -- <params.json> <output.png>
# params.json fields (all optional, defaulting to OpenPBR's own spec defaults):
#   base_color: [r,g,b] linear   metallic: float   roughness: float   ior: float
#   coat_weight, coat_roughness, coat_ior: float
#   sheen_weight, sheen_roughness: float   sheen_tint: [r,g,b]
#   subsurface_weight: float   subsurface_radius: [r,g,b]
#   transmission_weight: float
#   width, height: int (default 256)
import json
import math
import sys

import bpy
import mathutils

argv = sys.argv[sys.argv.index("--") + 1:]
params_path, out_path = argv[0], argv[1]
with open(params_path) as f:
    p = json.load(f)

width = p.get("width", 256)
height = p.get("height", 256)

# Fresh scene: Blender's factory startup scene has a default cube/light/camera we don't want.
bpy.ops.wm.read_factory_settings(use_empty=True)
scene = bpy.context.scene
scene.render.engine = "CYCLES"
scene.cycles.samples = 256  # converged enough for a perceptual (not bit-exact) comparison
scene.cycles.use_denoising = False  # this environment's Blender has no OpenImageDenoiser build
scene.render.resolution_x = width
scene.render.resolution_y = height
scene.render.resolution_percentage = 100
scene.render.image_settings.file_format = "PNG"
scene.render.image_settings.color_depth = "16"
# Raw/Standard view transform: no filmic tone-mapping, matching the architecture doc's
# "Blender filmic/tonemap off" requirement — we compare in the same linear-ish space our
# own ACEScg->display pipeline (kernel/shading/openpbr.ToDisplay) produces.
scene.view_settings.view_transform = "Standard"
scene.view_settings.look = "None"
scene.view_settings.exposure = 0
scene.view_settings.gamma = 1

bpy.ops.mesh.primitive_uv_sphere_add(radius=1.0, location=(0, 0, 0), segments=48, ring_count=24)
sphere = bpy.context.active_object
bpy.ops.object.shade_smooth()

mat = bpy.data.materials.new("openpbr_reference")
mat.use_nodes = True
bsdf = mat.node_tree.nodes["Principled BSDF"]


def set_input(name, value):
    if name in bsdf.inputs:
        bsdf.inputs[name].default_value = value


set_input("Base Color", (*p.get("base_color", [0.8, 0.8, 0.8]), 1.0))
set_input("Metallic", p.get("metallic", 0.0))
set_input("Roughness", p.get("roughness", 0.3))
set_input("IOR", p.get("ior", 1.5))
set_input("Coat Weight", p.get("coat_weight", 0.0))
set_input("Coat Roughness", p.get("coat_roughness", 0.0))
set_input("Coat IOR", p.get("coat_ior", 1.5))
set_input("Sheen Weight", p.get("sheen_weight", 0.0))
set_input("Sheen Roughness", p.get("sheen_roughness", 0.5))
sheen_tint = p.get("sheen_tint")
if sheen_tint:
    set_input("Sheen Tint", (*sheen_tint, 1.0))
set_input("Subsurface Weight", p.get("subsurface_weight", 0.0))
subsurface_radius = p.get("subsurface_radius")
if subsurface_radius:
    set_input("Subsurface Radius", tuple(subsurface_radius))
set_input("Transmission Weight", p.get("transmission_weight", 0.0))
sphere.data.materials.append(mat)

# One directional (sun) light — same direction convention as our own live tests
# (renderer.SceneLight.Direction: unit vector FROM the surface TOWARD the light).
light_data = bpy.data.lights.new("key", type="SUN")
light_data.energy = p.get("light_intensity", 3.0)
light_obj = bpy.data.objects.new("key", light_data)
scene.collection.objects.link(light_obj)
light_dir = p.get("light_direction", [0.3, 0.5, 0.8])
# A sun lamp shines along its local -Z; point +Z at light_dir so it shines toward the
# origin from that direction (i.e. -Z == -light_dir, the vector FROM the light TO the
# surface), matching "Direction is the unit vector from a lit surface toward the light".
z = mathutils.Vector(light_dir).normalized()
light_obj.rotation_euler = z.to_track_quat("Z", "Y").to_euler()

cam_data = bpy.data.cameras.new("cam")
cam_data.lens_unit = "FOV"
cam_data.angle_y = 2 * math.atan(p.get("tan_half_fov_y", 0.4))
cam_obj = bpy.data.objects.new("cam", cam_data)
cam_obj.location = tuple(p.get("eye", [0, -4, 0]))
scene.collection.objects.link(cam_obj)
scene.camera = cam_obj
look_at = mathutils.Vector((0, 0, 0)) - mathutils.Vector(cam_obj.location)
cam_obj.rotation_euler = look_at.to_track_quat("-Z", "Y").to_euler()

scene.render.filepath = out_path
bpy.ops.render.render(write_still=True)
print("RENDERED", out_path)
