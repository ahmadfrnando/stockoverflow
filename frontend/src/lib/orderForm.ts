import type { CreateOrderInput } from '../api/types'
import { validateQty } from './validation'

export interface OrderRow {
    sku: string
    qty: string
}

export type FieldErrors = Record<string, string>

function normalizeSku(sku: string): string {
    return sku.trim().toUpperCase()
}

export function validateOrderForm(customer: string, rows: OrderRow[]): FieldErrors {
    const errors: FieldErrors = {}
    if (customer.trim() === '') errors.customer = 'Customer is required'
    if (rows.length === 0) errors.items = 'Add at least one item'

    const seen = new Set<string>()
    rows.forEach((row, i) => {
        const sku = normalizeSku(row.sku)
        if (sku === '') errors[`items[${i}].sku`] = 'SKU is required'
        else if (seen.has(sku)) errors[`items[${i}].sku`] = 'Duplicate SKU'
        seen.add(sku)

        const qtyError = validateQty(row.qty)
        if (qtyError) errors[`items[${i}].qty`] = qtyError
    })
    return errors
}

export function toOrderInput(customer: string, rows: OrderRow[]): CreateOrderInput {
    return {
        customer: customer.trim(),
        items: rows.map((row) => ({ sku: normalizeSku(row.sku), qty: Number(row.qty) })),
    }
}

export function insufficientStockErrors(rows: OrderRow[], sku: string, available: number): FieldErrors {
    const index = rows.findIndex((row) => normalizeSku(row.sku) === sku)
    const key = index >= 0 ? `items[${index}].qty` : 'items'
    return { [key]: `Insufficient stock for ${sku}: only ${available} available` }
}
