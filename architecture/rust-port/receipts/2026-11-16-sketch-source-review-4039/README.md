# Line and rectangle source review

The pinned source is `b576445b98cffd686dcc05aa8c017de2a591c8c8`.
Four Line and three Two Point Rectangle scenario blocks match their earlier
versions in the `c56a6f6` snapshot. The Line feature now has a shared
background that requires an editable sketch and an idle viewport pointer.
Both features now carry the common sketch viewport tag. Other scenarios in
those files also changed; this review does not classify or verify those cases.

The seven classifications remain inventoried. Earlier receipts remain
historical, and no prior baseline is bound to the new feature hash by this
review. The added background must run in any current native acceptance test.
