const VERSIONS = ['latest'];

export function VersionSelector() {
  return (
    <select aria-label="Version" defaultValue={VERSIONS[0]}>
      {VERSIONS.map((version) => (
        <option key={version} value={version}>
          {version}
        </option>
      ))}
    </select>
  );
}
