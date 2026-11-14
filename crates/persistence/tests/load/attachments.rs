// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_persistence::load::load_parameter_package;

#[test]
fn malformed_attachment_load_claims_no_identity_and_consumes_no_guid() {
    let mut index = DocumentIdentityIndex::default();
    let mut generator = source();
    let encoded = b"schemaVersion: 2\ndocumentType: 1\nattachments: [{name: bad, kind: embedded, fullFileName: file, resourceId: invalid}]\n";
    let result = load_parameter_package(
        Path::new("attachments.opd"),
        1,
        "alpha",
        &mut index,
        &mut generator,
        &mut reader(encoded),
    );
    assert!(result.unwrap_err().to_string().contains("attachment"));
    assert!(index.identity(1).is_none());
    assert_eq!(generator.next, 0);
}
