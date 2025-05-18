import {createCipheriv, createHmac, randomBytes} from 'crypto';
import axios from 'axios';
import {signPayloadECDSA} from "./scrypto";
import {ISessionTestResponse} from "../models/response/ISessionTestResponse";

export interface Session {
    clientID: string;
    ecdsaPriv: Buffer | Uint8Array;
    ks: Buffer;
    kEnc: Buffer;
    kMac: Buffer;
    testURL: string;
}

function pkcs7Pad(data: Buffer): Buffer {
    const padSize = 16 - (data.length % 16);
    const padding = Buffer.alloc(padSize, padSize);
    return Buffer.concat([data, padding]);
}

// Создание сессии
export function NewSession(
    clientID: string,
    ecdsaPriv: Buffer | Uint8Array,
    ks: Buffer,
    testURL: string
): Session {
    // Деривация ключей из ks
    const kMac = createHmac('sha256', ks).update('mac').digest();
    const kEnc = createHmac('sha256', ks).update('enc').digest();

    return {
        clientID,
        ecdsaPriv,
        ks,
        kEnc,
        kMac,
        testURL
    };
}

// Выполнение теста сессии (аналог DoSessionTest в Go)
export async function doSessionTest(
    session: Session,
    plaintext: string
): Promise<void> {
    try {
        // 1. Подготовка метаданных
        const timestamp = Buffer.alloc(8);
        timestamp.writeBigInt64BE(BigInt(Date.now()), 0);

        const nonce = randomBytes(16);

        // 2. Формируем blob = timestamp + nonce + plaintext
        const blob = Buffer.concat([
            timestamp,
            nonce,
            Buffer.from(plaintext, 'utf-8')
        ]);

        // 3. Шифрование AES-256-CBC
        const iv = randomBytes(16);
        const cipher = createCipheriv('aes-256-cbc', session.kEnc, iv);
        const paddedData = pkcs7Pad(blob);
        const ciphertext = Buffer.concat([
            cipher.update(paddedData),
            cipher.final()
        ]);

        // 4. Расчет HMAC-SHA256
        const hmac = createHmac('sha256', session.kMac);
        hmac.update(iv);
        hmac.update(ciphertext);
        const tag = hmac.digest();

        // 5. Формирование итогового пакета
        const pkg = Buffer.concat([iv, ciphertext, tag]);
        const encryptedMessage = pkg.toString('base64');

        // 6. Подпись пакета
        const clientSignature = await signPayloadECDSA(session.ecdsaPriv, pkg);

        // 7. Отправка на сервер
        const response = await axios.post<ISessionTestResponse>(
            session.testURL,
            {
                encrypted_message: encryptedMessage,
                client_signature: clientSignature
            },
            {
                headers: {
                    'Content-Type': 'application/json',
                    'X-Client-ID': session.clientID
                }
            }
        );

        console.log('Успешная проверка сессии!');
        console.log('Сервер расшифровал:', response.data.plaintext);
    } catch (error) {
        if (axios.isAxiosError(error)) {
            throw new Error(`Ошибка сессии: ${error.response?.status} - ${JSON.stringify(error.response?.data)}`);
        }
        throw new Error(`Ошибка сессии: ${error instanceof Error ? error.message : String(error)}`);
    }
}