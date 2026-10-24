# examples-gen

Fills the outputs of the examples in `examples/README.md`.

In a code block, a command starts with `➜ ` and its output goes below it. When the line after a command is `<output>`, the tool runs the command and puts its output (stdout and stderr) in place of the placeholder:

```
➜ xprin test examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml -v
<output>
```

A block without a placeholder is not touched. To regenerate an output, replace it with `<output>`, or use the options below.

Some outputs depend on the Crossplane CLI (for example, the ones of the Crossplane CLI v1). A code block with a tag, in a comment on the line right before it, is filled only when the tool runs with the same `--tag`, and a block without a tag only when it runs without one:

````
<!-- examples-gen: v1 -->
```
➜ xprin check
<output>
```
````

## Run it

From the root of the repository, with `xprin`, the Crossplane CLI, Docker and `yq` available (`xprin` has to be in the `PATH`):

```shell
go build -o /tmp/bin/xprin ./cmd/xprin
PATH=/tmp/bin:$PATH go run ./cmd/examples-gen examples/README.md                            # fill the placeholders
PATH=/tmp/bin:$PATH go run ./cmd/examples-gen --regenerate-all examples/README.md           # regenerate every output
PATH=/tmp/bin:$PATH go run ./cmd/examples-gen --regenerate-match=hooks examples/README.md   # the commands that contain "hooks"
```

Add `--tag=v1` to fill the blocks tagged `v1` instead of the untagged ones, with the Crossplane CLI v1 in the `PATH`. `--regenerate-all` and `--regenerate-match` can't be used together. The commands run in the working directory and can change files (for example `update-goldens`), so look at the diff afterwards.

## Run it with Earthly

Docker-in-Docker, with the Crossplane CLIs and `yq`. Only `examples/README.md` is exported:

```shell
earthly --allow-privileged +regen-examples                                # fill the placeholders
earthly --allow-privileged +regen-examples --REGENERATE_ALL=true          # regenerate every output
earthly --allow-privileged +regen-examples --REGENERATE_MATCH=hooks       # the commands that contain "hooks"
```

It runs the examples with the Crossplane CLI v2 (the untagged blocks), and then with v1 (the blocks that are tagged).
