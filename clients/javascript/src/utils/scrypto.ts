import {ec as EC} from 'elliptic';
import crypto from "crypto";

export interface DerSig {
    R: bigint;
    S: bigint;
}

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

export function signatureToDer (sig: DerSig): Buffer  {
    const rBytes = Buffer.from(sig.R.toString(16)).toString('hex');
    const sBytes = Buffer.from(sig.S.toString(16)).toString('hex');

    const rLength = rBytes.length;
    const sLength = sBytes.length;

    const totalLength = 2 + rLength + 2 + sLength;
    const der = Buffer.alloc(2 + totalLength);

    let offset = 0;
    der[offset++] = 0x30; // SEQUENCE
    der[offset++] = totalLength;
    der[offset++] = 0x02; // INTEGER
    der[offset++] = rLength;
    Buffer.from(rBytes, 'hex').copy(der, offset);
    offset += rLength;
    der[offset++] = 0x02; // INTEGER
    der[offset++] = sLength;
    Buffer.from(sBytes, 'hex').copy(der, offset);

    return der;
}

export const signPayloadECDSA = (privKey: EC.KeyPair, data: Buffer): string => {
    const hash = crypto.createHash('sha256').update(data).digest('hex');
    const signature = privKey.sign(hash, 'hex', { canonical: true });
    const der = signatureToDer({
        R: BigInt('0x' + signature.r.toString('hex')),
        S: BigInt('0x' + signature.s.toString('hex'))
    });
    return der.toString('base64');
};

export const generateRandomBytes = (size: number): [string, Buffer] => {
    const buf = crypto.randomBytes(size);
    return [buf.toString('base64'), buf];
};

export function encryptRSA(pubKey: string, plaintext: Buffer): string {
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

export function decodeBase64(s: string): Buffer {
    return Buffer.from(s, 'base64');
}

export function loadRSAPubDER(pem: Buffer): Buffer {
    const publicKey = crypto.createPublicKey({
        key: pem,
        format: 'pem',
        type: 'spki',
    });

    return publicKey.export({ format: 'der', type: 'spki' }) as Buffer;
}

const ec = new EC('p256');

export function parseECDSAPriv(pem: Buffer): EC.KeyPair {
    const key = crypto.createPrivateKey({
        key: pem,
        format: 'pem',
        type: 'sec1',
    });
    const der = key.export({ format: 'der', type: 'sec1' }) as Buffer;
    const privScalar = der.slice(-32);
    return ec.keyFromPrivate(privScalar);
}



