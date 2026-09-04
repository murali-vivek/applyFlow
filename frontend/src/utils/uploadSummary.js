export function buildUploadSummary(result) {
  const lines = [];
  const skipped = result.skipped || {};

  if (skipped.missingCompanyName) {
    lines.push(`${skipped.missingCompanyName} row(s) skipped — missing Company Name`);
  }
  if (skipped.missingRole) {
    lines.push(`${skipped.missingRole} row(s) skipped — missing Role`);
  }
  if (skipped.missingEmail) {
    lines.push(`${skipped.missingEmail} row(s) skipped — missing Company Mail`);
  }
  if (skipped.invalidEmail) {
    lines.push(`${skipped.invalidEmail} row(s) skipped — invalid email`);
  }
  if (skipped.duplicateEmail) {
    lines.push(`${skipped.duplicateEmail} duplicate email(s) removed`);
  }
  if (skipped.emptyRows) {
    lines.push(`${skipped.emptyRows} empty row(s) ignored`);
  }

  const companyLabel = result.rowCount === 1 ? 'company is' : 'companies are';
  lines.push(`${result.rowCount} ${companyLabel} ready for outreach.`);

  return lines.join('\n');
}

export function hasUploadWarnings(result) {
  const skipped = result.skipped || {};
  return (
    skipped.missingCompanyName > 0 ||
    skipped.missingRole > 0 ||
    skipped.missingEmail > 0 ||
    skipped.invalidEmail > 0 ||
    skipped.duplicateEmail > 0 ||
    skipped.emptyRows > 0
  );
}
