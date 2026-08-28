import { modelOptionName, videoModelCapabilityOf, type AiConfig, type VideoDurationRule, type VideoModelCapability, type VideoPricingVariant } from "@/stores/use-config-store";

export type VideoPriceEstimate = {
    amount: string;
    currency: string;
    unitPrice: string;
};

export function getVideoCapability(config: AiConfig, model = config.videoModel || config.model) {
    return videoModelCapabilityOf(config, model) || knownVideoCapability(model);
}

// CTMOAI publishes this model's limits in /v1/models. Keep a small local
// fallback so a cold bootstrap or a temporarily unavailable catalogue cannot
// expose unsupported reference types or durations in the workbench.
function knownVideoCapability(model: string): VideoModelCapability | undefined {
    const normalized = modelOptionName(model).trim().toLowerCase();
    if (!normalized.includes("minimax-h3-quantized-768p")) return undefined;
    return {
        resolutions: ["768p"],
        default_resolution: "768p",
        durations: [4, 5, 6, 7, 8, 9, 10],
        default_duration: 6,
        aspect_ratios: ["16:9", "9:16", "1:1", "2:3", "3:2", "3:4", "4:3", "21:9"],
        default_aspect_ratio: "16:9",
        supports_guidances: true,
        max_references: { image: 4, video: 0, audio: 0 },
    };
}

export function videoResolutionOptions(capability?: VideoModelCapability) {
    const values = capability?.output?.resolutions?.length ? capability.output.resolutions : capability?.resolutions;
    return unique(values || []).map(normalizeResolutionToken);
}

export function videoDefaultResolution(capability?: VideoModelCapability) {
    const options = videoResolutionOptions(capability);
    const preferred = normalizeResolutionToken(capability?.output?.default_resolution || capability?.default_resolution || "");
    return options.includes(preferred) ? preferred : options[0] || "720p";
}

export function videoDurationOptions(capability: VideoModelCapability | undefined, resolution: string) {
    const normalizedResolution = normalizeResolutionToken(resolution);
    const rule = findResolutionRule(capability?.output?.durations_by_resolution, normalizedResolution);
    if (rule) {
        if (rule.type === "range" || (rule.min !== undefined && rule.max !== undefined)) {
            const min = Math.max(1, Math.ceil(Number(rule.min)));
            const max = Math.max(min, Math.floor(Number(rule.max)));
            const step = Math.max(1, Math.floor(Number(rule.step) || 1));
            const values: number[] = [];
            for (let value = min; value <= max && values.length < 120; value += step) values.push(value);
            return values;
        }
        if (Array.isArray(rule.values)) return normalizeNumberOptions(rule.values);
    }
    const fallback = capability?.durations || capability?.output?.durations;
    const options = normalizeNumberOptions(fallback || []);
    if (options.length) return options;
    const defaultDuration = Number(capability?.output?.default_duration || capability?.default_duration || 0);
    return defaultDuration > 0 ? [Math.floor(defaultDuration)] : [6];
}

export function videoDefaultDuration(capability: VideoModelCapability | undefined, resolution: string) {
    const options = videoDurationOptions(capability, resolution);
    const preferred = Math.floor(Number(capability?.output?.default_duration || capability?.default_duration || 0));
    return options.includes(preferred) ? preferred : options[0] || 6;
}

export function videoAspectRatioOptions(capability: VideoModelCapability | undefined, resolution: string) {
    const normalizedResolution = normalizeResolutionToken(resolution);
    const byResolution = findResolutionValues(capability?.output?.aspect_ratios_by_resolution, normalizedResolution);
    const values = byResolution?.length ? byResolution : capability?.aspect_ratios;
    const fallback = values?.length ? values : [capability?.output?.default_aspect_ratio || capability?.default_aspect_ratio || "16:9"];
    return unique(fallback.filter((value): value is string => typeof value === "string" && /^\d+:\d+$/.test(value)));
}

export function videoDefaultAspectRatio(capability: VideoModelCapability | undefined, resolution: string) {
    const options = videoAspectRatioOptions(capability, resolution);
    const preferred = capability?.output?.default_aspect_ratio || capability?.default_aspect_ratio || "";
    return options.includes(preferred) ? preferred : options[0] || "16:9";
}

export function videoSupportsAudio(capability?: VideoModelCapability) {
    return capability?.output?.supports_generated_audio ?? false;
}

export function videoSupportsWatermark(capability?: VideoModelCapability) {
    return capability?.output?.supports_watermark ?? false;
}

export function videoSupportsGuidance(capability?: VideoModelCapability) {
    if (!capability) return false;
    return videoSupportsReferenceKind(capability, "image") || videoSupportsReferenceKind(capability, "video") || videoSupportsReferenceKind(capability, "audio");
}

export type VideoReferenceKind = "image" | "video" | "audio";

/** Returns the provider-declared maximum, or undefined when the provider did not declare one. */
export function videoReferenceLimit(capability: VideoModelCapability | undefined, kind: VideoReferenceKind) {
    if (!capability) return undefined;
    const references = capability.max_references || capability.references;
    if (!references || typeof references !== "object") return undefined;
    const value = references[kind];
    const parsed = Number(value);
    return Number.isFinite(parsed) ? Math.max(0, Math.floor(parsed)) : undefined;
}

