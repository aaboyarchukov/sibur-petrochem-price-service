// DI-точка: единственное место, где выбирается реализация PricingApi.
// Замена mock <-> http здесь, без правки экранов и сторов.
import type { PricingApi } from './PricingApi'
import { MockPricingApi } from './mock/MockPricingApi'
import { HttpPricingApi } from './http/HttpPricingApi'

// Composition root: по умолчанию реальный backend; mock — по env-флагу
// (VITE_API_MODE=mock) для работы без сервера. Vitest подменяет через setPricingApi.
export function createPricingApi(): PricingApi {
  if (import.meta.env.VITE_API_MODE === 'mock') return new MockPricingApi()
  return new HttpPricingApi()
}

// Singleton-доступ для Pinia-сторов (работают вне setup, inject недоступен).
let instance: PricingApi | null = null

export function setPricingApi(api: PricingApi): void {
  instance = api
}

export function usePricingApi(): PricingApi {
  if (!instance) instance = createPricingApi()
  return instance
}
