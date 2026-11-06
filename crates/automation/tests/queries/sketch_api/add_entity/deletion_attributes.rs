// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::attribute::DocumentAttributes;

struct AttributeDeletionProbe {
    circle: SketchEntityId,
    targets: Vec<String>,
}

fn attributed_primitives(
    kind: DocumentType,
) -> (DiscoveryWorkspace, u64, QueryGuids, AttributeDeletionProbe) {
    let (mut files, document, mut source) = configured_files(kind, Default::default());
    create_empty_sketch(&mut files, &mut source);
    let circle = add_call(&mut files, &mut source, &circle_input()).unwrap();
    let survivor = add_call(&mut files, &mut source, &circle_input()).unwrap();
    let targets = vec![
        String::new(),
        row(&files, circle.entity_id).reference_key,
        defining_point_key(&files, document, &circle),
        row(&files, survivor.entity_id).reference_key,
    ];
    tag_targets(&mut files, document, &mut source, &targets);
    let probe = AttributeDeletionProbe {
        circle: SketchEntityId(circle.entity_id),
        targets,
    };
    (files, document, source, probe)
}

fn defining_point_key(
    files: &DiscoveryWorkspace,
    document: u64,
    circle: &AddSketchEntityResult,
) -> String {
    files
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name
        .sketch_entity_key(circle.point_i_ds.as_ref().unwrap()[0])
        .to_string()
}

fn tag_targets(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    targets: &[String],
) {
    let client = Client::new(FileHost::new(files, source, "test"));
    for target in targets {
        let request = SetAttributeArgs {
            document,
            set: "DeletionProbe".into(),
            name: "tag".into(),
            target: target.clone(),
            value: Variant {
                value: VariantValue::Double(-0.0),
                unit: String::new(),
            },
        };
        assert!(client.attributes().set_on(&request).unwrap().found);
    }
}

fn attributes(files: &DiscoveryWorkspace, document: u64) -> &DocumentAttributes {
    &files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .attributes
}

fn delete_attributed_circle(
    files: &mut DiscoveryWorkspace,
    document: u64,
    probe: &AttributeDeletionProbe,
) {
    let sketch = files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .sketches()[0]
        .id();
    let report = files
        .commands(document)
        .unwrap()
        .delete_sketch_entities(sketch, &[probe.circle])
        .unwrap();
    assert_eq!(report.entities.len(), 2);
    assert!(report.entities.contains(&probe.circle));
}

fn resolve_target(files: &DiscoveryWorkspace, target: &str) -> bool {
    Client::new(FileQueryHost::new(files))
        .sketch()
        .resolve_reference(&ResolveSketchReferenceArgs {
            reference_key: target.into(),
        })
        .unwrap()
        .found
}

fn assert_readable_metadata(
    files: &DiscoveryWorkspace,
    document: u64,
    probe: &AttributeDeletionProbe,
) {
    for target in &probe.targets {
        let result = get_tag(files, document, target);
        assert!(result.found);
        assert_eq!(result.attribute.target, *target);
        assert_signed_zero(result.attribute.value.value);
    }
    assert_eq!(metadata_target_names(files, document), probe.targets);
}

fn assert_signed_zero(value: VariantValue) {
    let VariantValue::Double(value) = value else {
        panic!("attribute {value:?}: expected a Double");
    };
    assert_eq!(value.to_bits(), (-0.0_f64).to_bits());
}

fn assert_retired_geometry(files: &DiscoveryWorkspace, probe: &AttributeDeletionProbe) {
    assert!(!resolve_target(files, &probe.targets[1]));
    assert!(!resolve_target(files, &probe.targets[2]));
    assert!(resolve_target(files, &probe.targets[3]));
}

#[test]
fn typed_attributes_remain_readable_when_entity_and_defining_point_names_retire() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        verify_attribute_history(kind);
    }
}

