import { createHash } from 'node:crypto'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)
const upstream = require('../upstream/dist/micromustache.cjs')
let sink

function argument(name, fallback) {
  const index = process.argv.indexOf(`--${name}`)
  return index === -1 ? fallback : process.argv[index + 1]
}

function sha256(value) {
  return createHash('sha256').update(value).digest('hex')
}

function countDataNodes(value) {
  if (value === null || value === undefined) return 0
  if (Array.isArray(value)) return 1 + value.reduce((sum, child) => sum + countDataNodes(child), 0)
  if (typeof value === 'object') return 1 + Object.values(value).reduce((sum, child) => sum + countDataNodes(child), 0)
  return 1
}

function metrics(workload) {
  let dataNodes = countDataNodes(workload.data)
  for (const variant of workload.dataVariants || []) dataNodes = Math.max(dataNodes, countDataNodes(variant))
  let paths = []
  if (workload.template) paths = upstream.tokenize(workload.template, workload.options).paths
  return {
    templateBytes: Buffer.byteLength(workload.template || '', 'utf8'),
    templateCharacters: Array.from(workload.template || '').length,
    interpolations: paths.length,
    pathCount: workload.template ? paths.length : (workload.api === 'get' || workload.api === 'getRef' ? 1 : 0),
    dataNodes,
  }
}

function stableEqual(left, right) {
  return JSON.stringify(left) === JSON.stringify(right)
}

function validateSuite(suite) {
  if (suite.schemaVersion !== 1 || !Array.isArray(suite.workloads)) throw new Error('invalid workload suite')
  const seen = new Set()
  const required = new Set(['tokenize', 'get', 'getRef', 'render', 'renderFn', 'renderFnAsync', 'compile', 'renderer.construct', 'renderer.render', 'renderer.renderFn', 'renderer.renderFnAsync'])
  for (const workload of suite.workloads) {
    if (!workload.id || !workload.api || !workload.category || !workload.size) throw new Error('workload identity fields are required')
    if (seen.has(workload.id)) throw new Error(`duplicate workload id ${workload.id}`)
    seen.add(workload.id)
    required.delete(workload.api)
    if (!stableEqual(metrics(workload), workload.metrics)) throw new Error(`workload metrics mismatch: ${workload.id}`)
    if (workload.expected?.mode !== 'success' || !Number.isInteger(workload.expected.resolverCalls) || workload.expected.resolverCalls < 0) throw new Error(`invalid expected result: ${workload.id}`)
    if (workload.timedSetup !== 'included' && workload.timedSetup !== 'excluded') throw new Error(`invalid timedSetup: ${workload.id}`)
  }
  if (required.size !== 0) throw new Error(`missing required API workloads: ${Array.from(required).join(',')}`)
}

function selectData(workload, iteration) {
  const variants = workload.dataVariants || []
  return variants.length === 0 ? (workload.data || {}) : variants[iteration % variants.length]
}

function prepare(workload) {
  const timedResolver = (path) => Object.hasOwn(workload.resolver || {}, path) ? workload.resolver[path] : undefined
  const timedAsyncResolver = async (path) => Object.hasOwn(workload.resolver || {}, path) ? workload.resolver[path] : undefined
  if (['renderer.render', 'renderer.renderFn', 'renderer.renderFnAsync'].includes(workload.api)) {
    return { workload, renderer: upstream.compile(workload.template, workload.options), timedResolver, timedAsyncResolver }
  }
  if (workload.api === 'renderer.construct') {
    return { workload, tokens: upstream.tokenize(workload.template, workload.options), timedResolver, timedAsyncResolver }
  }
  return { workload, timedResolver, timedAsyncResolver }
}

function executeValue(prepared, iteration, resolver, asyncResolver, validation) {
  const workload = prepared.workload
  const data = selectData(workload, iteration)
  let value
  switch (workload.api) {
    case 'tokenize': value = upstream.tokenize(workload.template, workload.options); break
    case 'get': value = upstream.get(data, workload.path, workload.options); break
    case 'getRef': value = upstream.getRef(data, workload.ref, workload.options); break
    case 'render': value = upstream.render(workload.template, data, workload.options); break
    case 'renderFn': value = upstream.renderFn(workload.template, resolver, data, workload.options); break
    case 'renderFnAsync': value = upstream.renderFnAsync(workload.template, asyncResolver, data, workload.options); break
    case 'compile': {
      const renderer = upstream.compile(workload.template, workload.options)
      value = validation ? renderer.render(data) : renderer
      break
    }
    case 'renderer.construct': {
      const renderer = new upstream.Renderer(prepared.tokens, workload.options)
      value = validation ? renderer.render(data) : renderer
      break
    }
    case 'renderer.render': value = prepared.renderer.render(data); break
    case 'renderer.renderFn': value = prepared.renderer.renderFn(resolver, data); break
    case 'renderer.renderFnAsync': value = prepared.renderer.renderFnAsync(asyncResolver, data); break
    default: throw new Error(`unsupported API ${workload.api}`)
  }
  return value
}

