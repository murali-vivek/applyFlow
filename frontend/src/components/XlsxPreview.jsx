export default function XlsxPreview({ fileName, rowCount, preview }) {
  if (!preview?.length) return null;

  return (
    <div className="xlsx-preview">
      <div className="xlsx-preview-header">
        <strong>{fileName}</strong>
        <span className="muted"> · {rowCount} row{rowCount !== 1 ? 's' : ''} total · showing first {preview.length}</span>
      </div>
      <div className="xlsx-preview-table-wrap">
        <table className="xlsx-preview-table">
          <thead>
            <tr>
              <th>Company Name</th>
              <th>Role</th>
              <th>Company Mail</th>
            </tr>
          </thead>
          <tbody>
            {preview.map((row, i) => (
              <tr key={i}>
                <td>{row.companyName}</td>
                <td>{row.role}</td>
                <td>{row.companyMail}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
