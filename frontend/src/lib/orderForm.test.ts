import { describe, expect, it } from 'vitest'
import { insufficientStockErrors, toOrderInput, validateOrderForm } from './orderForm'

describe('validateOrderForm', () => {
    it('passes a valid form', () => {
        expect(validateOrderForm('Budi', [{ sku: 'SKU-001', qty: '2' }])).toEqual({})
    })

    it('requires customer and at least one item', () => {
        expect(validateOrderForm(' ', [])).toEqual({
            customer: 'Customer is required',
            items: 'Add at least one item',
        })
    })

    it('flags duplicate skus case-insensitively', () => {
        const errors = validateOrderForm('Budi', [
            { sku: 'SKU-001', qty: '1' },
            { sku: ' sku-001 ', qty: '1' },
        ])
        expect(errors['items[1].sku']).toBe('Duplicate SKU')
    })

    it('validates qty per row', () => {
        const errors = validateOrderForm('Budi', [{ sku: 'SKU-001', qty: '0' }])
        expect(errors['items[0].qty']).toBe('Qty must be greater than 0')
    })
})

describe('toOrderInput', () => {
    it('trims, uppercases and converts qty to number', () => {
        expect(toOrderInput(' Budi ', [{ sku: ' sku-001', qty: '3' }])).toEqual({
            customer: 'Budi',
            items: [{ sku: 'SKU-001', qty: 3 }],
        })
    })
})

describe('insufficientStockErrors', () => {
    it('points at the row holding the failing sku', () => {
        const rows = [
            { sku: 'SKU-001', qty: '1' },
            { sku: 'SKU-002', qty: '99' },
        ]
        expect(insufficientStockErrors(rows, 'SKU-002', 5)).toEqual({
            'items[1].qty': 'Insufficient stock for SKU-002: only 5 available',
        })
    })
})
