# gocoro

Lua coroutine implementation in a Go package. This repository is the accompanying code for the blog post [Implementing Lua Coroutines In Go][1].

It has different implementations depending on the branches. The `simple-int` branch provides the basic implementation for a simple use-case that only yields an integer. The `generic` (and `master`) branch is the full-featured, empty interface-based implementation that is closest to Lua's coroutines. The `make-func` branch is an experimental branch for a reflect package-based implementation.

[1]: http://0value.com/implementing-lua-coroutines-in-Go
