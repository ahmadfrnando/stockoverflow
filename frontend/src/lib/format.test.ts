import { describe, expect, it } from 'vitest'
import { formatRupiah, timeAgo } from './format'

describe('formatRupiah', () => {
    it('uses dot thousand separators', () => {
        expect(formatRupiah(1234567)).toBe('Rp 1.234.567')
    })

    it('format zero', () => {
        expect(formatRupiah(0)).toBe('Rp 0')
    })
})

describe('timeAgo', () => {
    const now = new Date('2026-10-29T12:00:00Z')

    it('returns a dash where there is no movement', () => {
        expect(timeAgo(null, now)).toBe('-')
    })

    it('returns just now under a minute', () => {
        expect(timeAgo('2026-10-29T11:59:30Z', now)).toBe('just now')
    })

    it('returns minutes', () => {
        expect(timeAgo('2026-10-29T11:45:00Z', now)).toBe('15m ago')
    })

    it('returns hours', () => {
        expect(timeAgo('2026-10-29T10:00:00Z', now)).toBe('2h ago')
    })

    it('returns days', () => {
        expect(timeAgo('2026-10-20T12:00:00Z', now)).toBe('9d ago')
    })
})