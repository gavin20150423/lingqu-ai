import { useEffect, useId, useMemo, useState } from "react";
import { Cpu, Layers3 } from "lucide-react";
import { useTranslation } from "react-i18next";

import i18n from "@/i18n";
import { Select, SelectContent, SelectItem, SelectTrigger } from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { decodeChannelModel, encodeChannelModel, modelOptionLabel, modelOptionName, selectableModelsByCapability, type AiConfig, type ModelCapability, type ModelChannel } from "@/stores/use-config-store";

type ModelPickerProps = {
    config: AiConfig;
    value?: string;
    onChange: (model: string) => void;
    capability?: ModelCapability;
    className?: string;
    fullWidth?: boolean;
    grouped?: boolean;
    stacked?: boolean;
    placeholder?: string;
    onMissingConfig?: () => void;
};

export function ModelPicker({ config, value, onChange, capability, className, fullWidth = false, grouped = false, stacked = false, placeholder, onMissingConfig }: ModelPickerProps) {
    const { t } = useTranslation();
    const pickerId = useId();
    const [open, setOpen] = useState(false);
    const options = useMemo(() => Array.from(new Set([...(config.channelMode === "local" && !capability ? [value] : []), ...selectableModelsByCapability(config, capability)].filter((model): model is string => Boolean(model)))), [capability, config, value]);
    const current = value || "";
    const pickerPlaceholder = placeholder || t("settingsPanels.model.select");

    useEffect(() => {
        const closeOtherPicker = (event: Event) => {
            if ((event as CustomEvent<string>).detail !== pickerId) setOpen(false);
        };
        window.addEventListener("model-picker-open", closeOtherPicker);
        return () => window.removeEventListener("model-picker-open", closeOtherPicker);
    }, [pickerId]);

    if (grouped) {
        return <GroupedModelPicker config={config} value={value} onChange={onChange} capability={capability} className={className} fullWidth={fullWidth} stacked={stacked} placeholder={placeholder} onMissingConfig={onMissingConfig} />;
    }

    return (
        <Select
            open={open}
            value={current}
            onOpenChange={(nextOpen) => {
                if (nextOpen && !options.length && config.channelMode === "local") onMissingConfig?.();
                if (nextOpen) window.dispatchEvent(new CustomEvent("model-picker-open", { detail: pickerId }));
                setOpen(nextOpen);
            }}
            onValueChange={onChange}
        >
            <SelectTrigger
                className={cn(
                    "canvas-composer-model-picker h-8 w-fit max-w-full gap-2 rounded-full border border-input bg-transparent px-3 text-sm font-normal shadow-sm transition-colors",
                    fullWidth ? "w-full min-w-0 justify-start" : "min-w-[9rem] justify-start",
                    "data-[state=open]:border-ring data-[state=open]:ring-2 data-[state=open]:ring-ring/20",
                    className,
                )}
                onMouseDown={(event) => event.stopPropagation()}
                onPointerDown={(event) => event.stopPropagation()}
                title={current ? modelOptionLabel(config, current) : pickerPlaceholder}
            >
                <ModelIcon model={current} />
                <span className="canvas-model-picker-text min-w-0 flex-1 truncate text-left">{current ? modelOptionLabel(config, current) : pickerPlaceholder}</span>
            </SelectTrigger>
            <SelectContent
                data-canvas-no-zoom
                className="z-[1200] w-80 max-w-[calc(100vw-24px)] rounded-xl border border-border/70 bg-popover p-1 shadow-xl"
                position="popper"
                align="start"
                side="bottom"
                sideOffset={6}
                onPointerDown={(event) => event.stopPropagation()}
                onMouseDown={(event) => event.stopPropagation()}
            >
                {options.length ? (
                    options.map((model) => (
                        <SelectItem key={model} value={model} textValue={modelOptionLabel(config, model)}>
                            <ModelLabel config={config} model={model} />
                        </SelectItem>
                    ))
                ) : (
                    <SelectItem value="__empty__" disabled>
                        {emptyModelLabel(config, capability)}
                    </SelectItem>
                )}
            </SelectContent>
        </Select>
    );
}

