# Changelog

## [1.1.1](https://github.com/saltfishpr/jsonpath/compare/v1.1.0...v1.1.1) (2026-10-24)


### Performance Improvements

* **document:** reduce parser allocations ([3573381](https://github.com/saltfishpr/jsonpath/commit/357338191b8afe12beed2fb6bb14f29f12a0827f))

## [1.1.0](https://github.com/saltfishpr/jsonpath/compare/v1.0.1...v1.1.0) (2026-10-24)


### Features

* compile queries with typed AST and shared document values ([73154e5](https://github.com/saltfishpr/jsonpath/commit/73154e55e256610e47ffbc9a6b714fec182b4292))
* compile queries with typed AST and shared document values ([e91834b](https://github.com/saltfishpr/jsonpath/commit/e91834bae6f69f100f99c170f1ab40aae7fbb055))


### Bug Fixes

* return CompileError for lexical and syntax errors ([614c151](https://github.com/saltfishpr/jsonpath/commit/614c1512640890d33b63ad3c0f3455c45e3d5e91))
* return false for invalid match and search regex patterns ([1826219](https://github.com/saltfishpr/jsonpath/commit/1826219c18c59979f58b1cd936e57a7bf97b207b))


### Performance Improvements

* reduce query allocations and optimize literal regex ([d8a57fd](https://github.com/saltfishpr/jsonpath/commit/d8a57fd8514c773b5491e39d23f0fd267603ed12))

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
