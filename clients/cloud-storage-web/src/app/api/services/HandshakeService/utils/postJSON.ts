import axios, {AxiosResponse} from "axios";

export const postJSON = async (
    url: string,
    payload: any,
    token: string,
    headers?: Record<string, string>
): Promise<AxiosResponse> => {
    try {
        return await axios.post(url, payload, {
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`,
                ...headers
            }
        });
    } catch (error) {
        if (axios.isAxiosError(error)) {
            throw new Error(
                `HTTP request failed: ${error.response?.status} ${JSON.stringify(error.response?.data)}`
            );
        }
        throw error;
    }
};