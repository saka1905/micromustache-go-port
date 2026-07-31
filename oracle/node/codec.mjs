function fail(message) {
  throw new TypeError(`Invalid encoded value: ${message}`)
}

function requireObject(node) {
  if (node === null || typeof node !== 'object' || Array.isArray(node)) {
    fail('expected an object envelope')
  }
}

function requireOnlyKeys(node, allowed) {
  for (const key of Object.keys(node)) {
    if (!allowed.includes(key)) {
      fail(`unexpected field ${JSON.stringify(key)}`)
    }
  }
}

export function decode(node) {
  requireObject(node)
  if (typeof node.type !== 'string') {
    fail('type must be a string')
  }

  switch (node.type) {
    case 'undefined':
      requireOnlyKeys(node, ['type'])
      return undefined
    case 'null':
      requireOnlyKeys(node, ['type'])
      return null
    case 'boolean':
      requireOnlyKeys(node, ['type', 'value'])
      if (typeof node.value !== 'boolean') fail('boolean value must be a boolean')
      return node.value
    case 'number':
      requireOnlyKeys(node, ['type', 'value'])
      if (typeof node.value !== 'number' || !Number.isFinite(node.value)) {
        fail('number value must be a finite JSON number')
      }
      return node.value
    case 'nan':
      requireOnlyKeys(node, ['type'])
      return Number.NaN
    case 'infinity':
      requireOnlyKeys(node, ['type'])
      return Number.POSITIVE_INFINITY
    case 'negative-infinity':
      requireOnlyKeys(node, ['type'])
      return Number.NEGATIVE_INFINITY
    case 'negative-zero':
      requireOnlyKeys(node, ['type'])
      return -0
    case 'string':
      requireOnlyKeys(node, ['type', 'value'])
      if (typeof node.value !== 'string') fail('string value must be a string')
      return node.value
    case 'array':
      requireOnlyKeys(node, ['type', 'value'])
      if (!Array.isArray(node.value)) fail('array value must be an array')
      return node.value.map(decode)
    case 'object': {
      requireOnlyKeys(node, ['type', 'value'])
      requireObject(node.value)
      const result = {}
      for (const key of Object.keys(node.value)) {
        Object.defineProperty(result, key, {
          configurable: true,
          enumerable: true,
          value: decode(node.value[key]),
          writable: true,
        })
      }
      return result
    }
    default:
      fail(`unknown type ${JSON.stringify(node.type)}`)
  }
}

export function encode(value) {
  if (value === undefined) return { type: 'undefined' }
  if (value === null) return { type: 'null' }

  switch (typeof value) {
    case 'boolean':
      return { type: 'boolean', value }
    case 'number':
      if (Number.isNaN(value)) return { type: 'nan' }
      if (value === Number.POSITIVE_INFINITY) return { type: 'infinity' }
      if (value === Number.NEGATIVE_INFINITY) return { type: 'negative-infinity' }
      if (Object.is(value, -0)) return { type: 'negative-zero' }
      return { type: 'number', value }
    case 'string':
      return { type: 'string', value }
    case 'object': {
      if (Array.isArray(value)) {
        return { type: 'array', value: value.map(encode) }
      }
      const prototype = Object.getPrototypeOf(value)
      if (prototype !== Object.prototype && prototype !== null) {
        throw new TypeError('Only plain objects can be encoded')
      }
      const encoded = Object.create(null)
      for (const key of Object.keys(value)) encoded[key] = encode(value[key])
      return { type: 'object', value: encoded }
    }
    default:
      throw new TypeError(`Unsupported JavaScript value type: ${typeof value}`)
  }
}
