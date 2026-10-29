export function validateQty(raw: string): string | null {
    const value = raw.trim()
    if (value === '') return 'Qty is required'
    if (!/^-?\d+$/.test(value)) return 'Qty must be a whole number'
    if (Number(value) <= 0) return 'Qty must be greater than 0'
    return null
}
