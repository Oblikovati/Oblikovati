// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn key(camera: u64, surface: u64) -> ProjectionKey {
    ProjectionKey {
        plane: SketchPlane::xy(),
        camera,
        surface,
    }
}

#[test]
fn exact_points_plane_camera_and_surface_are_required() {
    let mut cache = ProjectionCache::default();
    let points = [[0., 1.], [2., 3.]];
    let pixels = [[4, 5], [6, 7]];
    cache.store(key(1, 2), &points, &pixels);
    assert_eq!(cache.get(key(1, 2), &points), Some(pixels.to_vec()));
    assert!(cache.get(key(2, 2), &points).is_none());
    assert!(cache.get(key(1, 3), &points).is_none());
    assert!(
        cache
            .get(
                ProjectionKey {
                    plane: SketchPlane::xz(),
                    ..key(1, 2)
                },
                &points
            )
            .is_none()
    );
    assert!(cache.get(key(1, 2), &[[0., 1.], [2., 4.]]).is_none());
    assert!(cache.get(key(1, 2), &[[2., 3.], [0., 1.]]).is_none());
    assert!(cache.get(key(1, 2), &[[-0., 1.], [2., 3.]]).is_none());
}

#[test]
fn alternating_batches_remain_cached_and_eviction_is_bounded() {
    let mut cache = ProjectionCache::default();
    let a = [[0., 0.]];
    let b = [[1., 1.]];
    let c = [[2., 2.]];
    cache.store(key(1, 1), &a, &[[1, 1]]);
    cache.store(key(1, 1), &b, &[[2, 2]]);
    assert!(cache.get(key(1, 1), &a).is_some());
    cache.store(key(1, 1), &c, &[[3, 3]]);
    assert!(cache.get(key(1, 1), &a).is_some());
    assert!(cache.get(key(1, 1), &b).is_none());
    assert!(cache.get(key(1, 1), &c).is_some());
    assert_eq!(cache.entries.len(), 2);
}

#[test]
fn oversized_or_mismatched_results_are_not_cached() {
    let mut cache = ProjectionCache::default();
    let points = vec![[0., 0.]; MAX_POINTS + 1];
    let pixels = vec![[0, 0]; MAX_POINTS + 1];
    cache.store(key(0, 0), &points, &pixels);
    cache.store(key(0, 0), &[[0., 0.]], &[]);
    assert!(cache.entries.is_empty());
}

#[test]
fn clear_releases_retained_batches() {
    let mut cache = ProjectionCache::default();
    cache.store(key(1, 1), &[[0., 0.]], &[[1, 2]]);
    cache.clear();
    assert!(cache.entries.is_empty());
    assert!(cache.get(key(1, 1), &[[0., 0.]]).is_none());
}
