import {generateECDSAKeys, generateRSAPublicKeyDER,} from './src/utils/loadKeys';
import {doFinalizeAPI, doInitAPI} from "./src/handshake/http";
import {doSessionTest} from "./src/session/session";
import {generateNonce} from "./src/utils/scrypto";


const config = {
    initURL: 'http://localhost:8080/handshake/init',
    finURL: 'http://localhost:8080/handshake/finalize',
    sesURL: 'http://localhost:8080/session/test'
};

function generateBigMsg(sizeBytes: number): string {
    const [_,b] = generateNonce(sizeBytes);
    return btoa(String.fromCharCode(...b));
}

const MB10 = 10 * 1024;

async function main() {
    try {
        const rsaPubDER = await generateRSAPublicKeyDER();
        const [ecdsaPubDER, ecdsaPriv] = await generateECDSAKeys();

        //Init handshake
        const startInit = performance.now();
        const initResp = await doInitAPI(
            config.initURL,
            rsaPubDER,
            ecdsaPubDER,
            ecdsaPriv
        );

        console.log(`Init handshake time: ${performance.now() - startInit}ms`);

        // Finalize handshake
        const startFin = performance.now();
        const [finResp,ks] = await doFinalizeAPI(
            config.finURL,
            initResp,
            ecdsaPriv
        );

        console.log(`Finalize handshake time: ${performance.now() - startFin}ms`);

        //Test session
        const startSesTest = performance.now();
        const testMessage = generateBigMsg(MB10);

        await doSessionTest(
            config.sesURL,
            initResp,
            ecdsaPriv,
            ks,
            testMessage
        );

        console.log(`Session test time: ${performance.now() - startSesTest}ms`);

    } catch (err) {
        console.error('Ошибка:', err instanceof Error ? err.message : String(err));
        process.exit(1);
    }
}

// Запуск приложения
main().catch(err => {
    console.error('Unhandled error:', err);
    process.exit(1);
});