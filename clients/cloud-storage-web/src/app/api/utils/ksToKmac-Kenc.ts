import {deriveAESKey, deriveHMACKey, deriveKeyBytes} from "@/app/api/services/HandshakeService/utils/scrypto";
import {setKs} from "@/app/api/utils/ksInStorage";


export async function ksToKmacKenc(ks: string, store: { hasCryptoKey: boolean }): Promise<CryptoKey[]> {
    const ksBytes = Uint8Array.from(atob(ks), c => c.charCodeAt(0));
    const [kMacBytes, kEncBytes] = await Promise.all([
        deriveKeyBytes(ksBytes, "mac"),
        deriveKeyBytes(ksBytes, "enc")
    ]);

    const [kMac, kEnc] = await Promise.all([
        deriveHMACKey(kMacBytes),
        deriveAESKey(kEncBytes)
    ]);
    await setKs([kMac, kEnc], 60 * 8);
    store.hasCryptoKey = true;
    return [kMac, kEnc];
}