export interface AuthResponse {
    access_token: string;
    ks:{
        k_enc_iv: string;
        k_enc_data: string;
        k_mac_iv: string;
        k_mac_data: string;
    }
}