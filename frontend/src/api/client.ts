import type {
    AdjustStockInput,
    AdjustStockResult,
    ApiErrorBody,
    CreateOrderInput,
    DataResponse,
    Order,
    ProductList,
    StockMovement,
    ProductFilter
} from './types'

export class ApiError extends Error {
    status: number
    body: ApiErrorBody

    constructor(status: number, body:ApiErrorBody) {
        super(body.message ?? body.code)
        this.name = 'ApiError'
        this.status = status
        this.body = body
    }
}

function parseErrorBody(body: unknown, status: number): ApiErrorBody {
    if (body && typeof body == 'object' && 'error' in body) {
        return (body as {error: ApiErrorBody}).error
    }
    return {code: `HTTP_${status}`, message: `Request failed (${status})`}
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
    let res: Response
    try {
        res = await fetch(`/api${path}`, {
            ...init,
            headers: {'Content-Type': 'application/json', ...init?.headers},
        })
    } catch {
        throw new ApiError(0, {code: 'NETWORK_ERROR', message: 'cannot reach the server'})
    }

    if (res.status === 204) {
        return undefined as T
    }

    const body: unknown = await res.json().catch(() => null)
    if (!res.ok) {
        throw new ApiError(res.status, parseErrorBody(body, res.status))
    }

    return body as T
}

export interface ProductQuery {
    search ?: string
    status?: ProductFilter
    page ?: number
    limit ?: number
}

export const api = {
    listProducts(q: ProductQuery = {}) {
        const params = new URLSearchParams()
        if (q.search) params.set('search', q.search)
        if (q.status && q.status !== 'all') params.set('status', q.status)
        if (q.page) params.set('page', String(q.page))
        if (q.limit) params.set('limit', String(q.limit))
        return request<ProductList>(`/products?${params}`)
    },

    adjustStock(id: number, input: AdjustStockInput) {
        return request<DataResponse<AdjustStockResult>>(`/products/${id}/stock`, {
            method: 'POST',
            body: JSON.stringify(input)
        }).then((r) => r.data)
    },

    listMovements(id: number) {
        return request<DataResponse<StockMovement[]>>(`/products/${id}/movements`).then((r) => r.data)
    },

    createOrder(input: CreateOrderInput) {
        return request<DataResponse<Order>>(`/orders`, {
            method: 'POST',
            body: JSON.stringify(input)
        }).then((r) => r.data)
    },

    getOrder(id: number) {
        return request<DataResponse<Order>>(`/orders/${id}`).then((r) => r.data)
    },

    cancelOrder(id: number) {
        return request<DataResponse<Order>>(`/orders/${id}/cancel`, { method: 'POST'}).then((r) => r.data)
    },
}