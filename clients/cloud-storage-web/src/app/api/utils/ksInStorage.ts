import {LocalStorage} from 'ttl-localstorage';

// Сохраняем CryptoKey как Base64 строку
export async function setKs(keys: [CryptoKey, CryptoKey], ttlSeconds: number) {
    // Экспортируем ключи в бинарный формат (ArrayBuffer)
    const [key1, key2] = keys;
    const rawKey1 = await crypto.subtle.exportKey('raw', key1);
    const rawKey2 = await crypto.subtle.exportKey('raw', key2);

    // Конвертируем в Base64 строку
    const base64Key1 = arrayBufferToBase64(rawKey1);
    const base64Key2 = arrayBufferToBase64(rawKey2);

    // Сохраняем в LocalStorage
    LocalStorage.put('ks', {key1: base64Key1, key2: base64Key2}, ttlSeconds);
}

// Восстанавливаем CryptoKey из Base64 строки
export async function getKs(): Promise<[CryptoKey, CryptoKey] | null> {
    const stored = LocalStorage.get('ks');
    if (!stored) return null;

    try {
        const {key1, key2} = stored;

        // Конвертируем Base64 обратно в ArrayBuffer
        const rawKey1 = base64ToArrayBuffer(key1);
        const rawKey2 = base64ToArrayBuffer(key2);

        // Импортируем ключи (укажите ваш алгоритм!)
        const cryptoKey1 = await crypto.subtle.importKey(
            'raw',
            rawKey1,
            {
                name: 'HMAC',
                hash: { name: 'SHA-256' }
            },
            true,
            ['sign']
        );

        const cryptoKey2 = await crypto.subtle.importKey(
            'raw',
            rawKey2,
            { name: 'AES-CBC', length: 256 },
            true,
            ['encrypt']
        );

        return [cryptoKey1, cryptoKey2];
    } catch (error) {
        console.error('Ошибка восстановления ключей:', error);
        return null;
    }
}

// Вспомогательные функции для конвертации
function arrayBufferToBase64(buffer: ArrayBuffer): string {
    return btoa(String.fromCharCode(...new Uint8Array(buffer)));
}

function base64ToArrayBuffer(base64: string): ArrayBuffer {
    const binaryString = atob(base64);
    const bytes = new Uint8Array(binaryString.length);
    for (let i = 0; i < binaryString.length; i++) {
        bytes[i] = binaryString.charCodeAt(i);
    }
    return bytes.buffer;
}

export function removeKs() {
    LocalStorage.removeKey('ks');
}