function normalize(value) {
  if (value && Array.isArray(value.strings) && Array.isArray(value.paths)) return { strings: value.strings, paths: value.paths }
  return value
}

async function validate(prepared) {
  let calls = 0
  const resolver = (path) => { calls++; return prepared.timedResolver(path) }
  const asyncResolver = async (path) => { calls++; return prepared.timedAsyncResolver(path) }
  const value = await executeValue(prepared, 0, resolver, asyncResolver, true)
  if (calls !== prepared.workload.expected.resolverCalls) throw new Error(`resolver calls=${calls} want=${prepared.workload.expected.resolverCalls}`)
  return { status: 'PASS', api: prepared.workload.api, resultDigest: sha256(JSON.stringify(normalize(value))), resolverCalls: calls }
}

function isAsync(workload) {
  return workload.api === 'renderFnAsync' || workload.api === 'renderer.renderFnAsync'
}

async function measure(prepared, iterations) {
  const start = process.hrtime.bigint()
  if (isAsync(prepared.workload)) {
    for (let iteration = 0; iteration < iterations; iteration++) {
      sink = await executeValue(prepared, iteration, prepared.timedResolver, prepared.timedAsyncResolver, false)
    }
  } else {
    for (let iteration = 0; iteration < iterations; iteration++) {
      sink = executeValue(prepared, iteration, prepared.timedResolver, prepared.timedAsyncResolver, false)
    }
  }
  return process.hrtime.bigint() - start
}

async function benchmark(prepared, config) {
  const minimum = BigInt(config.minDurationMs) * 1000000n
  let iterations = 1
  while (await measure(prepared, iterations) < minimum) {
    if (iterations > config.maxIterations / 2) throw new Error('calibration reached max iterations before minimum duration')
    iterations *= 2
  }
  for (let warmup = 0; warmup < config.warmup; warmup++) await measure(prepared, iterations)
  const samples = []
  while (samples.length < config.samples) {
    const elapsed = await measure(prepared, iterations)
    if (elapsed < minimum) {
      if (iterations > config.maxIterations / 2) throw new Error('measured duration below minimum at max iterations')
      iterations *= 2
      samples.length = 0
      continue
    }
    const elapsedNs = Number(elapsed)
    const nsPerOp = elapsedNs / iterations
    if (!(elapsedNs > 0) || !(nsPerOp > 0) || !Number.isFinite(nsPerOp)) throw new Error('invalid measured sample')
    samples.push({ iterations, elapsedNs, nsPerOp })
  }
  return samples
}

async function main() {
  const path = argument('workloads')
  const mode = argument('mode', 'validate')
  const config = {
    warmup: Number(argument('warmup', '3')),
    samples: Number(argument('samples', '7')),
    minDurationMs: Number(argument('min-duration-ms', '200')),
    maxIterations: Number(argument('max-iterations', '16777216')),
    processTimeoutSeconds: Number(argument('process-timeout-seconds', '300')),
  }
  if (!path) throw new Error('workloads is required')
  if (!['validate', 'benchmark'].includes(mode)) throw new Error(`unsupported mode ${mode}`)
  if (config.warmup < 3 || config.samples < 7 || config.minDurationMs <= 0 || config.maxIterations <= 0 || config.processTimeoutSeconds <= 0) throw new Error('invalid benchmark config')
  const workloadBytes = readFileSync(path)
  const suite = JSON.parse(workloadBytes.toString('utf8'))
  validateSuite(suite)
  const output = { schemaVersion: 1, mode, runtime: 'node', workloadSha256: sha256(workloadBytes), config, results: [] }
  for (const workload of [...suite.workloads].sort((left, right) => left.id.localeCompare(right.id))) {
    const prepared = prepare(workload)
    const validation = await validate(prepared)
    const result = { id: workload.id, api: workload.api, validation }
    if (mode === 'benchmark') {
      result.samples = await benchmark(prepared, config)
      result.sinkDigest = sha256(`${validation.resultDigest}:${typeof sink}`)
    }
    output.results.push(result)
  }
  process.stdout.write(`${JSON.stringify(output)}\n`)
}

main().catch((error) => {
  process.stderr.write(`${error && error.stack ? error.stack : String(error)}\n`)
  process.exitCode = 1
})