export function videoSupportsReferenceKind(capability: VideoModelCapability | undefined, kind: VideoReferenceKind) {
    if (!capability) return true;
    const limit = videoReferenceLimit(capability, kind);
    if (limit !== undefined) return limit > 0;
    if (typeof capability.supports_guidances === "boolean") return capability.supports_guidances && kind === "image";
    const modes = capability.generation_modes || capability.output?.generation_modes;
    if (Array.isArray(modes)) return modes.some((mode) => /reference|omni|first.?frame/i.test(mode)) && kind === "image";
    return false;
}

export function videoReferenceModeLabel(capability?: VideoModelCapability) {
    return videoSupportsReferenceKind(capability, "video") || videoSupportsReferenceKind(capability, "audio") ? "reference" : "imageReference";
}

export function estimateVideoPrice(config: AiConfig, options: { hasVideoReference?: boolean } = {}): VideoPriceEstimate | null {
    const capability = getVideoCapability(config);
    if (!capability?.pricing_variants?.length) return null;
    const resolution = normalizeResolutionToken(config.vquality || videoDefaultResolution(capability));
    const ratio = normalizeRatio(config.size);
    const duration = Math.floor(Number(config.videoSeconds));
    if (!Number.isFinite(duration) || duration <= 0) return null;
    const candidates = capability.pricing_variants.filter((variant) => matchesVariant(variant, resolution, ratio));
    if (candidates.length !== 1 || candidates[0].unit_price === undefined) return null;
    const rawPrice = String(candidates[0].unit_price).trim();
    const audioPrice = videoSupportsAudio(capability) && config.videoGenerateAudio === "true" ? String(candidates[0].audio_unit_price ?? 0).trim() : "0";
    const unitPrice = parseDecimal(rawPrice, 6);
    const audioUnitPrice = parseDecimal(audioPrice, 6);
    if (unitPrice === null || audioUnitPrice === null) return null;
    let scaledUnits = unitPrice + audioUnitPrice;
    if (normalizeBillingUnit(candidates[0].billing_unit) === "per_second") {
        scaledUnits *= BigInt(duration);
    }
    if (options.hasVideoReference && Number(capability.reference_video_multiplier) > 1) {
        const multiplier = parseDecimal(String(capability.reference_video_multiplier), 6);
        if (multiplier !== null) scaledUnits = (scaledUnits * multiplier + BigInt(500000)) / BigInt(1000000);
    }
    const cents = (scaledUnits + BigInt(5000)) / BigInt(10000);
    return {
        amount: formatCents(cents),
        currency: candidates[0].currency || "USD",
        unitPrice: rawPrice,
    };
}

export function normalizeResolutionToken(value: string) {
    const normalized = String(value || "")
        .trim()
        .replace(/p$/i, "");
    return normalized ? `${normalized}p` : "720p";
}

export function normalizeRatio(value: string) {
    if (/^\d+:\d+$/.test(value || "")) return value;
    const match = String(value || "").match(/^(\d+)x(\d+)$/);
    if (!match) return "16:9";
    const width = Number(match[1]);
    const height = Number(match[2]);
    if (width === height) return "1:1";
    return width > height ? "16:9" : "9:16";
}

export function modelNameForVideoConfig(config: AiConfig) {
    return modelOptionName(config.videoModel || config.model);
}

function matchesVariant(variant: VideoPricingVariant, resolution: string, ratio: string) {
    if (!isSupportedBillingUnit(variant.billing_unit)) return false;
    if (variant.resolution && normalizeResolutionToken(variant.resolution) !== resolution) return false;
    if (variant.aspect_ratio && variant.aspect_ratio !== ratio) return false;
    return true;
}

function isSupportedBillingUnit(value: string | undefined) {
    const unit = String(value || "").trim().toLowerCase();
    return !unit || ["second", "per_second", "per-second", "request", "per_request", "per-request", "task", "per_task", "per-task"].includes(unit);
}

function normalizeBillingUnit(value: string | undefined) {
    const unit = String(value || "").trim().toLowerCase();
    return ["request", "per_request", "per-request", "task", "per_task", "per-task"].includes(unit) ? "per_request" : "per_second";
}

function findResolutionRule(rules: Record<string, VideoDurationRule> | undefined, resolution: string) {
    if (!rules) return undefined;
    return Object.entries(rules).find(([key]) => normalizeResolutionToken(key) === resolution)?.[1];
}

function findResolutionValues(values: Record<string, string[]> | undefined, resolution: string) {
    if (!values) return undefined;
    return Object.entries(values).find(([key]) => normalizeResolutionToken(key) === resolution)?.[1];
}

function normalizeNumberOptions(values: Array<number | string>) {
    return unique(values.map((value) => Math.floor(Number(value))).filter((value) => Number.isFinite(value) && value > 0)).sort((a, b) => a - b);
}

function unique<T>(values: T[]) {
    return Array.from(new Set(values));
}

function parseDecimal(value: string, scale: number) {
    const match = value.match(/^([+-]?)(\d+)(?:\.(\d+))?$/);
    if (!match) return null;
    const fraction = (match[3] || "").padEnd(scale, "0");
    const truncated = fraction.slice(0, scale);
    const units = BigInt(match[2]) * BigInt(10) ** BigInt(scale) + BigInt(truncated || "0");
    return match[1] === "-" ? -units : units;
}

function formatCents(cents: bigint) {
    const sign = cents < BigInt(0) ? "-" : "";
    const absolute = cents < BigInt(0) ? -cents : cents;
    return `${sign}${absolute / BigInt(100)}.${String(absolute % BigInt(100)).padStart(2, "0")}`;
}
