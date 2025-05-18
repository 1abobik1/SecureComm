import fs from 'fs';
import forge from 'node-forge';
import crypto from 'crypto';

export function loadDERPub(path: string): Buffer {
    const pem = fs.readFileSync(path, 'utf8');
    const pubKey = forge.pki.publicKeyFromPem(pem);
    const der = forge.asn1.toDer(forge.pki.publicKeyToAsn1(pubKey));
    return Buffer.from(der.getBytes(), 'binary');
}

export function loadECDSAPriv(path: string): Buffer {
    const pem = fs.readFileSync(path, 'utf8');
    const keyObject = crypto.createPrivateKey({ key: pem, format: 'pem', type: 'sec1' });
    return keyObject.export({ format: 'der', type: 'sec1' }) as Buffer;
}
