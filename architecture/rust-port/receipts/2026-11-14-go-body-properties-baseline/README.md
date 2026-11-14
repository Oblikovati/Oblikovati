# Go body-properties baseline

Go checkout: `aa8297d1b` (`go-legacy`). `app/material_ops.go` provides
`AssignAppearance` and part-level `PhysicalProperties`. The latter aggregates
mesh-derived body properties. `head/ui/materials_window.go` displays the
part-level properties. No matching Go body-properties dialog test was found.

```text
$ go test ./app -run '^(TestAssignAppearanceBodyAndFaceScope|TestPhysicalPropertiesNotice|TestPhysicalPropertiesWithoutPart)$' -count=1 -v
--- PASS: TestPhysicalPropertiesWithoutPart (0.00s)
--- PASS: TestPhysicalPropertiesNotice (0.02s)
--- PASS: TestAssignAppearanceBodyAndFaceScope (0.02s)
PASS
ok  oblikovati.org/app  0.024s
```

`TestAssignAppearanceBodyAndFaceScope` checks that synthetic keys map to the
assigned appearance in the store. Its premise is sufficient for the stored
mapping, but it does not validate those keys against a picked body, draw a
color, or test a dialog. The two physical-properties tests cover a part-level
notice and the error without a part. They do not check per-body values or
numeric accuracy. Thus these tests do not prove the six native dialog
scenarios; the scenario receipts and their own assertions are separate
acceptance evidence.
