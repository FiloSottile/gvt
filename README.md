`gvt` was a minimalistic Go vendoring tool made for the `vendor/` folder (once known as the
[GO15VENDOREXPERIMENT](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo/edit)).

It was based on [gb-vendor](https://github.com/constabulary/gb) by Dave Cheney.

Since Go 1.11, the `go` tool supports *modules*, a native solution to the dependency problem.

The `go` tool understands `gvt` manifest files, so you just have to run

```
GO111MODULE=on go mod init
GO111MODULE=on go mod vendor
```

to migrate and still populate the `vendor/` folder for backwards compatibility.

Read more [in the docs](https://tip.golang.org/cmd/go/#hdr-Modules__module_versions__and_more) or [on the wiki](https://golang.org/wiki/Modules).

Modules support is experimental in 1.11, but it will probably serve you better than gvt would.

  — So long, and thanks for all the fish!
