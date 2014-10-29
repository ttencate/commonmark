CommonMark for Go
=================

[`commonmark`](https://github.com/ttencate/commonmark) is a standards-compliant,
pluggable and fast [CommonMark](http://commonmark.org) implementation for
[Go](http://golang.org) (golang).

Installing
----------

    go get github.com/ttencate/commonmark

Documentation
-------------

- TODO find out how to get documentation up on godoc.org

Developing
----------

- TODO write about running the test suite
- TODO gofmt pre-commit hook

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
