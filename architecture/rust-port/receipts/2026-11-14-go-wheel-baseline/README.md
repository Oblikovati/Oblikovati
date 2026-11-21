# Navigation wheel reference baseline

Go reference: `aa8297d1b` (`go-legacy`). The following commands passed on
Linux:

```sh
go test ./app -run '^(TestToggleSteeringWheel|TestDispatchCancelDismissesSteeringWheel)$' -count=1
cd head
xvfb-run -a go test ./ui -run '^(TestSteeringWheelToolsRunActions|TestSteeringWedgePos)$' -count=1
```

The app tests cover toggling and cancelling the wheel. The head tests check
wedge placement and a subset of the wheel tool actions. They do not establish
the full, basic, and mini layouts or the native menus required by these
scenarios. The scenarios remain unverified until their fixture setup, native
runs, and captures are checked.
