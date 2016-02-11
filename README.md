# gvt, the go vendoring tool
[![GoDoc](https://godoc.org/github.com/FiloSottile/gvt?status.svg)](https://godoc.org/github.com/FiloSottile/gvt)
[![Build Status](https://travis-ci.org/FiloSottile/gvt.svg?branch=master)](https://travis-ci.org/FiloSottile/gvt)

`gvt` is a simple Go vendoring tool made for the
[GO15VENDOREXPERIMENT](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo/edit),
based on [gb-vendor](https://github.com/constabulary/gb).

It lets you easily and "idiomatically" include external dependencies in your repository to get
reproducible builds.

  * No need to learn a new tool or format!  
    You already know how to use `gvt`: just run `gvt fetch` when and like you would run `go get`.
    You can imagine what `gvt update` and `gvt delete` do.

  * No need to change how you build your project!  
    `gvt` downloads packages to `./vendor/...`. With `GO15VENDOREXPERIMENT=1` the stock Go compiler
    will find and use those dependencies automatically (without import path or GOPATH changes).

  * No need to manually chase, copy or cleanup dependencies!  
    `gvt` works recursively as you would expect, and lets you update vendored dependencies. It also
    writes a manifest to `./vendor/manifest` and never touches your system GOPATH. Finally, it
    strips the VCS metadata so that you can commit the vendored source cleanly.

  * No need for your users and occasional contributors to install **or even know about** gvt!  
    Packages whose dependencies are vendored with `gvt` are `go build`-able and `go get`-able out of
    the box by Go 1.5 with `GO15VENDOREXPERIMENT=1` set.

*Note that projects must live within the GOPATH tree in order to be go buildable with the
GO15VENDOREXPERIMENT flag.*

If you use and like (or dislike!) `gvt`, it would definitely make my day better if you dropped a
line at `gvt -at- filippo.io` :)

## Installation

With a [correctly configured](https://golang.org/doc/code.html#GOPATH) Go installation:

```
GO15VENDOREXPERIMENT=1 go get -u github.com/FiloSottile/gvt
```

## Usage

You know how to use `go get`? That's how you use `gvt fetch`.

```
# This will fetch the dependency into the ./vendor folder.
$ gvt fetch github.com/fatih/color
2015/09/05 02:38:06 fetching recursive dependency github.com/mattn/go-isatty
2015/09/05 02:38:07 fetching recursive dependency github.com/shiena/ansicolor

$ tree -d
.
└── vendor
    └── github.com
        ├── fatih
        │   └── color
        ├── mattn
        │   └── go-isatty
        └── shiena
            └── ansicolor
                └── ansicolor

9 directories

$ cat > main.go
package main
import "github.com/fatih/color"
func main() {
    color.Red("Hello, world!")
}

$ export GO15VENDOREXPERIMENT=1
$ go build .
$ ./hello
Hello, world!

$ git add main.go vendor/ && git commit

```

A full set of example usage can be found on [GoDoc](https://godoc.org/github.com/FiloSottile/gvt).

## Alternative: not checking in vendored source

Some developers prefer not to check in the source of the vendored dependencies. In that case you can
add lines like these to e.g. your `.gitignore`

    vendor/**
    !vendor/manifest

When you check out the source again, you can then run `gvt restore` to fetch all the dependencies at
the revisions specified in the `vendor/manifest` file.

Please consider that this approach has the following consequences:

  * the package consumer will need gvt to fetch the dependencies
  * the dependencies will need to remain available from the source repositories: if the original
    repository goes down or rewrites history, build reproducibility is lost
  * `go get` won't work on your package

## Troubleshooting

### `fatal: Not a git repository [...]`
### `error: tag 'fetch' not found.`

These errors can occur because you have an alias for `gvt` pointing to `git verify-tag`
(default if using oh-my-zsh).

Recent versions of oh-my-zsh [removed the alias](https://github.com/robbyrussell/oh-my-zsh/pull/4841). You can update with `upgrade_oh_my_zsh`.

Alternatively, run this, and preferably add it to your `~/.bashrc` / `~/.zshrc`: `unalias gvt`.

### `go build` can't find the vendored package

Make sure you set `GO15VENDOREXPERIMENT=1`.

Also note that GO15VENDOREXPERIMENT does not apply when outside the GOPATH tree. That is, your
project must be somewhere in a subfolder of `$GOPATH`.

## License

MIT licensed. See the LICENSE file for details.
