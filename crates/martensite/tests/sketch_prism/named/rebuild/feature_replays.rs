// SPDX-License-Identifier: GPL-2.0-only
//! Feature replay test submodules sharing the named rebuild harness.
#[allow(unused_imports)]
use super::*;
#[path = "boundary_patch_replay.rs"]
mod boundary_patch_replay;
#[path = "chamfer_v2.rs"]
mod chamfer_v2;
#[path = "combine_replay.rs"]
mod combine_replay;
#[path = "delete_face_replay.rs"]
mod delete_face_replay;
#[path = "direct_edit_replay.rs"]
mod direct_edit_replay;
#[path = "draft_replay.rs"]
mod draft_replay;
#[path = "extend_surface_replay.rs"]
mod extend_surface_replay;
#[path = "fillet_v2.rs"]
mod fillet_v2;
#[path = "finish_replay.rs"]
mod finish_replay;
#[path = "hole_frame_origin.rs"]
mod hole_frame_origin;
#[path = "hole_v2.rs"]
mod hole_v2;
#[path = "mark_replay.rs"]
mod mark_replay;
#[path = "replace_face_replay.rs"]
mod replace_face_replay;
#[path = "ruled_surface_replay.rs"]
mod ruled_surface_replay;
#[path = "shell_v2.rs"]
mod shell_v2;
#[path = "slot_replay.rs"]
mod slot_replay;
#[path = "split_replay.rs"]
mod split_replay;
#[path = "stitch_replay.rs"]
mod stitch_replay;
#[path = "surface_trim_replay.rs"]
mod surface_trim_replay;
#[path = "thicken_offset_replay.rs"]
mod thicken_offset_replay;
#[path = "thread_replay.rs"]
mod thread_replay;
