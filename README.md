# midterm [![GoDoc][Badge]][GoDoc]

> a pretty mid terminal library

Midterm is a virtual terminal emulator. There is no GUI, but it has
conveniences for rendering back to a terminal or to HTML.

Some examples:

* [Progrock] uses it for displaying progress logs.
* [Dagger] uses it for interactive shells, also rendered via Progrock.
* [Bass] uses it for rendering terminal output in its docs. (And also uses
  Progrock.)

### Project goals

* Compatibility with everyday tools like `htop` and `vim`.
* Good enough performance, though optimizations haven't been sought out yet so
  there's probably some low hanging fruit.
* Composability/versatility, e.g. forwarding OSC requests/responses between an
  outer terminal and a remote shell in a container.
* Anything you'd expect for interactive shells: full mouse support, 256 colors,
  copy/paste, etc. - though there's no GUI so this really just amounts to
  forwarding ANSI sequences between local/remote shells.

### What's it for?

This is not a GUI terminal emulator intended for everyday use. It's all
in-memory. If you want to wrap it in a GUI, feel free!

Right now it's used for
rendering terminals embedded in other TUIs (Progrock), and for rendering
terminal output in documentation.

### What's with the name?

It used to be called vt100, but then I added support for things beyond vt100
like scroll regions, and I don't want to keep renaming it.

I went with midterm because this library often sits in between a local and
remote terminal (e.g. `dagger shell`), so it's a middle terminal.
:man_shrugging:

### Credit

Based on [tonistiigi/vt100] which was was based on [jaguilar/vt100].

[Badge]: https://godoc.org/github.com/vito/midterm?status.svg
[Bass]: https://github.com/vito/bass
[Dagger]: https://github.com/dagger/dagger
[GoDoc]: https://godoc.org/github.com/vito/midterm
[Progrock]: https://github.com/vito/progrock
[jaguilar/vt100]: https://github.com/jaguilar/vt100
[tonistiigi/vt100]: https://github.com/tonistiigi/vt100
