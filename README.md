# gvt, the go vendoring tool

`gvt` is a simple Go vendoring tool made for the GO15VENDOREXPERIMENT.  
It's based entirely on [gb-vendor](https://github.com/constabulary/gb).

You run `gvt fetch` when you would run `go get`. `gvt` downloads packages to `./vendor/...`. With `GO15VENDOREXPERIMENT=1` set the compiler will find and use those dependencies without import path rewriting. `gvt` works recursively as you would expect, and lets you update vendored dependencies. It also writes a manifest to `./vendor/manifest`.

Packages whose dependencies are vendored with `gvt` are `go build`-able and `go get`-able by Go 1.5 with `GO15VENDOREXPERIMENT=1` set.

## Installation

```
go get -u github.com/FiloSottile/gvt
```

## Usage

```
$ gvt fetch github.com/fatih/color
2015/09/05 02:38:06 fetching recursive dependency github.com/mattn/go-isatty
2015/09/05 02:38:07 fetching recursive dependency github.com/shiena/ansicolor

$ tree -d
.
└── vendor
    └── github.com
        ├── fatih
        │   └── color
        ├── mattn
        │   └── go-isatty
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

$ gvt update github.com/fatih/color
```

Full usage on [GoDoc ![GoDoc](https://godoc.org/github.com/FiloSottile/gvt?status.svg)](http://godoc.org/github.com/FiloSottile/gvt)

## Why

There are many Go vendoring tools, but they all have some subset of the following problems

   * no GO15VENDOREXPERIMENT support: old tools are based on import path rewriting or GOPATH overrides
   * requirement to run on clients: some require the user to install the tool and run it after cloning, which breaks `go get`
   * **no real fetching support**: tools like Godep just copy packages from your GOPATH, instead of pulling it from the Internet
   * prominent metadata files: there's no need for the manifest to be in your repository root, or in its own empty folder
   * different build stack: gb-vendor is awesome but it requires you to build your project with gb
