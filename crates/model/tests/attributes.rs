// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::{json::Bytes, types::VariantValue};
use oblikovati_model::attribute::{AttributeTarget, DocumentAttributes};

fn target(context: u64) -> AttributeTarget {
    let mut encoded = context.to_le_bytes().to_vec();
    encoded.extend_from_slice(&1u32.to_le_bytes());
    encoded.extend_from_slice(&1u32.to_le_bytes());
    encoded.push(42);
    AttributeTarget::from_encoded(&encoded).unwrap()
}

#[test]
fn equivalent_envelopes_share_identity_and_malformed_keys_are_rejected() {
    let original = target(5);
    let current = AttributeTarget::from_encoded(original.encoded()).unwrap();
    assert_eq!(original, current);
    assert_eq!(&original.encoded()[..5], b"OKRK\x01");
    assert_ne!(target(5), target(6));
    assert!(AttributeTarget::from_encoded(&[]).is_err());
    let mut unknown = original.encoded().to_vec();
    unknown[4] = 3;
    assert!(AttributeTarget::from_encoded(&unknown).is_err());
    unknown[4] = 1;
    unknown.push(0);
    assert!(AttributeTarget::from_encoded(&unknown).is_err());
}

#[test]
fn edits_retain_anchor_set_and_attribute_order_until_removed() {
    let mut attributes = DocumentAttributes::default();
    let first = target(2);
    let second = target(1);
    assert!(attributes.ensure_set(&first, "Empty").unwrap());
    assert!(!attributes.ensure_set(&first, "Empty").unwrap());
    attributes
        .put(&first, "AddIn", "first", VariantValue::Integer(1))
        .unwrap();
    attributes
        .put(&first, "AddIn", "second", VariantValue::Integer(2))
        .unwrap();
    attributes
        .put(&second, "AddIn", "first", VariantValue::Integer(3))
        .unwrap();
    attributes
        .put(&first, "AddIn", "first", VariantValue::Integer(4))
        .unwrap();
    assert_eq!(
        attributes
            .anchors()
            .iter()
            .map(|anchor| &anchor.target)
            .collect::<Vec<_>>(),
        [&first, &second]
    );
    assert_eq!(
        attributes.anchors()[0]
            .sets()
            .iter()
            .map(|set| set.name.as_str())
            .collect::<Vec<_>>(),
        ["Empty", "AddIn"]
    );
    assert_eq!(
        attributes
            .find("AddIn", "")
            .iter()
            .map(|hit| hit.attribute.value.clone())
            .collect::<Vec<_>>(),
        [
            VariantValue::Integer(4),
            VariantValue::Integer(2),
            VariantValue::Integer(3)
        ]
    );
    let hits = attributes.find("AddIn", "first");
    assert_eq!(hits.len(), 2);
    assert_eq!(hits[0].target, &first);
    assert_eq!(hits[0].set.name, "AddIn");
    assert!(attributes.remove(&first, "AddIn", "first"));
    attributes
        .put(&first, "AddIn", "first", VariantValue::Integer(5))
        .unwrap();
    assert_eq!(
        attributes.set(&first, "AddIn").unwrap().attributes()[0].name,
        "second"
    );
    assert!(attributes.find("Empty", "").is_empty());
    assert!(attributes.find("AddIn", "missing").is_empty());
}

#[test]
fn removals_keep_empty_owners_and_do_not_create_missing_targets() {
    let mut attributes = DocumentAttributes::default();
    let key = target(1);
    assert!(!attributes.remove(&key, "AddIn", "tag"));
    assert!(!attributes.remove_set(&key, "AddIn"));
    assert!(!attributes.remove_target(&key));
    assert!(attributes.get(&key, "AddIn", "tag").is_none());
    assert!(attributes.anchors().is_empty());
    attributes
        .put(&key, "AddIn", "tag", VariantValue::Boolean(true))
        .unwrap();
    assert!(!attributes.remove(&key, "AddIn", "missing"));
    assert!(attributes.remove(&key, "AddIn", "tag"));
    assert!(
        attributes
            .set(&key, "AddIn")
            .unwrap()
            .attributes()
            .is_empty()
    );
    assert!(!attributes.remove_set(&key, "missing"));
    assert!(attributes.remove_set(&key, "AddIn"));
    assert!(attributes.anchors()[0].sets().is_empty());
    assert!(attributes.remove_target(&key));
    assert!(attributes.anchors().is_empty());
}