fn verify_attribute_history(kind: DocumentType) {
    let (mut files, document, mut source, probe) = attributed_primitives(kind);
    files.save(document, "test", &mut source).unwrap();
    let before = attributes(&files, document).clone();
    delete_attributed_circle(&mut files, document, &probe);
    assert!(before.same_recipe(attributes(&files, document)));
    assert_readable_metadata(&files, document, &probe);
    assert_retired_geometry(&files, &probe);
    files.commands(document).unwrap().undo().unwrap();
    assert!(resolve_target(&files, &probe.targets[1]));
    assert!(resolve_target(&files, &probe.targets[2]));
    assert!(!files.workspace().needs_save(document).unwrap());
    files.commands(document).unwrap().redo().unwrap();
    assert_retired_geometry(&files, &probe);
    assert!(before.same_recipe(attributes(&files, document)));
}

#[test]
fn native_reopen_and_copy_preserve_opaque_metadata_without_rebinding_retired_targets() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        verify_attribute_storage(kind);
    }
}

fn verify_attribute_storage(kind: DocumentType) {
    let (mut files, document, mut source, probe) = attributed_primitives(kind);
    delete_attributed_circle(&mut files, document, &probe);
    let before = attributes(&files, document).clone();
    let reopened = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert!(before.same_recipe(attributes(&files, reopened)));
    assert_readable_metadata(&files, reopened, &probe);
    assert_retired_geometry(&files, &probe);
    activate_copy(
        &mut files,
        reopened,
        &mut source,
        Path::new("/fixtures/deletion-attributes-copy.obk"),
    );
    let copy = files.workspace().active_document().unwrap();
    assert!(before.same_recipe(attributes(&files, copy)));
    assert_readable_metadata(&files, copy, &probe);
    assert_copy_target_isolation(&files, copy, &probe);
}

fn assert_copy_target_isolation(
    files: &DiscoveryWorkspace,
    document: u64,
    probe: &AttributeDeletionProbe,
) {
    for target in &probe.targets[1..] {
        assert!(!resolve_target(files, target));
    }
    let copied_key = rows(files)
        .into_iter()
        .find(|row| row.kind == "circle")
        .unwrap()
        .reference_key;
    assert!(resolve_target(files, &copied_key));
    assert!(!get_tag(files, document, &copied_key).found);
}

fn get_tag(files: &DiscoveryWorkspace, document: u64, target: &str) -> AttributeResult {
    let request = GetAttributeArgs {
        document,
        set: "DeletionProbe".into(),
        name: "tag".into(),
        target: target.into(),
    };
    Client::new(FileQueryHost::new(files))
        .attributes()
        .get_on(&request)
        .unwrap()
}
fn metadata_target_names(files: &DiscoveryWorkspace, document: u64) -> Vec<String> {
    let request = ListAttributesArgs {
        document,
        set: "DeletionProbe".into(),
        all_targets: true,
        ..Default::default()
    };
    let reply = Client::new(FileQueryHost::new(files))
        .attributes()
        .list(&request)
        .unwrap();
    reply
        .attributes
        .unwrap()
        .into_iter()
        .map(|row| row.target)
        .collect()
}

#[test]
fn explicit_attribute_delete_removes_a_retired_target_value_without_binding_geometry() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        verify_explicit_attribute_delete(kind);
    }
}

fn verify_explicit_attribute_delete(kind: DocumentType) {
    let (mut files, document, mut source, probe) = attributed_primitives(kind);
    delete_attributed_circle(&mut files, document, &probe);
    remove_retired_attribute(&mut files, document, &mut source, &probe.targets[1]);
    assert!(!get_tag(&files, document, &probe.targets[1]).found);
    assert!(get_tag(&files, document, &probe.targets[2]).found);
    assert_retired_geometry(&files, &probe);
    files.commands(document).unwrap().undo().unwrap();
    assert_readable_metadata(&files, document, &probe);
    assert_retired_geometry(&files, &probe);
    files.commands(document).unwrap().redo().unwrap();
    let reopened = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert!(!get_tag(&files, reopened, &probe.targets[1]).found);
    assert!(get_tag(&files, reopened, &probe.targets[2]).found);
    assert_retired_geometry(&files, &probe);
}

fn remove_retired_attribute(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    target: &str,
) {
    let request = DeleteAttributeArgs {
        document,
        set: "DeletionProbe".into(),
        name: "tag".into(),
        target: target.into(),
    };
    let result = Client::new(FileHost::new(files, source, "test"))
        .attributes()
        .delete_on(&request)
        .unwrap();
    assert_eq!(result.removed, 1);
}
