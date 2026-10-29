import { useState } from "react";
import type { FormEvent } from "react";
import { ApiError, api } from "../api/client";
import type { AdjustStockResult, MovementType, Product } from "../api/types";
import { validateQty } from "../lib/validation";

interface Props {
product: Product;
onClose: () => void;
onSaved: (result: AdjustStockResult) => void;
}

function describeError(err: unknown): string {
if (err instanceof ApiError) {
    if (err.body.code === "INSUFFICIENT_STOCK") {
    return `Insufficient stock: only ${err.body.available ?? 0} available.`;
    }
    if (err.body.fields) {
    return Object.values(err.body.fields).join(", ");
    }
    return err.body.message ?? err.body.code;
}
return "Unexpected error, please try again.";
}

export function AdjustStockModal({ product, onClose, onSaved }: Props) {
const [type, setType] = useState<MovementType>("in");
const [qty, setQty] = useState("");
const [note, setNote] = useState("");
const [saving, setSaving] = useState(false);
const [qtyError, setQtyError] = useState<string | null>(null);
const [serverError, setServerError] = useState<string | null>(null);

async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const error = validateQty(qty);
    setQtyError(error);
    setServerError(null);
    if (error) return;

    setSaving(true);
    try {
    const result = await api.adjustStock(product.id, {
        type,
        qty: Number(qty),
        note: note.trim(),
    });
    onSaved(result);
    } catch (err) {
    setServerError(describeError(err));
    } finally {
    setSaving(false);
    }
}

return (
    <div className="modal-backdrop">
    <div
        className="modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="adjust-title"
    >
        <h2 id="adjust-title">Adjust stock — {product.sku}</h2>
        <p>
        {product.name} · current stock: <strong>{product.stock}</strong>
        </p>

        <form onSubmit={handleSubmit} noValidate>
        <div className="radio-group">
            <label>
            <input
                type="radio"
                name="type"
                checked={type === "in"}
                onChange={() => setType("in")}
            />{" "}
            Stock in
            </label>
            <label>
            <input
                type="radio"
                name="type"
                checked={type === "out"}
                onChange={() => setType("out")}
            />{" "}
            Stock out
            </label>
        </div>

        <label className="field">
            Qty
            <input
            inputMode="numeric"
            value={qty}
            onChange={(e) => setQty(e.target.value)}
            aria-invalid={qtyError !== null}
            />
        </label>
        {qtyError && <p className="field-error">{qtyError}</p>}

        <label className="field">
            Note
            <input
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder="optional"
            />
        </label>

        {serverError && (
            <div role="alert" className="error">
            {serverError}
            </div>
        )}

        <div className="modal-actions">
            <button type="button" onClick={onClose} disabled={saving}>
            Cancel
            </button>
            <button type="submit" disabled={saving}>
            {saving ? "Saving…" : "Save"}
            </button>
        </div>
        </form>
    </div>
    </div>
);
}
