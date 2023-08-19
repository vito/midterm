# VT102

[![GoDoc](https://godoc.org/github.com/vito/vt102?status.svg)](https://godoc.org/github.com/vito/vt102)

> Based on [tonistiigi/vt100] which was was based on [jaguilar/vt100](https://github.com/jaguilar/vt100).

A Go virtual terminal library. Used by [Progrock] and [Dagger]. Intended to support full blown TUI applications like `vim` and `htop`. (This README was bootstrapped in `vim`. There were many bugs.)

There is no GUI; it's just an in-memory data structure. If you want to wrap it in a GUI, feel free! Right now it's used for rendering terminals embedded in other TUIs (Progrock), and for rendering terminal output in documentation.

Performance _is_ a priority but has not been a priority yet.

[Progrock]: https://github.com/vito/progrock
[Dagger]: https://github.com/dagger/dagger.
