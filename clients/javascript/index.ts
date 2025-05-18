import {loadDERPub, loadECDSAPriv} from './src/utils/loadKeys';
import {doFinalizeAPI, doInitAPI} from './src/http/http';
import {performance} from 'perf_hooks';
import {randomBytes} from "crypto";
import {doSessionTest} from "./src/utils/session";


const config = {
    rsaPubPath: 'src/keys/client_rsa.pub',
    ecdsaPubPath: 'src/keys/client_ecdsa.pub',
    ecdsaPrivPath: 'src/keys/client_ecdsa.pem',
    initURL: 'http://localhost:8080/handshake/init',
    finURL: 'http://localhost:8080/handshake/finalize',
    sesURL: 'http://localhost:8080/session/test'
};

function generateBigMsg(sizeBytes: number): string {
    const b = randomBytes(sizeBytes);
    return b.toString('base64');
}

const MB10 = 10 * 1024 * 1024; // 10MB в байтах

async function main() {
    try {
        // Загрузка ключей
        const rsaPubDER = loadDERPub(config.rsaPubPath);
        const ecdsaPubDER = loadDERPub(config.ecdsaPubPath);
        const ecdsaPriv = loadECDSAPriv(config.ecdsaPrivPath);


        // Init handshake
        const startInit = performance.now();
        const initResp = await doInitAPI(
            config.initURL,
            rsaPubDER,
            ecdsaPubDER,
            ecdsaPriv
        );
        console.log('Init response:', initResp);
        console.log(`Init handshake time: ${performance.now() - startInit}ms`);

        // Finalize handshake
        const startFin = performance.now();
        const finResp = await doFinalizeAPI(
            config.finURL,
            config.sesURL,
            initResp,
            ecdsaPriv
        );
        console.log('Finalize response:', finResp);
        console.log(`Finalize handshake time: ${performance.now() - startFin}ms`);

        // Test session
        const startSesTest = performance.now();
        const testMessage = generateBigMsg(MB10);
        console.log(`Testing session with ${MB10 / (1024 * 1024)}MB message...`);

        await doSessionTest(finResp, testMessage);

        console.log('Session test completed successfully');
        console.log(`Session test time: ${performance.now() - startSesTest}ms`);

    } catch (err) {
        console.error('Error:', err instanceof Error ? err.message : String(err));
        process.exit(1);
    }
}

// Запуск приложения
main().catch(err => {
    console.error('Unhandled error:', err);
    process.exit(1);
});