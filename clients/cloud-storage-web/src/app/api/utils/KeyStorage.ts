let volatileKey: CryptoKey | null = null;

export function getStoredKey(): Promise<CryptoKey> {
    if (!volatileKey) {
        return Promise.reject(new Error("Ключа нет"));
    }
    return Promise.resolve(volatileKey);
}

export function storeKey(key: CryptoKey) {
    volatileKey = key;
}

export function clearKey() {
    volatileKey = null;
}
