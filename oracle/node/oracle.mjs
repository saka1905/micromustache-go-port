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

function resolveAction(spec, pathName) {
  requireRecord(spec, 'resolver')
  const pathActions = spec.paths === undefined ? {} : requireRecord(spec.paths, 'resolver.paths')
  const action = Object.prototype.hasOwnProperty.call(pathActions, pathName)
    ? pathActions[pathName]
    : spec.default
  requireRecord(action, `resolver action for ${JSON.stringify(pathName)}`)

  switch (action.action) {
    case 'value':
      return decode(action.value)
    case 'undefined':
      return undefined
    case 'error':
      throw namedError(action.error)
    default:
      throw new TypeError(`Unknown resolver action for ${JSON.stringify(pathName)}`)
  }
}

function makeResolver(spec, asynchronous) {
  if (asynchronous) {
    return async (pathName) => resolveAction(spec, pathName)
  }
  return (pathName) => resolveAction(spec, pathName)
}

async function invoke(op, args) {
  requireRecord(args, 'args')
  const upstream = loadUpstream()
  switch (op) {
    case 'render':
      return upstream.render(args.template, decoded(args.data, {}), args.options)
    case 'renderFn':
      return upstream.renderFn(
        args.template,
        makeResolver(args.resolver, false),
        decoded(args.scope, {}),
        args.options
      )
    case 'renderFnAsync':
      return upstream.renderFnAsync(
        args.template,
        makeResolver(args.resolver, true),
        decoded(args.scope, {}),
        args.options
      )
    case 'compile.render':
      return upstream.compile(args.template, args.options).render(decoded(args.data, {}))
    case 'compile.renderFn':
      return upstream
        .compile(args.template, args.options)
        .renderFn(makeResolver(args.resolver, false), decoded(args.scope, {}))
    case 'compile.renderFnAsync':
      return upstream
        .compile(args.template, args.options)
        .renderFnAsync(makeResolver(args.resolver, true), decoded(args.scope, {}))
    case 'get':
      return upstream.get(decoded(args.scope, {}), args.path, args.options)
    case 'getRef':
      return upstream.getRef(decoded(args.scope, {}), args.ref, args.options)
    case 'tokenize':
      return upstream.tokenize(args.template, args.options)
    case 'renderer.render':
      return new upstream.Renderer(args.tokens, args.options).render(decoded(args.data, {}))
    case 'renderer.renderFn':
      return new upstream.Renderer(args.tokens, args.options).renderFn(
        makeResolver(args.resolver, false),
        decoded(args.scope, {})
      )
    case 'renderer.renderFnAsync':
      return new upstream.Renderer(args.tokens, args.options).renderFnAsync(
        makeResolver(args.resolver, true),
        decoded(args.scope, {})
      )
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
