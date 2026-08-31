import type { ReactNode } from "react";
import { useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";

import { createModelChannel, encodeChannelModel, modelOptionsFromChannels, useConfigStore, type VideoModelCapability } from "@/stores/use-config-store";
import { usePromptSourceScheduler } from "@/hooks/use-prompt-source-scheduler";

type LingquBridge = {
    apiUrl?: string;
    apiKey?: string;
    videoApiUrl?: string;
    keyId?: number;
    keyName?: string;
    groupName?: string;
    model?: string;
    imageKeyId?: number;
    imageApiKey?: string;
    imageKeyName?: string;
    imageGroupName?: string;
    imageKeys?: Array<{
        id?: number;
        apiKey?: string;
        keyName?: string;
        groupName?: string;
        model?: string;
    }>;
    textKeyId?: number;
    textApiKey?: string;
    textKeyName?: string;
    textGroupName?: string;
    textKeys?: Array<{
        id?: number;
        apiKey?: string;
        keyName?: string;
        groupName?: string;
        model?: string;
    }>;
    videoKeyId?: number;
    videoApiKey?: string;
    videoKeyName?: string;
    videoGroupName?: string;
};

type VideoModelPayload = VideoModelCapability & {
    id?: string;
    object?: string;
};

type VideoCapabilityPayload = VideoModelCapability & {
    id?: string;
};

export function ClientRootInit({ children }: { children: ReactNode }) {
    const { t } = useTranslation();
    const handledLingquBridge = useRef(false);
    const updateConfig = useConfigStore((state) => state.updateConfig);
    const config = useConfigStore((state) => state.config);

    usePromptSourceScheduler();

    useEffect(() => {
        if (handledLingquBridge.current) return;
        const bridgeKey = "lingqu:ai-creation:bridge";
        let bridge: LingquBridge | null = null;
        try {
            const raw = window.sessionStorage.getItem(bridgeKey) || window.localStorage.getItem(bridgeKey);
            bridge = raw ? (JSON.parse(raw) as LingquBridge) : null;
        } catch {
            bridge = null;
        }
        if (!bridge?.apiUrl && !bridge?.videoApiUrl) return;
        handledLingquBridge.current = true;
        updateConfig("credentialsManagedByHost", true);

        const imageApiKey = bridge.imageApiKey || bridge.apiKey || "";
        const textApiKey = bridge.textApiKey || "";
        const videoApiKey = bridge.videoApiKey || bridge.apiKey || "";
        const imageChannels = bridge.apiUrl
            ? (bridge.imageKeys || [])
                  .map((entry, index) => {
                      const apiKey = entry.apiKey?.trim() || "";
                      if (!apiKey) return null;
                      return createModelChannel({
                          id: entry.id ? `lingqu-image-${entry.id}` : `lingqu-image-${index + 1}`,
                          name: entry.groupName || entry.keyName || t("config.channels.defaultName"),
                          baseUrl: bridge.apiUrl,
                          apiKey,
                          apiFormat: "openai",
                          asyncImageTasks: true,
                          models: [{ name: entry.model || bridge.model || "gpt-image-2", capability: "image" }],
                      });
                  })
                  .filter((channel): channel is NonNullable<typeof channel> => Boolean(channel))
            : [];
        const imageChannel =
            imageChannels[0] ||
            (bridge.apiUrl && imageApiKey
                ? createModelChannel({
                      id: "lingqu-image",
                      name: bridge.imageGroupName || bridge.imageKeyName || bridge.groupName || bridge.keyName || t("config.channels.defaultName"),
                      baseUrl: bridge.apiUrl,
                      apiKey: imageApiKey,
                      apiFormat: "openai",
                      asyncImageTasks: true,
                      models: [{ name: bridge.model || "gpt-image-2", capability: "image" }],
                  })
                : null);
        const textChannels = bridge.apiUrl
            ? (bridge.textKeys || [])
                  .map((entry, index) => {
                      const apiKey = entry.apiKey?.trim() || "";
                      if (!apiKey) return null;
                      return createModelChannel({
                          id: entry.id ? `lingqu-text-${entry.id}` : `lingqu-text-${index + 1}`,
                          name: entry.groupName || entry.keyName || t("config.channels.defaultName"),
                          baseUrl: bridge.apiUrl!,
                          apiKey,
                          apiFormat: "openai",
                          models: [{ name: entry.model || "gpt-5.5", capability: "text" }],
                      });
                  })
                  .filter((channel): channel is NonNullable<typeof channel> => Boolean(channel))
            : [];
        const textChannel =
            textChannels[0] ||
            (bridge.apiUrl && textApiKey
                ? createModelChannel({
                      id: "lingqu-text",
                      name: bridge.textGroupName || bridge.textKeyName || t("config.channels.defaultName"),
                      baseUrl: bridge.apiUrl,
                      apiKey: textApiKey,
                      apiFormat: "openai",
                      models: [{ name: "gpt-5.5", capability: "text" }],
                  })
                : null);
        const videoChannel = bridge.videoApiUrl && videoApiKey
            ? createModelChannel({
                  id: "lingqu-video",
                  name: bridge.videoGroupName || bridge.videoKeyName || t("config.channels.defaultName"),
                  baseUrl: bridge.videoApiUrl,
                  apiKey: videoApiKey,
                  apiFormat: "openai",
                  models: [{ name: "grok-imagine-video", capability: "video" }],
              })
            : null;
        // In host-managed mode, only channels injected by the host are valid.
        // Dropping locally persisted channels prevents stale endpoints or keys
        // from bypassing the system-owned credential boundary.
        const channels = [
            ...(imageChannels.length > 0 ? imageChannels : imageChannel ? [imageChannel] : []),
            ...(textChannels.length > 0 ? textChannels : textChannel ? [textChannel] : []),
            ...(videoChannel ? [videoChannel] : []),
        ];
        if (!channels.length) channels.push(createModelChannel({ id: "default", name: t("config.channels.defaultName") }));
        const imageModel = imageChannel ? encodeChannelModel(imageChannel.id, imageChannel.models[0].name) : config.imageModel;
        updateConfig("channels", channels);
        updateConfig("models", modelOptionsFromChannels(channels));
        if (imageChannel) {
            updateConfig("baseUrl", imageChannel.baseUrl);
            updateConfig("apiKey", imageChannel.apiKey);
            updateConfig("model", imageModel);
            updateConfig("imageModel", imageModel);
        }
        updateConfig("textModel", textChannel ? encodeChannelModel(textChannel.id, textChannel.models[0].name) : "");

        if (!bridge.videoApiUrl || !videoChannel) return;
        const token = window.localStorage.getItem("auth_token");
        void fetch(`${bridge.videoApiUrl.replace(/\/$/, "")}/bootstrap`, {
            credentials: "same-origin",
            headers: {
                ...(token ? { Authorization: `Bearer ${token}` } : {}),
                ...(bridge.videoKeyId ? { "X-Video-Key-Id": String(bridge.videoKeyId) } : {}),
            },
        })
            .then(async (response) => {
                if (!response.ok) return null;
                return (await response.json()) as {
                    models?: { data?: VideoModelPayload[] };
                    capabilities?: { data?: VideoCapabilityPayload[] };
                };
            })
            .then((payload) => {
                const capabilityByModel = new Map((payload?.capabilities?.data || []).filter((capability): capability is VideoCapabilityPayload & { id: string } => Boolean(capability.id)).map((capability) => [capability.id, capability]));
                const models = (payload?.models?.data || []).filter((model): model is VideoModelPayload & { id: string } => Boolean(model.id));
                if (!models.length) return;
                const nextChannel = {
                    ...videoChannel,
                    models: models.map((model) => ({
                        name: model.id,
                        capability: "video" as const,
                        videoCapability: {
                            ...model,
                            ...(capabilityByModel.get(model.id) || {}),
                        } as VideoModelCapability,
                    })),
                };
                const nextChannels = channels.map((channel) => (channel.id === nextChannel.id ? nextChannel : channel));
                const nextVideoModel = encodeChannelModel(nextChannel.id, models[0].id);
                updateConfig("channels", nextChannels);
                updateConfig("models", modelOptionsFromChannels(nextChannels));
                updateConfig("videoModel", nextVideoModel);
            })
            .catch(() => undefined);
    }, [config.apiKey, config.channels, config.imageModel, config.videoModel, config.baseUrl, t, updateConfig]);

    return <>{children}</>;
}
