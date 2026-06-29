export interface SystemMetricsProps {
  metrics: Record<string, number>;
}

export function SystemMetrics({ metrics }: SystemMetricsProps) {
  return (
    <dl className="grid">
      {Object.entries(metrics).map(([key, value]) => (
        <div key={key}>
          <dt>{key}</dt>
          <dd>{value}</dd>
        </div>
      ))}
    </dl>
  );
}
