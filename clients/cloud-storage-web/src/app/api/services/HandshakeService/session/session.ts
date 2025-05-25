import {
    deriveAESKey,
    deriveHMACKey,
    deriveKeyBytes,
    encryptAES256CBC,
    generateNonce,
    signDataWithECDSA
} from "../utils/scrypto";
import {postJSON} from "../utils/postJSON"
import {IInitResponse} from "../models/response/IInitResponse";
import {AxiosResponse} from "axios";

export async function doSession(
    sesURL: string,
    initResp: IInitResponse,
    ecdsaPriv: CryptoKey,
    ks: string,
    payload: Uint8Array
): Promise<string> {
    // 1. Подготовка данных (timestamp + nonce + payload)
    const timestamp = new Uint8Array(8);
    const timestampValue = BigInt(Date.now());
    for (let i = 0; i < 8; i++) {
        timestamp[7 - i] = Number((timestampValue >> BigInt(8 * i)) & BigInt(0xFF));
    }

    const [_, nonce] = generateNonce(16);
    const [__, iv] = generateNonce(16);

    // Формируем blob = timestamp + nonce + payload
    const blob = new Uint8Array([...timestamp, ...nonce, ...payload]);

    // 2. Деривация ключей из ks
    const ksBytes = Uint8Array.from(atob(ks), c => c.charCodeAt(0));
    const [kMacBytes, kEncBytes] = await Promise.all([
        deriveKeyBytes(ksBytes, "mac"),
        deriveKeyBytes(ksBytes, "enc")
    ]);

    const [kMac, kEnc] = await Promise.all([
        deriveHMACKey(kMacBytes),
        deriveAESKey(kEncBytes)
    ]);

    // 3. Шифрование blob
    const ciphertext = await encryptAES256CBC(kEnc, iv, blob);

    // 4. Создание HMAC (IV || ciphertext)
    const hmacData = new Uint8Array([...iv, ...ciphertext]);
    const tag = await crypto.subtle.sign(
        { name: 'HMAC' },
        kMac,
        hmacData
    );

    // 5. Формирование сообщения (iv + ciphertext + tag)
    const encryptedMessage = new Uint8Array([...iv, ...ciphertext, ...new Uint8Array(tag)]);

    // 6. Подпись
     // Подписываем только iv + ciphertext + tag
    const clientSignature = await signDataWithECDSA(encryptedMessage, ecdsaPriv);

    // 7. Отправка
    const response: AxiosResponse<{ plaintext: string }> = await postJSON(
        sesURL,
        {
            client_signature: clientSignature,
            encrypted_message: btoa(String.fromCharCode(...encryptedMessage))
        },
        { 'X-Client-ID': initResp.client_id }
    );

    if (response.status !== 200) {
        throw new Error(`Session test failed: ${response.status} ${JSON.stringify(response.data)}`);
    }
    return response.data.plaintext;
}