// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::identity::{DocumentGuid, RecipeIdAllocator, RecipeIdError};

#[test]
fn allocated_ids_do_not_depend_on_other_documents() {
    let mut first = RecipeIdAllocator::default();
    let mut second = RecipeIdAllocator::default();
    for expected in 1..=1024 {
        assert_eq!(first.allocate(), Ok(expected));
    }
    assert_eq!(second.allocate(), Ok(1));
    assert_eq!(first.high_water(), 1024);
    assert_eq!(second.high_water(), 1);
}

#[test]
fn deleted_or_abandoned_ids_stay_reserved_after_restoration() {
    let mut ids = RecipeIdAllocator::default();
    let kept = ids.allocate().unwrap();
    let deleted = ids.allocate().unwrap();
    let abandoned = ids.allocate().unwrap();
    let mut loaded = RecipeIdAllocator::restore(Some(ids.high_water()), &[kept]).unwrap();
    let next = loaded.allocate().unwrap();
    assert!(next > abandoned && next > deleted);
    assert_eq!(next, 4);
    let mut empty = RecipeIdAllocator::restore(Some(4), &[]).unwrap();
    assert_eq!(empty.allocate(), Ok(5));
}

#[test]
fn legacy_restore_uses_the_largest_live_id_without_order_dependence() {
    for order in [[1, 8, 3], [8, 3, 1], [3, 1, 8]] {
        let mut loaded = RecipeIdAllocator::restore(None, &order).unwrap();
        assert_eq!(loaded.high_water(), 8);
        assert_eq!(loaded.allocate(), Ok(9));
    }
    let mut empty = RecipeIdAllocator::restore(None, &[]).unwrap();
    assert_eq!(empty.allocate(), Ok(1));
}

#[test]
fn invalid_persisted_namespaces_are_refused() {
    let cases = [
        (None, vec![0], RecipeIdError::ZeroId),
        (None, vec![1, 2, 1], RecipeIdError::DuplicateId(1)),
        (
            Some(2),
            vec![3],
            RecipeIdError::BeyondHighWater {
                id: 3,
                high_water: 2,
            },
        ),
    ];
    for (watermark, live_ids, expected) in cases {
        let actual = RecipeIdAllocator::restore(watermark, &live_ids).unwrap_err();
        assert_eq!(actual, expected);
        assert!(actual.to_string().contains("expected"));
        assert!(std::error::Error::source(&actual).is_none());
    }
}

#[test]
fn full_width_ids_never_wrap_or_change_after_exhaustion() {
    let mut ids = RecipeIdAllocator::restore(Some(u64::MAX - 1), &[]).unwrap();
    assert_eq!(ids.allocate(), Ok(u64::MAX));
    for _ in 0..2 {
        assert_eq!(ids.allocate(), Err(RecipeIdError::Exhausted));
        assert_eq!(ids.high_water(), u64::MAX);
    }
    let mut restored = RecipeIdAllocator::restore(None, &[u64::MAX]).unwrap();
    assert_eq!(restored.allocate(), Err(RecipeIdError::Exhausted));
    assert!(
        RecipeIdError::Exhausted
            .to_string()
            .contains(&u64::MAX.to_string())
    );
}

#[test]
fn saved_local_ids_preserve_sketch_reference_names() {
    let doc: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let mut ids = RecipeIdAllocator::default();
    let sketch = ids.allocate().unwrap();
    let point = ids.allocate().unwrap();
    let keys = (doc.sketch_key(sketch), doc.sketch_entity_key(point));
    let mut loaded = RecipeIdAllocator::restore(Some(ids.high_water()), &[sketch, point]).unwrap();
    assert_eq!(keys, (doc.sketch_key(sketch), doc.sketch_entity_key(point)));
    assert_ne!(keys.1, doc.sketch_entity_key(loaded.allocate().unwrap()));
}
