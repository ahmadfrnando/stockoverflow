import { describe, expect, it } from 'vitest'
import { validateQty } from './validation'

describe('validateQty', () => {
    it('accepts a positive integer', () => {
        expect(validateQty('5')).toBeNull()
        expect(validateQty(' 12 ')).toBeNull()
    })

    it('requires a value', () => {
        expect(validateQty('')).toBe('Qty is required')
        expect(validateQty('   ')).toBe('Qty is required')
    })

    it('rejects non-integers', () => {
        expect(validateQty('2.5')).toBe('Qty must be a whole number')
        expect(validateQty('abc')).toBe('Qty must be a whole number')
    })

    it('rejects zero and negatives', () => {
        expect(validateQty('0')).toBe('Qty must be greater than 0')
        expect(validateQty('-3')).toBe('Qty must be greater than 0')
    })
})
