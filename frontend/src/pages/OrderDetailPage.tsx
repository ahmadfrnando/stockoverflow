import { useEffect, useState } from "react";
import type { Order } from "../api/types";
import { ApiError, api } from "../api/client";
import { formatRupiah, timeAgo } from "../lib/format";

type State =
  | { status: "loading" }
  | { status: "error"; message: string }
  | { status: "ready"; order: Order };

export function OrderDetailPage({ id }: { id: number }) {
  const [state, setState] = useState<State>({ status: "loading" });
  const [cancelling, setCancelling] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);


  useEffect(() => {
    let cancelled = false;
    api
      .getOrder(id)
      .then((order) => {
        if (!cancelled) setState({ status: "ready", order });
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
  }, [id]);

  async function handleCancel() {
    if (!window.confirm(`Cancel order #${id}? Stock will be restored.`)) return;

    setCancelling(true);
    setActionError(null);
    try {
      const order = await api.cancelOrder(id);
      setState({ status: "ready", order });
    } catch (err) {
      if (err instanceof ApiError && err.body.code === "ALREADY_CANCELLED") {
        setActionError("This order is already cancelled.");
      } else {
        setActionError(err instanceof Error ? err.message : "Unexpected error");
      }
    } finally {
      setCancelling(false);
    }
  }


  if (state.status === "loading") return <p>Loading order…</p>;
  if (state.status === "error") {
    return (
      <div role="alert" className="error">
        Failed to load order: {state.message}
      </div>
    );
  }

  const { order } = state;
  return (
    <section>
      <h1>Order #{order.id}</h1>
      <p>
        Customer: <strong>{order.customer}</strong> · Status:{" "}
        <span
          className={
            order.status === "CREATED" ? "badge active" : "badge inactive"
          }
        >
          {order.status}
        </span>
      </p>

      <table>
        <thead>
          <tr>
            <th>SKU</th>
            <th>Qty</th>
            <th>Unit price</th>
            <th>Subtotal</th>
          </tr>
        </thead>
        <tbody>
          {order.items.map((item) => (
            <tr key={item.id}>
              <td>{item.sku}</td>
              <td>{item.qty}</td>
              <td>{formatRupiah(item.unit_price)}</td>
              <td>{formatRupiah(item.unit_price * item.qty)}</td>
            </tr>
          ))}
        </tbody>
        <tfoot>
          <tr>
            <th colSpan={3}>Total</th>
            <th>{formatRupiah(order.total)}</th>
          </tr>
        </tfoot>
      </table>
      {order.cancelled_at && <p>Cancelled {timeAgo(order.cancelled_at)}.</p>}

      {actionError && (
        <div role="alert" className="error">
          {actionError}
        </div>
      )}

      {order.status === "CREATED" && (
        <div className="modal-actions">
          <button
            className="danger"
            onClick={handleCancel}
            disabled={cancelling}
          >
            {cancelling ? "Cancelling…" : "Cancel order"}
          </button>
        </div>
      )}
    </section>
  );
}
