import { useEffect, useState } from "react";
import { api } from "../api/client";
import type { Product, ProductList, ProductFilter } from "../api/types";
import { formatRupiah, timeAgo } from "../lib/format";
import { useDebounce } from "../lib/useDebounce";

type State =
  | { status: "loading" }
  | { status: "error"; message: string }
  | { status: "ready"; result: ProductList };

export function ProductsPage() {
  const [state, setState] = useState<State>({ status: "loading" });
  const [reloadKey, setReloadKey] = useState(0);
  const [search, setSearch] = useState("");
  const [filter, setFilter] = useState<ProductFilter>("all");
  const debouncedSearch = useDebounce(search);

  useEffect(() => {
    let cancelled = false;
    api
      .listProducts({ search: debouncedSearch, status: filter, limit: 20 })
      .then((result) => {
        if (!cancelled) setState({ status: "ready", result });
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setState({
            status: "error",
            message: err instanceof Error ? err.message : "Unknown error",
          });
        }
      });
    return () => {
      cancelled = true;
    };
  }, [debouncedSearch, filter, reloadKey]);

  function retry() {
    setState({ status: "loading" });
    setReloadKey((k) => k + 1);
  }

  return (
    <section>
      <h1>Products</h1>
      <div className="toolbar">
        <input
          type="search"
          placeholder="Search SKU or name…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <select
          value={filter}
          onChange={(e) => setFilter(e.target.value as ProductFilter)}
        >
          <option value="all">All</option>
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
          <option value="low_stock">Low stock (≤ 10)</option>
        </select>
      </div>
      {state.status === "loading" && <p>Loading products…</p>}

      {state.status === "error" && (
        <div role="alert" className="error">
          <p>Failed to load products: {state.message}</p>
          <button onClick={retry}>Retry</button>
        </div>
      )}

      {state.status === "ready" && state.result.data.length === 0 && (
        <p>No products match your search or filter.</p>
      )}

      {state.status === "ready" && state.result.data.length > 0 && (
        <ProductTable products={state.result.data} />
      )}
    </section>
  );
}

function ProductTable({ products }: { products: Product[] }) {
  return (
    <table>
      <thead>
        <tr>
          <th>SKU</th>
          <th>Name</th>
          <th>Price</th>
          <th>Stock</th>
          <th>Status</th>
          <th>Last movement</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        {products.map((p) => (
          <tr key={p.id}>
            <td>{p.sku}</td>
            <td>{p.name}</td>
            <td>{formatRupiah(p.price)}</td>
            <td>{p.stock}</td>
            <td>
              <span className={p.is_active ? "badge active" : "badge inactive"}>
                {p.is_active ? "Active" : "Inactive"}
              </span>
            </td>
            <td title={p.last_movement_at ?? ""}>
              {timeAgo(p.last_movement_at)}
            </td>
            <td>{/* Adjust stock button comes in step 5 */}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
