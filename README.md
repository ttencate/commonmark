CommonMark for Go
=================

[`commonmark`](https://github.com/ttencate/commonmark) aims to be a
standards-compliant, pluggable and somewhat fast
[CommonMark](http://commonmark.org) implementation for [Go](http://golang.org)
(golang).

It is currently being implemented, and not ready for use yet.

Installing
----------

    go get github.com/ttencate/commonmark

Documentation
-------------

- TODO find out how to get documentation up on godoc.org

Architecture
------------

Unlike the [reference implementation](https://github.com/jgm/CommonMark) in C,
the `commonmark` package doesn't use single-pass parsers for the block and the
inline parsing. Instead, it scans the document many times, once for each type
of content: detect and convert code blocks, detect and convert blockquotes, and
so on.

Obviously, this is less performant than doing just one pass. But it is not as
bad as you might think: because each pass does only one thing, it can be very
simple and fast, because the loop body doesn't need complicated logic and
branching.

The advantage is twofold. First, the code is much easier to read and
understand. Each pass worries about just one thing, instead of having the logic
for all types of markup intertwined and often interacting in subtle ways.

The second advantage is pluggability. It is easy to add plugins for new content
types (e.g. tables, metadata blocks), because they can just be inserted as a
separate pass over the partially processed document tree.

Developing
----------

After you cloned the repository, please run

    ./scripts/install_hooks.sh

to install a Git pre-commit hook that reminds you to run `gofmt`.

- TODO write about running the test suite

Benchmarks
----------

- TODO benchmark and compare to some other implementations, in particular the C
  reference implementation

License
-------

Three-clause BSD license. See the `LICENSE` file for details.

Alternatives
------------

- [sudhirj/godown](https://github.com/sudhirj/godown), at the time of writing
  (Oct 2014), claims to be a "Fast and parallel CommonMark parser" but is as of
  yet incomplete (only 55 lines of code), and doesn't appear to be parallel.
- [valoox/gomk](https://github.com/valoox/gomk), at the time of writing (Oct
  2014), claims to be a "Go implementation of the standard CommonMark markdown
  parser" but contains no code yet.
- There might be a case for wrapping the
  [official C implementation](https://github.com/jgm/CommonMark) in a Go
  library. It will not be pluggable, but it will likely be faster. At the time
  of writing (Oct 2014), no such wrapper exists.
