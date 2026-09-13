import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import { createPinia } from "pinia";
import { createI18n } from "vue-i18n";
import type { SubscriptionPlan } from "@/types/payment";
import SubscriptionPlanCard from "../SubscriptionPlanCard.vue";

const i18n = createI18n({
  legacy: false,
  locale: "en",
  fallbackWarn: false,
  missingWarn: false,
  messages: {
    en: {
      payment: {
        days: "days",
        weeks: "weeks",
        months: "months",
        perMonth: "month",
        models: "Models",
        planCard: {
          quota: "Quota",
          rate: "Rate",
          unlimited: "Unlimited",
        },
        subscribeNow: "Subscribe now",
      },
    },
  },
});

const mountPlanCard = (groupPlatform: string, overrides: Partial<SubscriptionPlan> = {}) =>
  mount(SubscriptionPlanCard, {
    props: {
      plan: {
        id: 1,
        group_id: 10,
        group_platform: groupPlatform,
        name: "Pro",
        price: 10,
        amount: 1000,
        features: [],
        rate_multiplier: 1,
        validity_days: 30,
        validity_unit: "day",
        supported_model_scopes: ["claude", "gemini_text", "gemini_image"],
        is_active: true,
        ...overrides,
      },
    },
    global: { plugins: [i18n, createPinia()] },
  });

