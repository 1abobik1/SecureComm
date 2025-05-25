// Преобразование base64 в CryptoKey
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
export async function decryptStoredKey(password: string): Promise<CryptoKey> {
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

        return await crypto.subtle.importKey(
            "raw",
            decryptedKeyData,
            { name: "AES-GCM", length: 256 },
            true,
            ["encrypt", "decrypt"]
        );
    } catch (error) {
        console.error("Decryption error details:", error);
        throw new Error("Неверный пароль");
    }
}

// Основная функция для инициализации ключа из base64
export async function initializeKeyFromBase64(base64Key: string, password: string): Promise<void> {
    const cryptoKey = await importKeyFromBase64(base64Key);
    await encryptAndStoreKey(cryptoKey, password);
}

export async function decryptKsDataLogin(
    encryptedData: {
        k_enc_iv: string;
        k_enc_data: string;
        k_mac_iv: string;
        k_mac_data: string;
    },
    password: string
): Promise<{
    k_enc_iv: Uint8Array;
    k_enc_data: Uint8Array;
    k_mac_iv: Uint8Array;
    k_mac_data: Uint8Array;
}> {
    try {
        // 1. Создаем ключ из пароля (общий для всех полей)
        const passwordKey = await crypto.subtle.importKey(
            "raw",
            new TextEncoder().encode(password),
            { name: "PBKDF2" },
            false,
            ["deriveKey"]
        );

        // 2. Функция для расшифровки одного поля
        const decryptField = async (ivBase64: string, dataBase64: string) => {
            // Декодируем base64 в Uint8Array
            const iv = Uint8Array.from(atob(ivBase64), c => c.charCodeAt(0));
            const encryptedData = Uint8Array.from(atob(dataBase64), c => c.charCodeAt(0));

            // Используем salt из iv (или можно добавить отдельное поле для salt)
            const salt = iv.slice(0, 16); // Берем первые 16 байт iv как salt

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

            const decrypted = await crypto.subtle.decrypt(
                { name: "AES-GCM", iv },
                derivedKey,
                encryptedData
            );

            return new Uint8Array(decrypted);
        };

        // 3. Расшифровываем все поля параллельно
        const [k_enc_iv, k_enc_data, k_mac_iv, k_mac_data] = await Promise.all([
            decryptField(encryptedData.k_enc_iv, encryptedData.k_enc_data),
            decryptField(encryptedData.k_enc_data, encryptedData.k_enc_data),
            decryptField(encryptedData.k_mac_iv, encryptedData.k_mac_data),
            decryptField(encryptedData.k_mac_data, encryptedData.k_mac_data),
        ]);

        return {
            k_enc_iv,
            k_enc_data,
            k_mac_iv,
            k_mac_data
        };
    } catch (error) {
        console.error("Decryption error:", error);
        throw new Error("Неверный пароль");
    }
}