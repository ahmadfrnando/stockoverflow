import { useEffect, useState } from "react";
import { api } from "../api/client";
import type { ProductSummary } from "../api/types";
import { formatRupiah } from "../lib/format";

export function SummaryCards({ reloadKey }: { reloadKey: number }) {
  const [summary, setSummary] = useState<ProductSummary | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    api
      .productSummary()
      .then((result) => {
        if (!cancelled) {
          setSummary(result);
          setError(null);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled)
          setError(err instanceof Error ? err.message : "Unknown error");
      });
    return () => {
      cancelled = true;
    };
  }, [reloadKey]);

  if (error) {
    return (
      <div role="alert" className="error">
        Failed to load summary: {error}
      </div>
    );
  }
  if (!summary) return <p>Loading summary…</p>;

  return (
    <div className="cards">
      <Card label="Total products" value={String(summary.total_products)} />
      <Card label="Active products" value={String(summary.active_products)} />
      <Card
        label="Total stock value"
        value={formatRupiah(summary.total_stock_value)}
      />
      <Card label="Low stock (≤ 10)" value={String(summary.low_stock_count)} />
    </div>
  );
}

function Card({ label, value }: { label: string; value: string }) {
  return (
    <div className="card">
      <span className="card-label">{label}</span>
      <strong className="card-value">{value}</strong>
    </div>
  );
}
