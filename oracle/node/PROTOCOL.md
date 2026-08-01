# Node oracle NDJSON protocol

This protocol exposes the fixed micromustache implementation only as a development and test oracle. It must never be called by the Go package at runtime or used as a product fallback.

## Transport

- Input and output are UTF-8 NDJSON.
- Each non-empty input line is one request; each request produces one response line in the same order.
- `stdout` contains response JSON only. Fatal diagnostics use `stderr`.
- Every request has a unique non-empty string `id`, a supported string `op`, and an operation-specific `args` object.
- Every response repeats the request `id`.
- The fixed CommonJS bundle is loaded fresh for every request so its internal caches are not shared between requests.

Success:

```json
{"id":"case-1","ok":true,"value":{"type":"string","value":"Hello"}}
```

Failure:

```json
{"id":"case-1","ok":false,"error":{"name":"TypeError","message":"..."}}
```

Errors include stable `name` and `message` fields. Stack traces are intentionally excluded. Invalid JSON cannot supply an id, so its failure response uses `null`.

## Encoded values

All data, scope, result, and resolver values use a recursive envelope. Because every value node has a declared `type`, a plain object cannot collide with a special-value tag.

| JavaScript value | Encoding |
| --- | --- |
| `undefined` | `{"type":"undefined"}` |
| `null` | `{"type":"null"}` |
| boolean | `{"type":"boolean","value":true}` |
| finite number | `{"type":"number","value":1.5}` |
| `NaN` | `{"type":"nan"}` |
| `Infinity` | `{"type":"infinity"}` |
| `-Infinity` | `{"type":"negative-infinity"}` |
| negative zero | `{"type":"negative-zero"}` |
| string | `{"type":"string","value":"text"}` |
| array | `{"type":"array","value":[...encoded values...]}` |
| plain object | `{"type":"object","value":{"key":...encoded value...}}` |

`bigint`, `Date`, and function values are not accepted. A validation-only resolver action may create a fixed `Symbol` internally to exercise the unsupported-value boundary; requests still cannot provide code.

## Resolver specification

Resolver operations accept a declarative resolver. No request-provided JavaScript is evaluated.

```json
{
  "paths": {
    "user.name": {"action":"value","value":{"type":"string","value":"Ada"}},
    "missing": {"action":"undefined"},
    "broken": {"action":"error","error":{"name":"RangeError","message":"broken"}}
  },
  "default": {"action":"undefined"}
}
```

Synchronous operations return or throw the selected action directly. Asynchronous operations use an `async` resolver, so values and errors become Promise fulfillment and rejection. A validation-only `delayMs` delays fulfillment or rejection by a fixed number of milliseconds, and `trace: true` records declarative resolver calls. Resolver lookup depends only on the exact path string and shares no mutable state between requests.

## Operations

The fixed package exports eight runtime names: `render`, `renderFn`, `renderFnAsync`, `compile`, `get`, `getRef`, `tokenize`, and `Renderer`. Function-valued `compile` results are represented by immediate method calls.

| `op` | Required `args` | Fixed API call |
| --- | --- | --- |
| `render` | `template`, optional encoded `data`, optional `options` | `render(template, scope, options)` |
| `renderFn` | `template`, `resolver`, optional encoded `scope`, optional `options` | `renderFn(template, resolver, scope, options)` |
| `renderFnAsync` | same as `renderFn` | `await renderFnAsync(...)` |
| `compile.render` | `template`, optional encoded `data`, optional `options` | `compile(template, options).render(scope)` |
| `compile.renderFn` | `template`, `resolver`, optional encoded `scope`, optional `options` | `compile(...).renderFn(...)` |
| `compile.renderFnAsync` | same as `compile.renderFn` | `await compile(...).renderFnAsync(...)` |
| `get` | encoded `scope`, string `path`, optional `options` | `get(scope, path, options)` |
| `getRef` | encoded `scope`, string-array `ref`, optional `options` | `getRef(scope, ref, options)` |
| `tokenize` | `template`, optional `options` | `tokenize(template, options)` |
| `renderer.render` | `tokens`, optional encoded `data`, optional `options` | `new Renderer(tokens, options).render(scope)` |
| `renderer.renderFn` | `tokens`, `resolver`, optional encoded `scope`, optional `options` | `new Renderer(...).renderFn(...)` |
| `renderer.renderFnAsync` | same as `renderer.renderFn` | `await new Renderer(...).renderFnAsync(...)` |
| `compile` | `template`, optional `options` | compile and return the public token/options observation |
| `renderer.construct` | `tokens`, optional `options` | construct and return the public token/options observation |
| `compile.sequence` | `template`, optional `options`, declarative `steps` | compile once and run ordered renderer steps on the same instance |
| `renderer.sequence` | `tokens`, optional `options`, declarative `steps` | construct once and run ordered renderer steps on the same instance |

`tokens` follows the measured constructor shape: `{ "strings": [string...], "paths": [string...] }`. Sequence steps are limited to `render`, `renderFn`, and `renderFnAsync` plus encoded data/scope, declarative resolver tables, context labels, and trace flags. Each step returns its own success or error envelope so reuse after a failure is observable. No arbitrary code, network input, or state shared between requests is permitted. The committed smoke file has an additional boolean `expectOk` field used only by the smoke verifier; the oracle ignores it.
