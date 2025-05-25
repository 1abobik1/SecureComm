// Преобразование base64 в CryptoKey
import {setKs, setKsLogin} from "@/app/api/utils/ksInStorage";

//signup ks

export async function importKeyFromBase64(base64Key: string): Promise<CryptoKey> {
    try {
        // Декодируем base64 в Uint8Array
        const binaryString = atob(base64Key);
        const bytes = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
            bytes[i] = binaryString.charCodeAt(i);
        }

        // Импортируем ключ
        return await window.crypto.subtle.importKey(
            "raw",
            bytes,
            { name: "AES-GCM", length: 256 },
            true,
            ["encrypt", "decrypt"]
        );
    } catch (error) {
        console.error("Key import error:", error);
        throw new Error("Ошибка импорта ключа");
    }
}

// Шифрование ключа файла с использованием пароля (теперь сохраняет в localStorage)
export async function encryptAndStoreKey(key: CryptoKey, password: string): Promise<void> {
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
        const exportedKey = await crypto.subtle.exportKey("raw", key);
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

        // Сохраняем в localStorage
        localStorage.setItem('encryptedFileKey', JSON.stringify(result));


    } catch (error) {
        console.error("Encryption error:", error);
        throw new Error("Ошибка шифрования");
    }
}

// Дешифровка ключа из localStorage
export async function decryptStoredKey(password: string): Promise<boolean> {
    try {
        const storedData = localStorage.getItem('encryptedFileKey');
        if (!storedData) {
            throw new Error("No encrypted key found in localStorage");
        }

        // 1. Парсим и преобразуем данные
        const parsedData = JSON.parse(storedData);

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

        const ks = await crypto.subtle.importKey(
            "raw",
            decryptedKeyData,
            { name: "AES-GCM", length: 256 },
            true,
            ["encrypt", "decrypt"]
        )

        setKs(ks, 60 * 8)
        return true
    } catch (error) {
        console.error("Decryption error details:", error);
        throw new Error("Неверный пароль");
    }
}

//login ks

async function decryptAESGCM(
    password: string,
    ivB64: string,
    ctB64: string
): Promise<Uint8Array> {
    // Вспомогательная функция base64 → Uint8Array
    const b64ToArr = (b64: string) =>
        Uint8Array.from(atob(b64), (c) => c.charCodeAt(0));

    const iv = b64ToArr(ivB64);
    const ct = b64ToArr(ctB64);

    // 1) Хешируем пароль в SHA-256
    const pwUtf8 = new TextEncoder().encode(password);
    const pwHash = await crypto.subtle.digest("SHA-256", pwUtf8);

    // 2) Импортируем как AES-GCM ключ
    const aesKey = await crypto.subtle.importKey(
        "raw",
        pwHash,
        "AES-GCM",
        false,
        ["decrypt"]
    );

    // 3) Дешифруем
    const plain = await crypto.subtle.decrypt(
        { name: "AES-GCM", iv },
        aesKey,
        ct
    );

    return new Uint8Array(plain);
}

/**
 * Фетчим зашифрованные ключи, расшифровываем их
 */
export async function decryptStoredKeyLogin(
    ks:{
        k_enc_iv: string;
        k_enc_data: string;
        k_mac_iv: string;
        k_mac_data: string;
    },
    password: string
): Promise<boolean> {


    //расшифровываем оба ключа
    const [kEnc, kMac] = await Promise.all([
        decryptAESGCM(password, ks.k_enc_iv, ks.k_enc_data),
        decryptAESGCM(password, ks.k_mac_iv, ks.k_mac_data),
    ]);
    setKsLogin([ kEnc, kMac ], 60 * 8);
    return true;
}