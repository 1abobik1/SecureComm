import crypto, {createHash, KeyObject} from "crypto"; // Для кодирования/декодирования DER
import {ec as EC} from 'elliptic';
// @ts-ignore
import asn1 from 'asn1.js';

export function derToSignature (der: Buffer): DerSig {
    let offset = 0;

    // Check SEQUENCE tag
    if (der[offset++] !== 0x30) throw new Error('Invalid DER: expected SEQUENCE');

    // Read sequence length
    const seqLength = der[offset++];
    if (seqLength >= 0x80) throw new Error('Long form length not supported');

    // Read INTEGER tag for R
    if (der[offset++] !== 0x02) throw new Error('Expected INTEGER for R');
    const rLength = der[offset++];
    const rBytes = der.slice(offset, offset + rLength);
    offset += rLength;
    const R = BigInt('0x' + rBytes.toString('hex'));

    // Read INTEGER tag for S
    if (der[offset++] !== 0x02) throw new Error('Expected INTEGER for S');
    const sLength = der[offset++];
    const sBytes = der.slice(offset, offset + sLength);
    const S = BigInt('0x' + sBytes.toString('hex'));
    return { R, S };
}


// Определяем интерфейс для подписи
export interface DerSig {
    R: bigint;
    S: bigint;
}

// Функция для подписи данных с помощью ECDSA (аналог Go-версии)
export async function signPayloadECDSA(
    privateKeyPEM: string | Buffer | Uint8Array, // Приватный ключ в PEM-формате
    data: Buffer,          // Данные для подписи
): Promise<string> {
    try {
        // 1. Хешируем данные (SHA-256)
        const hash = createHash('sha256').update(data).digest();

        // 2. Создаем экземпляр ECDSA (например, для кривой P-256)
        const ec = new EC('p256');
        const key = ec.keyFromPrivate(privateKeyPEM, 'pem');

        // 3. Подписываем хеш
        const signature = key.sign(hash, { canonical: true });

        // 4. Получаем R и S в виде bigint
        const derSig: DerSig = {
            R: BigInt(`0x${signature.r.toString(16)}`),
            S: BigInt(`0x${signature.s.toString(16)}`),
        };

        // 5. Кодируем в DER-формат (ASN.1)
        const derEncoded = encodeECDSASignature(derSig);

        // 6. Возвращаем в base64
        return derEncoded.toString('base64');
    } catch (err) {
        throw new Error(`ECDSA signing failed: ${err instanceof Error ? err.message : String(err)}`);
    }
}

// Вспомогательная функция для кодирования подписи в DER (ASN.1)
function encodeECDSASignature(sig: DerSig): Buffer {
    const ECDSASignature = asn1.define('ECDSASignature', function (this: any) {
        this.seq().obj(
            this.key('R').int(), // Соответствует полю R в DerSig
            this.key('S').int(), // Соответствует полю S в DerSig
        );
    });

    return ECDSASignature.encode(
        {
            R: sig.R,
            S: sig.S,
        },
        'der',
    );
}

export const generateRandomBytes = (size: number): [string, Buffer] => {
    const buf = crypto.randomBytes(size);
    return [buf.toString('base64'), buf];
};

export function encryptRSA(pubKey: string| KeyObject, plaintext: Buffer): string {
    const encrypted = crypto.publicEncrypt(
        {
            key: pubKey,
            padding: crypto.constants.RSA_PKCS1_OAEP_PADDING,
            oaepHash: 'sha256',
        },
        plaintext
    );
    return encrypted.toString('base64');
}

export function loadRSAPubDER(pem: Buffer): Buffer {
    const publicKey = crypto.createPublicKey({
        key: pem,
        format: 'pem',
        type: 'spki',
    });

    return publicKey.export({ format: 'der', type: 'spki' }) as Buffer;
}

