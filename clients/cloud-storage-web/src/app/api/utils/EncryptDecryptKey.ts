// Генерация ключа для шифрования файлов (AES-256)
import {storeKey} from "@/app/api/utils/KeyStorage";

export async function generateFileEncryptionKey(): Promise<CryptoKey> {
    return await window.crypto.subtle.generateKey(
        {
            name: "AES-GCM",
            length: 256,
        },
        true,
        ["encrypt", "decrypt"]
    );
}

// Шифрование ключа файла с использованием пароля
export async function encryptKeyWithPassword(fileKey: CryptoKey, password: string): Promise<string> {
    try {
        const salt = crypto.getRandomValues(new Uint8Array(16));
        const iv = crypto.getRandomValues(new Uint8Array(12));

        // 1. Создаем ключ из пароля
        const passwordKey = await crypto.subtle.importKey(
            "raw",
            new TextEncoder().encode(password),
            { name: "PBKDF2" },
            false,
            ["deriveKey"]
        );

        const derivedKey = await crypto.subtle.deriveKey(
            {
                name: "PBKDF2",
                salt,
                iterations: 100000,
                hash: "SHA-256",
            },
            passwordKey,
            { name: "AES-GCM", length: 256 },
            false,
            ["encrypt"]
        );

        // 2. Шифруем основной ключ
        const exportedKey = await crypto.subtle.exportKey("raw", fileKey);
        const encryptedKey = await crypto.subtle.encrypt(
            { name: "AES-GCM", iv },
            derivedKey,
            exportedKey
        );

        // 3. Подготовка данных для хранения
        const result = {
            salt: Array.from(salt),
            iv: Array.from(iv),
            encryptedKey: Array.from(new Uint8Array(encryptedKey))
        };

        return JSON.stringify(result);
    } catch (error) {
        console.error("Encryption error:", error);
        throw new Error("Ошибка шифрования");
    }
}

export async function decryptKeyWithPassword(encryptedDataString: string, password: string): Promise<void> {
    try {
        // 1. Парсим и преобразуем данные
        const parsedData = JSON.parse(encryptedDataString);

        const salt = new Uint8Array(parsedData.salt);
        const iv = new Uint8Array(parsedData.iv);
        const encryptedKey = new Uint8Array(parsedData.encryptedKey).buffer;

        // 2. Создаем ключ из пароля
        const passwordKey = await crypto.subtle.importKey(
            "raw",
            new TextEncoder().encode(password),
            { name: "PBKDF2" },
            false,
            ["deriveKey"]
        );

        const derivedKey = await crypto.subtle.deriveKey(
            {
                name: "PBKDF2",
                salt,
                iterations: 100000,
                hash: "SHA-256",
            },
            passwordKey,
            { name: "AES-GCM", length: 256 },
            false,
            ["decrypt"]
        );

        // 3. Дешифруем основной ключ
        const decryptedKeyData = await crypto.subtle.decrypt(
            { name: "AES-GCM", iv },
            derivedKey,
            encryptedKey
        );

        const fileKey = await crypto.subtle.importKey(
            "raw",
            decryptedKeyData,
            { name: "AES-GCM", length: 256 },
            true,
            ["encrypt", "decrypt"]
        );

        storeKey(fileKey);
    } catch (error) {
        console.error("Decryption error details:", error);
        throw new Error("Неверный пароль");
    }
}
