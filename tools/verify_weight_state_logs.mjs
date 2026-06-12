import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const projectRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const server = fs.readFileSync(path.join(projectRoot, 'go/ohos/Tcp/ctcpserver.go'), 'utf8')
const realtimeSave = fs.readFileSync(path.join(projectRoot, 'go/ohos/Tcp/realtime_save.go'), 'utf8')

assert.match(server, /\[WAM_WEIGHT_STATE\].*channel=.*state=/)
assert.doesNotMatch(server, /CTCP StWeightResult 回推: remote=/)
assert.doesNotMatch(server, /appendPayloadHexChunks\("CTCP StWeightResult 回推"/)
assert.match(
  server,
  /head\.NCmdId != cmdFSMStatistics[\s\S]{0,180}head\.NCmdId != cmdWAMWeightInfo/
)
assert.doesNotMatch(realtimeSave, /CTCP realtime save ok:/)
assert.match(realtimeSave, /CTCP realtime save failed:/)
assert.match(realtimeSave, /_, err := database\.SaveRealtimeFruitInfo\(input\)/)

console.log('weight state backend log checks passed')