function GroupedModelPicker({ config, value, onChange, capability, className, fullWidth, stacked = false, placeholder, onMissingConfig }: ModelPickerProps) {
    const { t } = useTranslation();
    const channels = useMemo(
        () => config.channels.filter((channel) => channel.models.some((model) => !capability || model.capability === capability)),
        [capability, config.channels],
    );
    const decoded = decodeChannelModel(value || "");
    const selectedChannel = useMemo<ModelChannel | undefined>(() => {
        if (decoded?.channelId) {
            const exact = channels.find((channel) => channel.id === decoded.channelId);
            if (exact) return exact;
        }
        if (decoded?.model || value) {
            const modelName = decoded?.model || value || "";
            const matching = channels.find((channel) => channel.models.some((model) => model.name === modelName && (!capability || model.capability === capability)));
            if (matching) return matching;
        }
        return channels[0];
    }, [capability, channels, decoded?.channelId, decoded?.model, value]);
    const modelOptions = selectedChannel?.models.filter((model) => !capability || model.capability === capability) || [];
    const selectedModel = decoded?.model && modelOptions.some((model) => model.name === decoded.model) ? decoded.model : "";
    const groupPlaceholder = t("settingsPanels.model.selectGroup");
    const modelPlaceholder = placeholder || t("settingsPanels.model.select");
    const selectClass = cn(
        "h-11 min-w-0 rounded-xl border border-input bg-transparent px-3 text-sm font-normal shadow-sm transition-colors",
        "data-[state=open]:border-ring data-[state=open]:ring-2 data-[state=open]:ring-ring/20",
    );

    return (
        <div className={cn("grid min-w-0 gap-2", stacked ? "grid-cols-1" : "grid-cols-2", fullWidth ? "w-full" : "w-fit", className)}>
            <Select
                value={selectedChannel?.id || ""}
                onOpenChange={(open) => {
                    if (open && !channels.length) onMissingConfig?.();
                }}
                onValueChange={(channelId) => {
                    const channel = channels.find((item) => item.id === channelId);
                    const firstModel = channel?.models.find((model) => !capability || model.capability === capability);
                    if (channel && firstModel) onChange(encodeChannelModel(channel.id, firstModel.name));
                }}
            >
                <SelectTrigger
                    className={cn(selectClass, "w-full justify-start gap-2")}
                    onMouseDown={(event) => event.stopPropagation()}
                    onPointerDown={(event) => event.stopPropagation()}
                    title={selectedChannel?.name || groupPlaceholder}
                >
                    <Layers3 className="size-4 shrink-0 opacity-65" />
                    <span className="min-w-0 flex-1 truncate text-left">{selectedChannel?.name || groupPlaceholder}</span>
                </SelectTrigger>
                <SelectContent data-canvas-no-zoom className="z-[1200] max-h-[min(var(--radix-select-content-available-height),26rem)] w-[var(--radix-select-trigger-width)] max-w-[calc(100vw-24px)] rounded-xl" position="popper" align="start" side="bottom" sideOffset={6}>
                    {channels.length ? channels.map((channel) => <SelectItem key={channel.id} value={channel.id} className="min-w-0 overflow-hidden [&>span:last-child]:min-w-0 [&>span:last-child]:flex-1 [&>span:last-child]:overflow-hidden"><span className="min-w-0 truncate" title={channel.name}>{channel.name}</span></SelectItem>) : <SelectItem value="__empty__" disabled>{t("settingsPanels.model.addFirst")}</SelectItem>}
                </SelectContent>
            </Select>

            <Select
                value={selectedModel}
                onOpenChange={(open) => {
                    if (open && !modelOptions.length) onMissingConfig?.();
                }}
                onValueChange={(modelName) => {
                    if (selectedChannel) onChange(encodeChannelModel(selectedChannel.id, modelName));
                }}
            >
                <SelectTrigger
                    className={cn(selectClass, "w-full justify-start gap-2")}
                    onMouseDown={(event) => event.stopPropagation()}
                    onPointerDown={(event) => event.stopPropagation()}
                    title={selectedModel ? modelOptionLabel(config, encodeChannelModel(selectedChannel?.id || "", selectedModel)) : modelPlaceholder}
                >
                    <Cpu className="size-4 shrink-0 opacity-65" />
                    <span className="min-w-0 flex-1 truncate text-left">{selectedModel || modelPlaceholder}</span>
                </SelectTrigger>
                <SelectContent data-canvas-no-zoom className="z-[1200] max-h-[min(var(--radix-select-content-available-height),26rem)] w-[var(--radix-select-trigger-width)] max-w-[calc(100vw-24px)] rounded-xl" position="popper" align="start" side="bottom" sideOffset={6}>
                    {modelOptions.length ? modelOptions.map((modelOption) => <SelectItem key={modelOption.name} value={modelOption.name} textValue={modelOption.name} className="min-w-0 overflow-hidden [&>span:last-child]:min-w-0 [&>span:last-child]:flex-1 [&>span:last-child]:overflow-hidden" title={modelOption.name}><ModelLabel config={config} model={encodeChannelModel(selectedChannel?.id || "", modelOption.name)} /></SelectItem>) : <SelectItem value="__empty__" disabled>{emptyModelLabel(config, capability)}</SelectItem>}
                </SelectContent>
            </Select>
        </div>
    );
}

function emptyModelLabel(config: AiConfig, capability?: ModelCapability) {
    const label = capability ? i18n.t(`settingsPanels.model.capabilities.${capability}`) : "";
    if (capability && config.models.length) return i18n.t("settingsPanels.model.assign", { capability: label });
    return config.models.length ? i18n.t("settingsPanels.model.noMatch", { capability: label }) : i18n.t("settingsPanels.model.addFirst");
}

function ModelLabel({ config, model }: { config: AiConfig; model: string }) {
    return (
        <span className="flex w-full min-w-0 items-center gap-2 overflow-hidden">
            <ModelIcon model={model} />
            <span className="truncate">{modelOptionLabel(config, model)}</span>
        </span>
    );
}

function ModelIcon({ model }: { model: string }) {
    const icon = resolveModelIcon(modelOptionName(model));
    return icon ? <img src={icon} alt="" className="size-4 shrink-0 dark:invert" /> : <Cpu className="size-4 shrink-0 opacity-70" />;
}

function resolveModelIcon(model: string) {
    const name = model.toLowerCase();
    if (name.includes("claude") || name.includes("anthropic")) return "/icons/claude.svg";
    if (name.includes("gemini") || name.includes("google")) return "/icons/gemini.svg";
    if (name.includes("gpt") || name.includes("openai")) return "/icons/openai.svg";
    if (name.includes("grok") || name.includes("grok")) return "/icons/grok.svg";
    if (name.includes("deepseek") || name.includes("deepseek")) return "/icons/deepseek.svg";
    if (name.includes("glm") || name.includes("glm")) return "/icons/glm.svg";
    return "";
}
