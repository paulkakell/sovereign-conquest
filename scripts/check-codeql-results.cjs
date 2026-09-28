'use strict';

const fs = require('node:fs');

// CodeQL upload success does not mean that the analysis found no vulnerabilities.
// Gate the actual SARIF results, including existing and suppressed findings.
function checkSarif(document) {
  if (document.version !== '2.1.0' || !Array.isArray(document.runs) || !document.runs.length) {
    throw new Error('Expected a nonempty SARIF 2.1.0 analysis');
  }
  const findings = [];
  let total = 0;
  for (const run of document.runs) {
    const driver = run.tool?.driver;
    if (driver?.name !== 'CodeQL' || !Array.isArray(driver.rules) || !Array.isArray(run.results)) {
      throw new Error('Expected CodeQL rules and results');
    }
    if (run.invocations?.some(invocation => invocation.executionSuccessful === false)) {
      throw new Error('CodeQL analysis did not complete successfully');
    }
    for (const result of run.results) {
      const reference = result.rule;
      const componentReference = reference?.toolComponent;
      const component = componentReference === undefined ? driver :
        (Number.isInteger(componentReference.index) ? run.tool.extensions?.[componentReference.index] : undefined);
      if (!Array.isArray(component?.rules)) throw new Error('Unknown SARIF rule component');
      const ruleID = result.ruleId ?? reference?.id;
      const ruleIndex = result.ruleIndex ?? reference?.index;
      const rule = component.rules.find(candidate => candidate.id === ruleID);
      if (!rule || (reference?.id !== undefined && reference.id !== ruleID) ||
          (ruleIndex !== undefined && component.rules[ruleIndex] !== rule)) {
        throw new Error('SARIF result references an unknown or inconsistent rule');
      }
      total++;
      const score = rule.properties?.['security-severity'];
      if (score === undefined) continue; // Non-security diagnostic rule.
      if (!/^(?:[0-9](?:\.[0-9]+)?|10(?:\.0+)?)$/.test(String(score))) {
        throw new Error('Invalid CodeQL security severity');
      }
      if (Number(score) >= 7) {
        findings.push({
          rule: rule.id, severity: Number(score),
          locations: result.locations?.map(location => location.physicalLocation) || [],
        });
      }
    }
  }
  return { results: total, highOrCritical: findings };
}

if (require.main === module) {
  try {
    if (process.argv.length < 3) throw new Error('Supply at least one CodeQL SARIF file');
    for (const filename of process.argv.slice(2)) {
      const report = checkSarif(JSON.parse(fs.readFileSync(filename, 'utf8')));
      console.log(JSON.stringify({ file: filename, ...report }));
      if (report.highOrCritical.length) process.exitCode = 1;
    }
  } catch (error) {
    console.error(`CodeQL gate failed: ${error.message}`);
    process.exitCode = 1;
  }
}

module.exports = { checkSarif };
