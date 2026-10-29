import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { ApiError, api } from "../api/client";
import type { Order, Product } from "../api/types";
import { formatRupiah } from "../lib/format";
import {
  insufficientStockErrors,
  toOrderInput,
  validateOrderForm,
} from "../lib/orderForm";
import type { FieldErrors, OrderRow } from "../lib/orderForm";

interface Props {
  onCreated: (order: Order) => void;
}

const emptyRow: OrderRow = { sku: "", qty: "1" };

export function NewOrderPage({ onCreated }: Props) {
  const [products, setProducts] = useState<Product[]>([]);
  const [customer, setCustomer] = useState("");
  const [rows, setRows] = useState<OrderRow[]>([emptyRow]);
  const [errors, setErrors] = useState<FieldErrors>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api
      .listProducts({ status: "active", limit: 100 })
      .then((result) => setProducts(result.data))
      .catch(() => setFormError("Could not load products"));
  }, []);

  function updateRow(index: number, patch: Partial<OrderRow>) {
    setRows((current) =>
      current.map((row, i) => (i === index ? { ...row, ...patch } : row)),
    );
  }

  function addRow() {
    setRows((current) => [...current, emptyRow]);
  }

  function removeRow(index: number) {
    setRows((current) => current.filter((_, i) => i !== index));
    setErrors({});
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError(null);

    const localErrors = validateOrderForm(customer, rows);
    setErrors(localErrors);
    if (Object.keys(localErrors).length > 0) return;

    setSaving(true);
    try {
      const order = await api.createOrder(toOrderInput(customer, rows));
      onCreated(order);
    } catch (err) {
      if (
        err instanceof ApiError &&
        err.body.code === "INSUFFICIENT_STOCK" &&
        err.body.sku
      ) {
        setErrors(
          insufficientStockErrors(rows, err.body.sku, err.body.available ?? 0),
        );
      } else if (err instanceof ApiError && err.body.fields) {
        setErrors(err.body.fields);
      } else {
        setFormError(err instanceof Error ? err.message : "Unexpected error");
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <section>
      <h1>New order</h1>
      <form onSubmit={handleSubmit} noValidate>
        <label className="field">
          Customer
          <input
            value={customer}
            onChange={(e) => setCustomer(e.target.value)}
          />
        </label>
        {errors.customer && <p className="field-error">{errors.customer}</p>}

        <h2>Items</h2>
        {errors.items && <p className="field-error">{errors.items}</p>}

        {rows.map((row, i) => {
          const skuError = errors[`items[${i}].sku`];
          const qtyError = errors[`items[${i}].qty`];
          return (
            <div
              key={i}
              className={
                skuError || qtyError ? "order-row has-error" : "order-row"
              }
            >
              <select
                value={row.sku}
                onChange={(e) => updateRow(i, { sku: e.target.value })}
                aria-label="Product"
              >
                <option value="">Select product…</option>
                {products.map((p) => (
                  <option key={p.id} value={p.sku}>
                    {p.sku} — {p.name} ({formatRupiah(p.price)}, stock {p.stock}
                    )
                  </option>
                ))}
              </select>
              <input
                inputMode="numeric"
                value={row.qty}
                onChange={(e) => updateRow(i, { qty: e.target.value })}
                aria-label="Qty"
              />
              <button
                type="button"
                onClick={() => removeRow(i)}
                disabled={rows.length === 1}
              >
                Remove
              </button>
              {skuError && <p className="field-error">{skuError}</p>}
              {qtyError && <p className="field-error">{qtyError}</p>}
            </div>
          );
        })}

        <button type="button" onClick={addRow}>
          + Add item
        </button>

        {formError && (
          <div role="alert" className="error">
            {formError}
          </div>
        )}

        <div className="modal-actions">
          <button type="submit" disabled={saving}>
            {saving ? "Creating…" : "Create order"}
          </button>
        </div>
      </form>
    </section>
  );
}
