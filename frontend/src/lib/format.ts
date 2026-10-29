const rupiah = Intl.NumberFormat('id-ID')

export function formatRupiah(amount: number): string {
    return `Rp ${rupiah.format(amount)}`
}

export function timeAgo(iso: string | null, now: Date = new Date()): string {
    if (!iso) return '-'

    const seconds = Math.floor((now.getTime() - new Date(iso).getTime()) / 1000)
    if (seconds < 60) return 'just now'

    const minutes = Math.floor(seconds / 60)
    if (minutes < 60) return `${minutes}m ago`

    const hours = Math.floor(minutes / 60)
    if (hours < 60) return `${hours}h ago`

    return `${Math.floor(hours / 24)}d ago`
}