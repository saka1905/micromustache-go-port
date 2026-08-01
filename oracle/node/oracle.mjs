import { createReadStream } from 'node:fs'
import { createRequire } from 'node:module'
import { createInterface } from 'node:readline'
import { fileURLToPath, pathToFileURL } from 'node:url'
import path from 'node:path'
import process from 'node:process'

import { decode, encode } from './codec.mjs'

const require = createRequire(import.meta.url)
const upstreamEntry = fileURLToPath(new URL('../upstream/dist/micromustache.cjs', import.meta.url))
const requiredExports = [
  'Renderer',
  'compile',
  'get',
  'getRef',
  'render',
  'renderFn',
  'renderFnAsync',
  'tokenize',
]

function loadUpstream() {
  const resolvedEntry = require.resolve(upstreamEntry)
  delete require.cache[resolvedEntry]
  const loaded = require(resolvedEntry)
  for (const exportName of requiredExports) {
    if (!(exportName in loaded)) {
      throw new Error(`Built upstream entry is missing export ${exportName}`)
    }
  }
  return loaded
}

function requireRecord(value, label) {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) {
    throw new TypeError(`${label} must be an object`)
  }
  return value
}

function decoded(value, fallback) {
  return value === undefined ? fallback : decode(value)
}

function namedError(spec) {
  requireRecord(spec, 'resolver error')
  if (typeof spec.name !== 'string' || typeof spec.message !== 'string') {
    throw new TypeError('resolver error name and message must be strings')
  }
  const error = new Error(spec.message)
  error.name = spec.name
  return error
}

function resolverAction(spec, pathName) {
  requireRecord(spec, 'resolver')
  const pathActions = spec.paths === undefined ? {} : requireRecord(spec.paths, 'resolver.paths')
  const action = Object.prototype.hasOwnProperty.call(pathActions, pathName)
    ? pathActions[pathName]
    : spec.default
  requireRecord(action, `resolver action for ${JSON.stringify(pathName)}`)

  if (action.delayMs !== undefined && (!Number.isInteger(action.delayMs) || action.delayMs < 0)) {
    throw new TypeError(`resolver delayMs must be a non-negative integer for ${JSON.stringify(pathName)}`)
  }
  return action
}

function resolveAction(action, pathName) {
  switch (action.action) {
    case 'value':
      return decode(action.value)
    case 'undefined':
      return undefined
    case 'error':
      throw namedError(action.error)
    case 'unsupported':
      return Symbol(`unsupported:${pathName}`)
    default:
      throw new TypeError(`Unknown resolver action for ${JSON.stringify(pathName)}`)
  }
}

function makeResolver(spec, asynchronous, calls = undefined) {
  if (asynchronous) {
    return async (pathName) => {
      if (calls !== undefined) calls.push(pathName)
      const action = resolverAction(spec, pathName)
      if (action.delayMs > 0) {
        await new Promise((resolve) => setTimeout(resolve, action.delayMs))
      }
      return resolveAction(action, pathName)
    }
  }
  return (pathName) => {
    if (calls !== undefined) calls.push(pathName)
    return resolveAction(resolverAction(spec, pathName), pathName)
  }
}

function traced(value, calls, enabled) {
  return enabled ? { result: value, calls } : value
}

async function invokeResolver(renderer, args, asynchronous) {
  const calls = []
  const resolver = makeResolver(args.resolver, asynchronous, calls)
  const scope = decoded(args.scope, {})
  const value = asynchronous
    ? await renderer.renderFnAsync(resolver, scope)
    : renderer.renderFn(resolver, scope)
  return traced(value, calls, args.trace === true)
}