describe("SubscriptionPlanCard", () => {
  it("does not show Antigravity model scopes for OpenAI plans", () => {
    const text = mountPlanCard("openai").text();

    expect(text).not.toContain("Claude");
    expect(text).not.toContain("Gemini");
    expect(text).not.toContain("Imagen");
  });

  it("shows model scopes for Antigravity plans", () => {
    const text = mountPlanCard("antigravity").text();

    expect(text).toContain("Claude");
    expect(text).toContain("Gemini");
    expect(text).toContain("Imagen");
  });

  // #4607：管理端保存的单位是复数（months/weeks），此前用户侧只匹配单数
  // 'month'，「1 个月」的套餐卡片被显示成「1天」。测试环境的 vue-i18n 为
  // runtime-only 构建，t() 原样返回 key，故按 key 断言单位分支。
  it("renders plural admin-form validity units instead of mislabeled days (#4607)", () => {
    expect(mountPlanCard("openai", { validity_days: 1, validity_unit: "months" }).text()).toContain("/ payment.perMonth");
    expect(mountPlanCard("openai", { validity_days: 3, validity_unit: "months" }).text()).toContain("/ 3payment.months");
    expect(mountPlanCard("openai", { validity_days: 2, validity_unit: "weeks" }).text()).toContain("/ 2payment.weeks");
    expect(mountPlanCard("openai", { validity_days: 30, validity_unit: "day" }).text()).toContain("/ 30payment.days");
  });

  it("uses the configured currency symbol and defaults legacy sale prices to CNY", () => {
    const cnyPlan = mountPlanCard("openai", { currency: "CNY", original_price: 20 }).text();

    expect(cnyPlan).toContain("¥10.00");
    expect(cnyPlan).toContain("原价 ¥20.00");
    expect(mountPlanCard("openai", { currency: "USD" }).text()).toContain("$10.00");
    expect(mountPlanCard("openai", { currency: "" }).text()).toContain("¥10.00");
  });

  it("keeps legacy checkout plans purchasable when for_sale is omitted", () => {
    const wrapper = mountPlanCard("openai", { for_sale: undefined });

    expect(wrapper.find(".subscription-plan-card__availability").text()).toContain("可购买");
    expect(wrapper.find(".subscription-plan-card__footer button").attributes("disabled")).toBeUndefined();
  });

  it("keeps explicitly hidden plans unavailable", () => {
    const wrapper = mountPlanCard("openai", { for_sale: false });

    expect(wrapper.find(".subscription-plan-card__availability").text()).toContain("暂不可用");
    expect(wrapper.find(".subscription-plan-card__footer button").attributes("disabled")).toBeDefined();
  });

  it.each([
    ["long Chinese", "企业全球加速专业订阅套餐（含高级模型与优先支持）"],
    ["long English", "Enterprise Global Acceleration Subscription with Priority Support"],
    ["unbroken token", "EnterpriseGlobalAccelerationSubscriptionWithPrioritySupport1234567890"],
  ])("keeps the full %s plan title accessible in a bounded two-line area", (_label, name) => {
    const wrapper = mountPlanCard("openai", { name });
    const title = wrapper.get("h3");

    expect(title.text()).toBe(name);
    expect(title.attributes("title")).toBe(name);
    expect(title.element.parentElement?.classList).toContain("subscription-plan-card__head");
  });

  it("keeps title, badge, price, description, and purchase action in separate bounded regions", () => {
    const wrapper = mountPlanCard("openai", {
      name: "Enterprise Global Acceleration Subscription with Priority Support",
      price: 123.45,
      currency: "USD",
      description: "Includes advanced models and priority support.",
    });
    const title = wrapper.get("h3");
    const price = wrapper.find(".subscription-plan-card__price");

    expect(wrapper.find(".subscription-plan-card__identity").text()).toContain("OpenAI");
    expect(price.text()).toContain("$123.45");
    expect(price.text()).toContain("/ 30payment.days");
    expect(wrapper.get("p").text()).toBe("Includes advanced models and priority support.");
    expect(wrapper.find(".subscription-plan-card__footer button").text()).toContain("立即订阅");
  });

  it("keeps short plan titles compact and aligned", () => {
    const wrapper = mountPlanCard("openai", { name: "Pro", description: "" });
    const title = wrapper.get("h3");

    expect(title.text()).toBe("Pro");
    expect(title.attributes("title")).toBe("Pro");
    expect(wrapper.find(".subscription-plan-card__identity").text()).toContain("OpenAI");
    expect(wrapper.find(".subscription-plan-card__price").text()).toContain("¥10.00");
  });

  it("shows plan-level monthly quota when entitlements are empty", () => {
    const wrapper = mountPlanCard("openai", {
      entitlements: {},
      daily_limit_usd: 54,
      weekly_limit_usd: 115.2,
      monthly_limit_usd: 270,
    });

    expect(wrapper.find(".subscription-plan-card__metrics").text()).toContain("$270.00");
    expect(wrapper.find(".subscription-plan-card__entitlement-value").text()).toBe("深度推理协作，适合复杂工程任务");
    expect(wrapper.find(".subscription-plan-card__entitlement-value").text()).not.toContain("计费");
  });

  it("keeps the standalone entitlement summary distinct by tier", () => {
    const basic = mountPlanCard("openai", { name: "Basic" });
    const ultra = mountPlanCard("openai", { name: "Ultra" });

    expect(basic.find(".subscription-plan-card__entitlement-value").text()).toBe("GPT / Codex 入门，覆盖常见工作任务");
    expect(ultra.find(".subscription-plan-card__entitlement-value").text()).toBe("团队自动化工作流，支撑高峰期调用");
    expect(basic.findAll(".subscription-plan-card__features li").map((item) => item.text()).join(" ")).not.toContain("套餐权益");
  });

  it.each([
    ["GPT Image 2.5", "Basic", "$0.08 额度"],
    ["GPT Image 2.5", "Plus", "$0.08 额度"],
    ["GPT Image 2.5", "Standard", "$0.08 额度"],
    ["GPT Image 2.5", "Pro", "$0.06 额度"],
    ["GPT Image 2.5", "Ultra", "$0.05 额度"],
    ["香蕉生图", "Basic", "$0.15 额度"],
    ["香蕉生图", "Plus", "$0.15 额度"],
    ["香蕉生图", "Standard", "$0.13 额度"],
    ["香蕉生图", "Pro", "$0.12 额度"],
    ["香蕉生图", "Ultra", "$0.10 额度"],
  ])("shows the image quota consumed per generated image for %s %s", (groupName, name, unitPrice) => {
    const wrapper = mountPlanCard("openai", { group_name: groupName, name });

    expect(wrapper.find(".subscription-plan-card__entitlement-value").text()).toContain(unitPrice);
  });

  it("keeps image unit pricing after the product series is renamed", () => {
    expect(mountPlanCard("openai", { group_name: "GPT Image订阅", name: "Pro" })
      .find(".subscription-plan-card__entitlement-value").text()).toContain("$0.06 额度");
    expect(mountPlanCard("openai", { group_name: "nano banana订阅", name: "Ultra" })
      .find(".subscription-plan-card__entitlement-value").text()).toContain("$0.10 额度");
  });
});
