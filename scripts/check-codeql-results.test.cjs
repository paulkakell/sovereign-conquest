'use strict';

const assert = require('node:assert/strict');
const test = require('node:test');
const { checkSarif } = require('./check-codeql-results.cjs');

function fixture(score = '7.5') {
  return { version: '2.1.0', runs: [{
    tool: { driver: { name: 'CodeQL', rules: [{ id: 'go/path-injection', properties: { 'security-severity': score } }] } },
    results: [{ ruleId: 'go/path-injection', ruleIndex: 0, level: 'warning' }],
  }] };
}

test('empty results pass, missing analysis fails', () => {
  const report = fixture();
  report.runs[0].results = [];
  assert.deepEqual(checkSarif(report), { results: 0, highOrCritical: [] });
  for (const invalid of [{}, { version: '2.1.0', runs: [] }, { version: '2.1.0', runs: [{}] }]) {
    assert.throws(() => checkSarif(invalid));
  }
});

for (const score of ['7', '7.5', '9.8', '10.0']) {
  test(`blocks security score ${score} even at warning level`, () => {
    assert.equal(checkSarif(fixture(score)).highOrCritical.length, 1);
  });
}

test('lower severity and non-security diagnostics remain visible', () => {
  for (const score of ['0', '4.3', '6.9', undefined]) {
    const report = fixture();
    report.runs[0].tool.driver.rules[0].properties['security-severity'] = score;
    assert.deepEqual(checkSarif(report), { results: 1, highOrCritical: [] });
  }
});

test('suppression and baseline state cannot hide high findings', () => {
  const report = fixture();
  Object.assign(report.runs[0].results[0], {
    level: 'none', baselineState: 'unchanged', suppressions: [{ kind: 'inSource', status: 'accepted' }],
  });
  assert.equal(checkSarif(report).highOrCritical.length, 1);
});

test('all runs are checked', () => {
  const report = fixture('4.3');
  report.runs.push(fixture().runs[0]);
  assert.equal(checkSarif(report).highOrCritical.length, 1);
});

test('resolves CodeQL query-pack extension rules', () => {
  const report = fixture();
  const run = report.runs[0];
  run.tool.extensions = [{ name: 'codeql/go-queries', rules: run.tool.driver.rules }];
  run.tool.driver.rules = [];
  run.results = [{ ruleId: 'go/path-injection', rule: { id: 'go/path-injection', index: 0, toolComponent: { index: 0 } } }];
  assert.equal(checkSarif(report).highOrCritical.length, 1);
  run.results[0].rule.toolComponent.index = 99;
  assert.throws(() => checkSarif(report), /Unknown SARIF rule component/);
});

test('broken or unsuccessful reports fail closed', () => {
  for (const mutate of [
    run => { delete run.results; },
    run => { run.tool.driver.name = 'Other'; },
    run => { run.results[0].ruleId = 'unknown'; },
    run => { run.results[0].ruleIndex = 9; },
    run => { run.invocations = [{ executionSuccessful: false }]; },
    run => { run.tool.driver.rules[0].properties['security-severity'] = 'invalid'; },
    run => { run.tool.driver.rules[0].properties['security-severity'] = '11'; },
  ]) {
    const report = fixture();
    mutate(report.runs[0]);
    assert.throws(() => checkSarif(report));
  }
});
