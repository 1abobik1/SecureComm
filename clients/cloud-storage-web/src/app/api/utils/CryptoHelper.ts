import {getStoredKey} from "@/app/api/utils/KeyStorage";

function generateIV(): Uint8Array {
    return crypto.getRandomValues(new Uint8Array(12));
}

export const cryptoHelper = {
    async encryptFile(file: File): Promise<File> {
        const arrayBuffer = await file.arrayBuffer();
        const iv = generateIV();
        const key = await getStoredKey();

        const encrypted = await crypto.subtle.encrypt(
            { name: 'AES-GCM', iv },
            key,
            arrayBuffer
        );

        const combined = new Uint8Array(iv.length + encrypted.byteLength);
        combined.set(iv, 0);
        combined.set(new Uint8Array(encrypted), iv.length);

        return new File([combined], file.name, { type: file.type });
    },

    async decryptFile(file: File): Promise<Blob> {
        const combined = new Uint8Array(await file.arrayBuffer());
        const iv = combined.slice(0, 12);
        const data = combined.slice(12);

        const key = await getStoredKey();

        const decrypted = await crypto.subtle.decrypt(
            { name: 'AES-GCM', iv },
            key,
            data
        );

        return new Blob([decrypted], { type: file.type });
    }
};