async function runSequence(renderer, steps) {
  if (!Array.isArray(steps)) throw new TypeError('steps must be an array')
  const results = []
  for (const step of steps) {
    requireRecord(step, 'sequence step')
    try {
      let value
      switch (step.op) {
        case 'render':
          value = renderer.render(decoded(step.data, {}))
          break
        case 'renderFn':
          value = await invokeResolver(renderer, step, false)
          break
        case 'renderFnAsync':
          value = await invokeResolver(renderer, step, true)
          break
        default:
          throw new RangeError(`Unsupported sequence step: ${String(step.op)}`)
      }
      results.push({ ok: true, value })
    } catch {
      results.push({ ok: false })
    }
  }
  return results
}

async function invoke(op, args) {
  requireRecord(args, 'args')
  const upstream = loadUpstream()
  switch (op) {
    case 'render':
      return upstream.render(args.template, decoded(args.data, {}), args.options)
    case 'renderFn':
      return invokeResolver(upstream.compile(args.template, args.options), args, false)
    case 'renderFnAsync':
      return invokeResolver(upstream.compile(args.template, args.options), args, true)
    case 'compile':
      upstream.compile(args.template, args.options)
      return { kind: 'renderer' }
    case 'compile.render':
      return upstream.compile(args.template, args.options).render(decoded(args.data, {}))
    case 'compile.renderFn':
      return invokeResolver(upstream.compile(args.template, args.options), args, false)
    case 'compile.renderFnAsync':
      return invokeResolver(upstream.compile(args.template, args.options), args, true)
    case 'compile.sequence':
      return runSequence(upstream.compile(args.template, args.options), args.steps)
    case 'get':
      return upstream.get(decoded(args.scope, {}), args.path, args.options)
    case 'getRef':
      return upstream.getRef(decoded(args.scope, {}), args.ref, args.options)
    case 'tokenize':
      return upstream.tokenize(args.template, args.options)
    case 'renderer.render':
      return new upstream.Renderer(args.tokens, args.options).render(decoded(args.data, {}))
    case 'renderer.construct':
      new upstream.Renderer(args.tokens, args.options)
      return { kind: 'renderer' }
    case 'renderer.renderFn':
      return invokeResolver(new upstream.Renderer(args.tokens, args.options), args, false)
    case 'renderer.renderFnAsync':
      return invokeResolver(new upstream.Renderer(args.tokens, args.options), args, true)
    case 'renderer.sequence':
      return runSequence(new upstream.Renderer(args.tokens, args.options), args.steps)
    default:
      throw new RangeError(`Unsupported operation: ${String(op)}`)
  }
}

function errorResponse(id, error) {
  const name = error && typeof error.name === 'string' ? error.name : typeof error
  const message = error && typeof error.message === 'string' ? error.message : String(error)
  return { id, ok: false, error: { name, message } }
}

export async function handleRequest(request) {
  let id = null
  try {
    requireRecord(request, 'request')
    if (typeof request.id !== 'string' || request.id.length === 0) {
      throw new TypeError('request.id must be a non-empty string')
    }
    id = request.id
    if (typeof request.op !== 'string' || request.op.length === 0) {
      throw new TypeError('request.op must be a non-empty string')
    }
    const value = await invoke(request.op, request.args)
    return { id, ok: true, value: encode(value) }
  } catch (error) {
    return errorResponse(id, error)
  }
}

async function main() {
  const inputFlag = process.argv.indexOf('--input-file')
  const input = inputFlag === -1 ? process.stdin : createReadStream(process.argv[inputFlag + 1])
  const lines = createInterface({ input, crlfDelay: Infinity })

  for await (const line of lines) {
    if (line.trim() === '') continue
    let response
    try {
      response = await handleRequest(JSON.parse(line))
    } catch (error) {
      response = errorResponse(null, error)
    }
    process.stdout.write(`${JSON.stringify(response)}\n`)
  }
}

const entryUrl = process.argv[1] ? pathToFileURL(path.resolve(process.argv[1])).href : ''
if (entryUrl === import.meta.url) {
  main().catch((error) => {
    process.stderr.write(`${error && error.stack ? error.stack : String(error)}\n`)
    process.exitCode = 1
  })
}
