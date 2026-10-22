# Changelog

## [1.0.1](https://github.com/saltfishpr/jsonpath/compare/v1.0.0...v1.0.1) (2026-10-22)


### Performance Improvements

* avoid big integer allocations in common number comparisons ([9e80426](https://github.com/saltfishpr/jsonpath/commit/9e80426d36c7dfd7bf90caec71961da0b329a5e4))
* fast-path literal regex matching and align benchmarks ([4b94bba](https://github.com/saltfishpr/jsonpath/commit/4b94bbabd23e62be2c11ee932ccc2a3347e28a64))
* pool jsonNode allocations in blocks ([9380fa3](https://github.com/saltfishpr/jsonpath/commit/9380fa3b11cb1fc2f4b1df83558e69db8751dbba))
* reduce allocations in normalized path construction ([9ee5149](https://github.com/saltfishpr/jsonpath/commit/9ee5149551b5a016a337fd576e0683eb042e02a8))
* reduce allocations in parsing and singular query evaluation ([6fea547](https://github.com/saltfishpr/jsonpath/commit/6fea547125e86c1c73e4c1fae2f545f881f6b44f))

## [1.0.0](https://github.com/saltfishpr/jsonpath/compare/v0.2.0...v1.0.0) (2026-10-21)


### ⚠ BREAKING CHANGES

* the Get, GetMany, Result and JSONType APIs are removed; callers must switch to Compile/Parse with Values or Nodes.

### Features

* rewrite as RFC 9535 compliant implementation ([d99687e](https://github.com/saltfishpr/jsonpath/commit/d99687ee6d347a905a5bacaa15dc5aa66cdf8758))

## [0.2.0](https://github.com/saltfishpr/jsonpath/compare/v0.1.0...v0.2.0) (2026-02-13)


### Features

* v0.2.0 ([d7b03a7](https://github.com/saltfishpr/jsonpath/commit/d7b03a75730c7e4a5b82124623a928f58b944ab7))
