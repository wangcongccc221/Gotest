import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const projectRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const clientSource = fs.readFileSync(
  path.join(projectRoot, 'go/ohos/Tcp/ctcpclient.go'),
  'utf8'
)
const webSocketSource = fs.readFileSync(
  path.join(projectRoot, 'go/ohos/Tcp/websocket.go'),
  'utf8'
)

assert.match(clientSource, /cTCPHCSaveParas\s*=\s*int32\(0x0018\)/)
assert.match(webSocketSource, /case "saveParasToFlash":/)
assert.match(
  webSocketSource,
  /handleSimpleFSMCommand\("saveParasToFlash", cTCPHCSaveParas, control\)/
)
assert.match(
  webSocketSource,
  /sendCommandAckDetail\(\s*topic,\s*commandID,\s*destID,\s*payloadBytes,\s*result,\s*commandAckMessage\(result\),\s*control\.RequestID,\s*\)/
)

console.log('project config backend source checks passed')
