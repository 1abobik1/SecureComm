import {setKs} from "@/app/api/utils/ksInStorage";
import {deriveAESKey, deriveHMACKey} from "@/app/api/services/HandshakeService/utils/scrypto";

export async function encryptAndStoreKey(
    kmac: CryptoKey,
    kenc: CryptoKey,
    password: string,
): Promise<void> {
    try {

        // 2. Экспортируем ключи в сырые байты
        const [exportedMac, exportedEnc] = await Promise.all([
            crypto.subtle.exportKey("raw", kmac),
            crypto.subtle.exportKey("raw", kenc)
        ]);

        const macBytes = new Uint8Array(exportedMac);
        const encBytes = new Uint8Array(exportedEnc);

        // 3. Шифруем каждый ключ с помощью пароля
        const [encryptedMac, encryptedEnc] = await Promise.all([
            encryptWithPassword(password, macBytes),
            encryptWithPassword(password, encBytes)
        ]);

        // 4. Формируем объект для хранения
        const storageData = {
            k_enc_iv: encryptedEnc.iv,
            k_enc_data: encryptedEnc.ciphertext,
            k_mac_iv: encryptedMac.iv,
            k_mac_data: encryptedMac.ciphertext
        };

        // 5. Сохраняем в localStorage
        localStorage.setItem('encryptedFileKey', JSON.stringify(storageData));
    } catch (error) {
        console.error('Failed to encrypt and store keys:', error);
        throw new Error('Ошибка при шифровании ключей');
    }
}

async function encryptWithPassword(
    password: string,
    data: Uint8Array
): Promise<{ iv: string; ciphertext: string }> {
    // 1. Хешируем пароль в SHA-256
    const pwUtf8 = new TextEncoder().encode(password);
    const pwHash = await crypto.subtle.digest("SHA-256", pwUtf8);

    // 2. Генерируем IV
    const iv = crypto.getRandomValues(new Uint8Array(12)); // 12 байт для AES-GCM

    // 3. Импортируем как AES-GCM ключ (делаем extractable: true)
    const aesKey = await crypto.subtle.importKey(
        "raw",
        pwHash,
        { name: "AES-GCM" },
        true, // Ключ можно экспортировать
        ["encrypt"]
    );

    // 4. Шифруем данные
    const ciphertext = await crypto.subtle.encrypt(
        { name: "AES-GCM", iv },
        aesKey,
        data
    );

    // 5. Возвращаем в base64
    return {
        iv: btoa(String.fromCharCode(...iv)),
        ciphertext: btoa(String.fromCharCode(...new Uint8Array(ciphertext)))
    };
}

async function decryptAESGCM(
    password: string,
    ivB64: string,
    ctB64: string
): Promise<Uint8Array> {
    const b64ToArr = (b64: string) =>
        Uint8Array.from(atob(b64), (c) => c.charCodeAt(0));

    const iv = b64ToArr(ivB64);
    const ct = b64ToArr(ctB64);

    const pwUtf8 = new TextEncoder().encode(password);
    const pwHash = await crypto.subtle.digest("SHA-256", pwUtf8);

    const aesKey = await crypto.subtle.importKey(
        "raw",
        pwHash,
        { name: "AES-GCM" },
        false,
        ["decrypt"]
    );

    const plain = await crypto.subtle.decrypt(
        { name: "AES-GCM", iv },
        aesKey,
        ct
    );

    return new Uint8Array(plain);
}

export async function decryptStoredKey(
    ks: {
        k_enc_iv: string;
        k_enc_data: string;
        k_mac_iv: string;
        k_mac_data: string;
    },
    password: string,
    store: { hasCryptoKey: boolean }
): Promise<boolean> {
    try {
        const [kEncBytes, kMacBytes] = await Promise.all([
            decryptAESGCM(password, ks.k_enc_iv, ks.k_enc_data),
            decryptAESGCM(password, ks.k_mac_iv, ks.k_mac_data),
        ]);

        const [kMac, kEnc] = await Promise.all([
            deriveHMACKey(kMacBytes),
            deriveAESKey(kEncBytes)
        ]);

        await setKs([kMac, kEnc], 60 * 8);
        store.hasCryptoKey = true;

        return true;
    } catch (error) {
        console.error('Failed to decrypt stored keys:', error);
        throw new Error('Неверный пароль или поврежденные данные');
    }
}