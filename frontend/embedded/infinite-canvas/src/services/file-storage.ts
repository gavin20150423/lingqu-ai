import localforage from "localforage";
import { nanoid } from "nanoid";

export type UploadedFile = { url: string; storageKey: string; bytes: number; mimeType: string; width?: number; height?: number; durationMs?: number };

const store = localforage.createInstance({ name: "infinite-canvas", storeName: "media_files" });
const objectUrls = new Map<string, string>();

export async function uploadMediaFile(input: string | Blob, prefix = "file"): Promise<UploadedFile> {
    const source = typeof input === "string" ? await (await fetch(input)).blob() : input;
    const blob = prefix.startsWith("video") ? await normalizeVideoBlob(source) : source;
    const storageKey = `${prefix}:${nanoid()}`;
    await store.setItem(storageKey, blob);
    const url = URL.createObjectURL(blob);
    objectUrls.set(storageKey, url);
    const meta = blob.type.startsWith("video/") ? await readVideoMeta(url) : blob.type.startsWith("audio/") ? await readAudioMeta(url) : {};
    return { url, storageKey, bytes: blob.size, mimeType: blob.type || "application/octet-stream", ...meta };
}

export async function resolveMediaUrl(storageKey?: string, fallback = "") {
    if (!storageKey) return fallback;
    const cached = objectUrls.get(storageKey);
    if (cached) return cached;
    const stored = await store.getItem<Blob>(storageKey);
    if (!stored) return fallback;
    const blob = storageKey.startsWith("video") ? await normalizeVideoBlob(stored) : stored;
    if (blob !== stored) await store.setItem(storageKey, blob);
    const url = URL.createObjectURL(blob);
    objectUrls.set(storageKey, url);
    return url;
}

export async function getMediaBlob(storageKey: string) {
    return store.getItem<Blob>(storageKey);
}

export async function setMediaBlob(storageKey: string, blob: Blob) {
    const normalized = storageKey.startsWith("video") ? await normalizeVideoBlob(blob) : blob;
    await store.setItem(storageKey, normalized);
    const url = URL.createObjectURL(normalized);
    objectUrls.set(storageKey, url);
    return url;
}

/**
 * Some upstream video endpoints return a valid MP4 body with an
 * application/octet-stream (or empty) content type. Browsers will not always
 * sniff that blob when it is used as a video source, so give it a playable
 * media type while preserving an explicitly valid video type.
 */
export async function normalizeVideoBlob(blob: Blob, mimeHint?: string) {
    const currentType = blob.type.toLowerCase();
    const detectedType = await detectVideoMime(blob);
    const hintedType = mimeHint?.toLowerCase();
    const mimeType = detectedType || (currentType.startsWith("video/") ? currentType : hintedType?.startsWith("video/") ? hintedType : "video/mp4");
    return currentType === mimeType ? blob : new Blob([blob], { type: mimeType });
}

async function detectVideoMime(blob: Blob) {
    const bytes = new Uint8Array(await blob.slice(0, 16).arrayBuffer());
    if (bytes.length >= 8 && String.fromCharCode(...bytes.slice(4, 8)) === "ftyp") return "video/mp4";
    if (bytes.length >= 4 && bytes[0] === 0x1a && bytes[1] === 0x45 && bytes[2] === 0xdf && bytes[3] === 0xa3) return "video/webm";
    if (bytes.length >= 4 && String.fromCharCode(...bytes.slice(0, 4)) === "OggS") return "video/ogg";
    return "";
}

export async function deleteStoredMedia(keys: Iterable<string>) {
    await Promise.all(
        Array.from(new Set(keys)).map(async (key) => {
            const url = objectUrls.get(key);
            if (url) URL.revokeObjectURL(url);
            objectUrls.delete(key);
            await store.removeItem(key);
        }),
    );
}

export async function cleanupUnusedMedia(usedData: unknown) {
    const usedKeys = collectMediaStorageKeys(usedData);
    const unused: string[] = [];
    await store.iterate((_value, key) => {
        if (!usedKeys.has(key)) unused.push(key);
    });
    await Promise.all(unused.map((key) => store.removeItem(key)));
}

export function collectMediaStorageKeys(value: unknown, keys = new Set<string>()) {
    if (!value || typeof value !== "object") return keys;
    if ("storageKey" in value && typeof value.storageKey === "string" && value.storageKey.includes(":")) keys.add(value.storageKey);
    Object.values(value).forEach((item) => (Array.isArray(item) ? item.forEach((child) => collectMediaStorageKeys(child, keys)) : collectMediaStorageKeys(item, keys)));
    return keys;
}

function readVideoMeta(url: string) {
    return new Promise<{ width: number; height: number; durationMs?: number }>((resolve) => {
        const video = document.createElement("video");
        const done = () => resolve({ width: video.videoWidth || 1280, height: video.videoHeight || 720, durationMs: Number.isFinite(video.duration) ? Math.round(video.duration * 1000) : undefined });
        video.onloadedmetadata = done;
        video.onerror = done;
        video.src = url;
    });
}

function readAudioMeta(url: string) {
    return new Promise<{ durationMs?: number }>((resolve) => {
        const audio = document.createElement("audio");
        const done = () => resolve({ durationMs: Number.isFinite(audio.duration) ? Math.round(audio.duration * 1000) : undefined });
        audio.onloadedmetadata = done;
        audio.onerror = done;
        audio.src = url;
    });
}
