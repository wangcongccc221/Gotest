import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const toolsDir = path.dirname(fileURLToPath(import.meta.url))
const projectRoot = path.resolve(toolsDir, '..')

function readSource(...parts) {
  return fs.readFileSync(path.join(projectRoot, ...parts), 'utf8')
}

const homeStatsSource = readSource('go', 'ohos', 'Tcp', 'home_stats.go')
const realtimeRowsSource = readSource('go', 'ohos', 'Tcp', 'realtime_save_rows.go')
const timeUtilsSource = readSource('go', 'ohos', 'Tcp', 'time_utils.go')

assert.match(
  timeUtilsSource,
  /time\.FixedZone\("Asia\/Shanghai", 8\*60\*60\)/,
  'TCP China time must remain fixed at UTC+8'
)

assert.match(
  homeStatsSource,
  /runningDate := cTCPLocalTime\(now\)\.Format\("2006-01-02 15:04"\)/,
  'Realtime homeStats process points must format RunningDate in Asia/Shanghai'
)

assert.match(
  realtimeRowsSource,
  /RunningDate:\s+cTCPLocalTime\(now\)\.Format\("2006-01-02 15:04"\)/,
  'Persisted process points must format RunningDate in Asia/Shanghai'
)

const utcYearEnd = Date.UTC(2026, 11, 31, 18, 30)
const chinaYearEnd = new Date(utcYearEnd + 8 * 60 * 60 * 1000)
assert.equal(chinaYearEnd.getUTCFullYear(), 2027, 'UTC+8 conversion should cross into the next year')
assert.equal(chinaYearEnd.getUTCMonth(), 0, 'UTC+8 conversion should cross into January')
assert.equal(chinaYearEnd.getUTCDate(), 1, 'UTC+8 conversion should cross into the next day')
assert.equal(chinaYearEnd.getUTCHours(), 2, 'UTC 18:30 should become China 02:30')

console.log('Processing curve China time checks passed.')
