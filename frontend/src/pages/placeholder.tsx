export function Placeholder({ title }: { title: string }) {
  return (
    <div className="card">
      <h1 className="fs20 b">{title}</h1>
      <p className="mut mt8">Coming soon.</p>
    </div>
  );
}
