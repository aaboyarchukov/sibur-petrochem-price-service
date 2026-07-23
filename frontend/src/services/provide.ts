// DI-точка: единственное место, где выбирается реализация PricingApi.
// Замена mock -> http здесь, без правки экранов и сторов.
import type { PricingApi } from './PricingApi'
import { MockPricingApi } from './mock/MockPricingApi'
// import { HttpPricingApi } from './http/HttpPricingApi'

// Composition root: собрать реализацию. Для реального backend — вернуть new HttpPricingApi().
export function createPricingApi(): PricingApi {
  return new MockPricingApi()
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
