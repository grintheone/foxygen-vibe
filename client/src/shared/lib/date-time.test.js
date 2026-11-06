import assert from "node:assert/strict";
import test from "node:test";
import {
  appDateTimeToISO, formatAppDate, formatAppDateTime, formatAppTime,
  parseTimestamp, toAppDateTimeValue, toAppDateValue,
} from "./date-time.js";

const originalTZ = process.env.TZ;
test.after(() => {
  if (originalTZ === undefined) delete process.env.TZ;
  else process.env.TZ = originalTZ;
});

test("timestamps display in Moscow regardless of the device timezone", () => {
  for (const zone of ["UTC", "Europe/Moscow", "Asia/Bangkok", "America/Los_Angeles"]) {
    process.env.TZ = zone;
    assert.equal(formatAppDateTime("2026-11-06T12:34:00Z"), "06.11.2026 15:34");
    assert.equal(formatAppDateTime("2026-11-06T15:34:00+03:00"), "06.11.2026 15:34");
    assert.equal(formatAppTime("2026-11-06T12:34:00"), "15:34");
    assert.equal(toAppDateTimeValue("2026-11-06T12:34:00Z"), "2026-11-06T15:34");
    assert.equal(appDateTimeToISO("2026-11-06T15:34"), "2026-11-06T12:34:00.000Z");
  }
});

test("calendar dates follow Moscow at day, month and year boundaries", () => {
  assert.equal(toAppDateValue("2026-12-31T21:30:00Z"), "2027-01-01");
  assert.equal(formatAppDate("2026-10-31T21:30:00Z"), "01.11.2026");
  assert.equal(formatAppTime("2026-11-06T21:00:00Z"), "00:00");
  assert.equal(formatAppDate("2026-11-06T21:00:00Z", { shortYear: true }), "07.11.26");
});

test("empty and invalid values do not produce timestamps", () => {
  assert.equal(formatAppDateTime(null), null);
  assert.equal(formatAppDateTime("invalid"), null);
  assert.equal(toAppDateValue("invalid"), "");
  assert.equal(appDateTimeToISO(""), "");
  assert.equal(appDateTimeToISO("invalid"), null);
  assert.equal(appDateTimeToISO("2026-11-06T15:34Z"), null);
  assert.ok(Number.isNaN(parseTimestamp(undefined).getTime()));
});
