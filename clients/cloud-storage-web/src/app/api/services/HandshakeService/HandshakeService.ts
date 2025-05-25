import {generateECDSAKeys, generateRSAPublicKeyDER} from "@/app/api/services/HandshakeService/utils/loadKeys";
import {doFinalizeAPI, doInitAPI} from "@/app/api/services/HandshakeService/handshake/handshake";
import {HANDSHAKE_URL} from "@/app/api/http/urls";

const HANDSHAKE_BASE_URL = HANDSHAKE_URL;

const config = {
    initURL: `${HANDSHAKE_BASE_URL}/handshake/init`,
    finURL: `${HANDSHAKE_BASE_URL}/handshake/finalize`,
};

export async function doHandshake(){
    try {
        const rsaPubDER = await generateRSAPublicKeyDER();
        const [ecdsaPubDER, ecdsaPriv] = await generateECDSAKeys();

        //Init handshake
        const initResp = await doInitAPI(
            config.initURL,
            rsaPubDER,
            ecdsaPubDER,
            ecdsaPriv
        );

        // Finalize handshake
        const [_, ks64] = await doFinalizeAPI(
            config.finURL,
            initResp,
            ecdsaPriv
        );
        return ks64;
    } catch (err) {
        console.error('Ошибка:', err instanceof Error ? err.message : String(err));
    }
}

