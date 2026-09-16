import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const source = readFileSync(resolve('src/views/admin/SubscriptionsView.vue'), 'utf8')

describe('admin subscription plan details', () => {
  it('keeps plan information in the list and assignment form', () => {
    expect(source).toContain('getPlanName(row.plan_id)')
    expect(source).toContain('v-model="assignForm.plan_id"')
    expect(source).toContain('subscriptionPlanOptions')
    expect(source).toContain('adminAPI.payment.getPlans()')
    expect(source).toContain('plan_id: assignForm.plan_id')
  })

  it('keeps plan editing in the adjustment dialog', () => {
    expect(source).toContain('v-model="extendPlanId"')
    expect(source).toContain('extendPlanOptions')
    expect(source).toContain('plan_id: extendPlanId.value')
    expect(source).toContain('formatPlanQuota(getPlanById(extendPlanId)!')
  })
})
