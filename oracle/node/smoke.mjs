import { readFileSync } from 'node:fs'
import { spawnSync } from 'node:child_process'
import path from 'node:path'
import process from 'node:process'

import { decode, encode } from './codec.mjs'

function option(name) {
  const index = process.argv.indexOf(name)
  if (index === -1 || index + 1 >= process.argv.length) throw new Error(`Missing ${name}`)
  return path.resolve(process.argv[index + 1])
}

function equivalent(left, right) {
  if (Object.is(left, right)) return true
  if (Array.isArray(left) && Array.isArray(right)) {
    return left.length === right.length && left.every((value, index) => equivalent(value, right[index]))
  }
  if (left && right && typeof left === 'object' && typeof right === 'object') {
    const leftKeys = Object.keys(left)
    const rightKeys = Object.keys(right)
    return (
      leftKeys.length === rightKeys.length &&
      leftKeys.every((key, index) => key === rightKeys[index] && equivalent(left[key], right[key]))
    )
  }
  return false
}

function verifyCodec() {
  const samples = [
    undefined,
    Number.NaN,
    Number.POSITIVE_INFINITY,
    Number.NEGATIVE_INFINITY,
    -0,
    null,
    false,
    42.5,
    'text',
    [undefined, Number.NaN],
    { nested: { value: Number.NEGATIVE_INFINITY } },
  ]
  for (const sample of samples) {
    const roundTrip = decode(JSON.parse(JSON.stringify(encode(sample))))
    if (!equivalent(sample, roundTrip)) throw new Error('Codec round-trip mismatch')
  }
}

function parseLines(text, label) {
  const lines = text.split(/\r?\n/u).filter((line) => line.trim() !== '')
  return lines.map((line, index) => {
    try {
      return JSON.parse(line)
    } catch (error) {
      throw new Error(`${label} line ${index + 1} is not valid JSON: ${error.message}`)
    }
  })
}

function runOracle(oraclePath, input) {
  const result = spawnSync(process.execPath, [oraclePath], {
    encoding: 'utf8',
    input,
    windowsHide: true,
  })
  if (result.error) throw result.error
  if (result.status !== 0) throw new Error(`Oracle exited ${result.status}: ${result.stderr}`)
  if (result.stderr !== '') throw new Error(`Unexpected oracle stderr: ${result.stderr}`)
  return result.stdout
}

verifyCodec()
const casesPath = option('--cases')
const oraclePath = option('--oracle')
const input = readFileSync(casesPath, 'utf8')
const requests = parseLines(input, 'request')
const ids = new Set()
for (const request of requests) {
  if (typeof request.id !== 'string' || ids.has(request.id)) throw new Error('Smoke ids must be unique strings')
  if (typeof request.expectOk !== 'boolean') throw new Error(`Smoke request ${request.id} needs expectOk`)
  ids.add(request.id)
}

const firstOutput = runOracle(oraclePath, input)
const secondOutput = runOracle(oraclePath, input)
if (firstOutput !== secondOutput) throw new Error('Smoke outputs differ between identical runs')

const responses = parseLines(firstOutput, 'response')
if (responses.length !== requests.length) throw new Error('Request and response counts differ')
for (let index = 0; index < requests.length; index += 1) {
  const request = requests[index]
  const response = responses[index]
  if (response.id !== request.id) throw new Error(`Response id mismatch at ${request.id}`)
  if (response.ok !== request.expectOk) throw new Error(`Unexpected ok status at ${request.id}`)
  if (response.ok) {
    decode(response.value)
  } else if (
    !response.error ||
    typeof response.error.name !== 'string' ||
    typeof response.error.message !== 'string'
  ) {
    throw new Error(`Invalid error response at ${request.id}`)
  }
}

process.stdout.write(`PASS smoke: ${requests.length} requests, codec round-trips, and two identical runs\n`)