#[test]
fn all_api_values_are_owned_and_signed_zero_is_part_of_the_recipe() {
    let key = target(7);
    let mut attributes = DocumentAttributes::default();
    for (index, value) in [
        VariantValue::Boolean(true),
        VariantValue::Integer(i64::MIN),
        VariantValue::Double(-0.0),
        VariantValue::String("text".into()),
        VariantValue::Bytes(Bytes(Some(vec![1, 2]))),
    ]
    .into_iter()
    .enumerate()
    {
        attributes
            .put(&key, "AddIn", &index.to_string(), value.clone())
            .unwrap();
        assert_eq!(
            attributes
                .get(&key, "AddIn", &index.to_string())
                .unwrap()
                .value,
            value
        );
    }
    let original = attributes.clone();
    assert!(attributes.same_recipe(&original));
    attributes
        .put(&key, "AddIn", "2", VariantValue::Double(0.0))
        .unwrap();
    assert_eq!(attributes, original);
    assert!(!attributes.same_recipe(&original));
    attributes
        .put(
            &key,
            "AddIn",
            "4",
            VariantValue::Bytes(Bytes(Some(vec![9]))),
        )
        .unwrap();
    assert_eq!(
        original.get(&key, "AddIn", "4").unwrap().value,
        VariantValue::Bytes(Bytes(Some(vec![1, 2])))
    );
    attributes
        .put(
            &key,
            "AddIn",
            "empty",
            VariantValue::Bytes(Bytes(Some(Vec::new()))),
        )
        .unwrap();
    assert_eq!(
        attributes.get(&key, "AddIn", "empty").unwrap().value,
        VariantValue::Bytes(Bytes(None))
    );
}

#[test]
fn invalid_edits_are_rejected_before_creating_an_anchor_or_changing_a_value() {
    let mut attributes = DocumentAttributes::default();
    let key = target(1);
    for value in [
        VariantValue::Uninitialized,
        VariantValue::Double(f64::NAN),
        VariantValue::Double(f64::INFINITY),
    ] {
        assert!(
            attributes
                .put(&key, "AddIn", "tag", value)
                .unwrap_err()
                .to_string()
                .contains("expected")
        );
        assert!(attributes.anchors().is_empty());
    }
    assert!(attributes.ensure_set(&key, "").is_err());
    assert!(
        attributes
            .put(&key, "", "tag", VariantValue::Integer(1))
            .is_err()
    );
    assert!(
        attributes
            .put(&key, "AddIn", "", VariantValue::Integer(1))
            .is_err()
    );
    assert!(attributes.anchors().is_empty());
    attributes
        .put(&key, "AddIn", "tag", VariantValue::Integer(1))
        .unwrap();
    let before = attributes.clone();
    assert!(
        attributes
            .put(&key, "AddIn", "tag", VariantValue::Double(f64::NAN))
            .is_err()
    );
    assert!(attributes.same_recipe(&before));
}

#[test]
fn semantic_attribute_targets_preserve_their_scheme_and_do_not_alias_go_keys() {
    use oblikovati_model::identity::SemanticReferenceName;
    let guid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let encoded = SemanticReferenceName::new(guid, 1, 1, "sketch-prism", "start", &[1])
        .unwrap()
        .encode()
        .unwrap();
    let semantic = AttributeTarget::from_encoded(&encoded).unwrap();
    assert_eq!(semantic.encoded(), encoded);
    let mut go_meaning = encoded;
    go_meaning[4] = 1;
    assert_ne!(
        semantic,
        AttributeTarget::from_encoded(&go_meaning).unwrap()
    );
}
