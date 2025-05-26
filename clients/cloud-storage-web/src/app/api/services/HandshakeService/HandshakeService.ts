import {generateECDSAKeys, generateRSAPublicKeyDER} from "@/app/api/services/HandshakeService/utils/loadKeys";
import {doFinalizeAPI, doInitAPI} from "@/app/api/services/HandshakeService/handshake/handshake";
import {USAGE_CLOUD_HANDSHAKE_URL} from "@/app/api/http/urls";

const HANDSHAKE_BASE_URL = USAGE_CLOUD_HANDSHAKE_URL;

const config = {
    initURL: `${HANDSHAKE_BASE_URL}/handshake/init`,
    finURL: `${HANDSHAKE_BASE_URL}/handshake/finalize`,
};

export async function doHandshake(){
    try {
        const rsaPubDER = await generateRSAPublicKeyDER();
        const [ecdsaPubDER, ecdsaPriv] = await generateECDSAKeys();

        const token = localStorage.getItem('token');
        if (!token) {
            throw new Error('No token found');
        }

        //Init handshake
        const initResp = await doInitAPI(
            config.initURL,
            rsaPubDER,
            ecdsaPubDER,
            ecdsaPriv,
            token
        );

        // Finalize handshake
        const ks64 = await doFinalizeAPI(
            config.finURL,
            initResp,
            ecdsaPriv,
            token
        );
        return ks64;
    } catch (err) {
        console.error('Ошибка:', err instanceof Error ? err.message : String(err));
    }
}

