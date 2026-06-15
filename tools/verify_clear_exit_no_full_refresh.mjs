import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const toolsDir = path.dirname(fileURLToPath(import.meta.url))
const projectRoot = path.resolve(toolsDir, '..')

function extractFunction(source, signature) {
  const start = source.indexOf(signature)
  assert.notEqual(start, -1, `${signature} should exist`)
  const brace = source.indexOf('{', start)
  assert.notEqual(brace, -1, `${signature} should have a body`)
  let depth = 0
  for (let index = brace; index < source.length; index++) {
    const char = source[index]
    if (char === '{') {
      depth++
    } else if (char === '}') {
      depth--
      if (depth === 0) {
        return source.slice(start, index + 1)
      }
    }
  }
  throw new Error(`${signature} body is not closed`)
}

const websocketSource = fs.readFileSync(
  path.join(projectRoot, 'go', 'ohos', 'Tcp', 'websocket.go'),
  'utf8'
)
const clearBody = extractFunction(
  websocketSource,
  'func ClearGradeExitData(control webSocketControlMessage)'
)
const clearHandlerBody = extractFunction(
  websocketSource,
  'func (c *webSocketClient) handleClearGradeExitData(control webSocketControlMessage)'
)

assert.match(
  clearBody,
  /cacheLatestGradeInfo\(destID,\s*grade\)/,
  'Cleared grade mappings must remain in the backend cache'
)
assert.doesNotMatch(
  clearBody,
  /requestStGlobalAfterConfigCommand/,
  'Clearing outlet mappings must not trigger repeated full StGlobal refreshes'
)
assert.doesNotMatch(
  clearBody,
  /refresh StGlobal scheduled/,
  'Clear success logs must not claim a full refresh was scheduled'
)
assert.match(
  clearHandlerBody,
  /sendCommandAckDetail\(/,
  'Clear-exit must use the acknowledgement helper that supports requestId'
)
assert.match(
  clearHandlerBody,
  /control\.RequestID/,
  'Clear-exit acknowledgement must echo the frontend requestId'
)
assert.doesNotMatch(
  clearHandlerBody,
  /sendCommandAck\(/,
  'Clear-exit must not use an acknowledgement helper that drops requestId'
)

console.log('Backend clear-exit refresh checks passed.')
