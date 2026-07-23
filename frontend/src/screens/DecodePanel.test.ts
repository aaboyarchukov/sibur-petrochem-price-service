import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import DecodePanel from './DecodePanel.vue'
import { useCalculationStore } from '@/stores/calculation'
import { setPricingApi } from '@/services/provide'
import { MockPricingApi } from '@/services/mock/MockPricingApi'

async function firstRowWith(status: string): Promise<string> {
  const page = await new MockPricingApi().listRows('calc-mine')
  return page.items.find((r) => r.status === status)!.row_id
}

describe('DecodePanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    setPricingApi(new MockPricingApi())
  })

  it('рендерит формулу и компоненты для посчитанной строки', async () => {
    const store = useCalculationStore()
    await store.start('2026-06')
    const rowId = await firstRowWith('calculated')
    await store.select(rowId)

    const wrapper = mount(DecodePanel)
    expect(wrapper.text()).toContain('Применённая формула')
    // хотя бы один компонент отрисован
    expect(wrapper.findAll('.components-table tbody tr').length).toBeGreaterThan(0)
  })

  it('показывает сообщение об отсутствии формулы для no_formula', async () => {
    const store = useCalculationStore()
    await store.start('2026-06')
    const rowId = await firstRowWith('no_formula')
    await store.select(rowId)

    const wrapper = mount(DecodePanel)
    expect(wrapper.text()).toContain('Формула не подобрана')
  })
})
