import localforage from "localforage";

import { nanoid } from "nanoid";
import i18n from "@/i18n";
import { readImageMeta } from "@/lib/image-utils";

export type UploadedImage = {
    url: string;
    storageKey: string;
    width: number;
    height: number;
    bytes: number;
    mimeType: string;
};

const store = localforage.createInstance({ name: "infinite-canvas", storeName: "image_files" });
const imageLogStore = localforage.createInstance({ name: "infinite-canvas", storeName: "image_generation_logs" });
const videoLogStore = localforage.createInstance({ name: "infinite-canvas", storeName: "video_generation_logs" });
const objectUrls = new Map<string, string>();
const storageKeysByObjectUrl = new Map<string, string>();

type StoredImageInput = string | Blob | { url?: string; dataUrl?: string; storageKey?: string };

export async function uploadImage(input: StoredImageInput): Promise<UploadedImage> {
    const blob = await resolveInputBlob(input);
    const storageKey = `image:${nanoid()}`;
    await store.setItem(storageKey, blob);
    const url = cacheObjectUrl(storageKey, blob);
    const meta = await readImageMeta(url);
    return { url, storageKey, width: meta.width, height: meta.height, bytes: blob.size, mimeType: blob.type || meta.mimeType };
}

export async function resolveImageUrl(storageKey?: string, fallback = "") {
    if (!storageKey) return fallback;
    const cached = objectUrls.get(storageKey);
    if (cached) return cached;
    const blob = await store.getItem<Blob>(storageKey);
    if (!blob) return fallback;
    return cacheObjectUrl(storageKey, blob);
}

export async function getImageBlob(storageKey: string) {
    return store.getItem<Blob>(storageKey);
}

export async function setImageBlob(storageKey: string, blob: Blob) {
    await store.setItem(storageKey, blob);
    return cacheObjectUrl(storageKey, blob);
}

export async function imageToDataUrl(image: { url?: string; dataUrl?: string; storageKey?: string }) {
    // Read durable files directly. Fetching their object URL can be rejected by
    // connect-src even though the same blob: URL is allowed by img-src.
    if (image.storageKey) {
        try {
            const storedBlob = await getImageBlob(image.storageKey);
            if (storedBlob) return blobToDataUrl(storedBlob);
        } catch {
            // Fall through to an embedded data URL or externally readable URL.
        }
    }
    const candidates = Array.from(new Set([image.dataUrl, image.url].filter((value): value is string => Boolean(value))));

    let lastError: unknown;
    for (const url of candidates) {
        if (url.startsWith("data:")) return url;
        try {
            const knownStorageKey = storageKeysByObjectUrl.get(url);
            if (knownStorageKey) {
                const storedBlob = await getImageBlob(knownStorageKey);
                if (storedBlob) return blobToDataUrl(storedBlob);
            }
            return blobToDataUrl(await (await fetch(url)).blob());
        } catch (error) {
            lastError = error;
        }
    }
    if (lastError) throw lastError;
    return "";
}

export async function deleteStoredImages(keys: Iterable<string>) {
    await Promise.all(
        Array.from(new Set(keys)).map(async (key) => {
            const url = objectUrls.get(key);
            if (url) {
                URL.revokeObjectURL(url);
                storageKeysByObjectUrl.delete(url);
            }
            objectUrls.delete(key);
            await store.removeItem(key);
        }),
    );
}

export async function cleanupUnusedImages(usedData: unknown) {
    const usedKeys = collectImageStorageKeys(usedData);
    await Promise.all([
        imageLogStore.iterate((value) => {
            collectImageStorageKeys(value, usedKeys);
        }),
        videoLogStore.iterate((value) => {
            collectImageStorageKeys(value, usedKeys);
        }),
    ]);
    const unused: string[] = [];
    await store.iterate((_value, key) => {
        if (!usedKeys.has(key)) unused.push(key);
    });
    await deleteStoredImages(unused);
}

export function collectImageStorageKeys(value: unknown, keys = new Set<string>()) {
    if (!value || typeof value !== "object") return keys;
    if ("storageKey" in value && typeof value.storageKey === "string" && value.storageKey.startsWith("image:")) keys.add(value.storageKey);
    Object.values(value).forEach((item) => (Array.isArray(item) ? item.forEach((child) => collectImageStorageKeys(child, keys)) : collectImageStorageKeys(item, keys)));
    return keys;
}

function blobToDataUrl(blob: Blob) {
    return new Promise<string>((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(String(reader.result || ""));
        reader.onerror = () => reject(new Error(i18n.t("common.imageReadFailed")));
        reader.readAsDataURL(blob);
    });
}

async function resolveInputBlob(input: StoredImageInput): Promise<Blob> {
    if (input instanceof Blob) return input;

    if (typeof input === "object" && input.storageKey) {
        const storedBlob = await store.getItem<Blob>(input.storageKey);
        if (storedBlob) return storedBlob;
    }

    const candidates = typeof input === "string" ? [input] : [input.dataUrl, input.url];
    let lastError: unknown;
    for (const candidate of candidates) {
        if (!candidate) continue;
        try {
            if (candidate.startsWith("data:")) return dataUrlToBlob(candidate);
            const knownStorageKey = storageKeysByObjectUrl.get(candidate);
            if (knownStorageKey) {
                const storedBlob = await store.getItem<Blob>(knownStorageKey);
                if (storedBlob) return storedBlob;
            }
            const response = await fetch(candidate);
            if (!response.ok) throw new Error(`${response.status} ${response.statusText}`.trim());
            return await response.blob();
        } catch (error) {
            lastError = error;
        }
    }

    throw lastError instanceof Error ? lastError : new Error(i18n.t("common.imageReadFailed"));
}

function dataUrlToBlob(dataUrl: string) {
    const match = dataUrl.match(/^data:([^;,]*)(;base64)?,([\s\S]*)$/);
    if (!match) throw new Error(i18n.t("common.imageReadFailed"));
    const mimeType = match[1] || "application/octet-stream";
    if (!match[2]) return new Blob([decodeURIComponent(match[3])], { type: mimeType });
    const binary = atob(match[3]);
    const bytes = Uint8Array.from(binary, (character) => character.charCodeAt(0));
    return new Blob([bytes], { type: mimeType });
}

function cacheObjectUrl(storageKey: string, blob: Blob) {
    const previous = objectUrls.get(storageKey);
    if (previous) {
        URL.revokeObjectURL(previous);
        storageKeysByObjectUrl.delete(previous);
    }
    const url = URL.createObjectURL(blob);
    objectUrls.set(storageKey, url);
    storageKeysByObjectUrl.set(url, storageKey);
    return url;
}
