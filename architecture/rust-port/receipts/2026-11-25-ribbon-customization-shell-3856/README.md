# Ribbon Customization while Shell is open

The published Apply scenario passed all 25 steps in the native Linux X11
desktop at 1440 × 900. Customize opened beside Shell and below the visible
solid. Applying an empty Tools target retained the empty User Commands
group while Customize was open. Closing Customize removed the group.
Cancelling Shell then returned to the existing solid.

[Apply report](apply/cucumber.json) · [Empty group capture](apply/empty-user-commands.png)

The published Cancel scenario passed 18 steps through discarding the
customization draft and selecting exactly one opening face. Its next
step required a certified face-removal preview. The pinned Martensite
build rejected that preview because the Shell vertex tolerance was
about 0.0005 and the permitted limit was about 0.0000001 (Martensite #554).

[Cancel diagnostic](cancel/cucumber.json) · [Kernel diagnostic capture](cancel/rejected-shell-preview.png)

The ordinary resize and reopen scenario passed all eight steps, and 67
Ribbon Customization and widget ABI library tests passed. The feature
hash matched before and after both Shell runs. The test checkout held
pending file-system changes outside this commit.

[Resize and reopen report](placement/cucumber.json) · [Source hash](input.sha256)
