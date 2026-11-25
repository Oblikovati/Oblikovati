# Home Recent search and filename width

The search input, Clear control, and magnifier share one field border.
The direct native test used X11 input to search in Grid View, clear the
query, and submit the empty field to restore the broader list.
[Filtered grid](grid-search.png) · [Restored grid](grid-restored.png)

The layout test checked the 36% filename allocation at 900 px and kept
the narrow Size heading and visible column checks. The 240 px filename
target applies where the visible metadata columns leave enough room.

Eight published scenarios and 98 steps passed on Linux under Xvfb,
Openbox, and Vulkan Lavapipe. The checks covered clearing and restoring
results, recall, persistence between List and Grid, filters, and bulk
selection. The exported Home Recent scenario source has SHA-256
`aa6286e17139d517475cdb3c9597c32ab339fc265423e6b53682ccf7ef0d097d`.
[Search scenarios](scenarios/cucumber.json) · [Restore scenario](restore/cucumber.json)
