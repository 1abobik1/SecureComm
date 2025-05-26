import {AxiosResponse} from 'axios';
import {CloudResponse, OneFileResponse} from "@/app/api/models/response/CloudResponse";
import {cloudApi} from '@/app/api/http/cloud';

export default class CloudService {
    static async getAllCloud(type: string): Promise<AxiosResponse<CloudResponse>> {
        return await cloudApi.get<CloudResponse>(`/files/all?type=${type}`);
    }


    static async uploadFiles(filename: string, mimeType: string, category: 'photo' | 'video' | 'text' | 'unknown', encryptedBlob: Uint8Array) {
        return await cloudApi.post<OneFileResponse>(`/files/one/encrypted`,  {
            headers: {
                'X-Orig-Filename': filename,
                'X-Orig-Mime': mimeType,
                'X-File-Category': category,
                'Content-Type': 'application/octet-stream',
            },
            encryptedBlob,
        });
    }


    static async deleteFile(type: string, obj_id: string) {
        return await cloudApi.delete(`files/one?id=${obj_id}&type=${type}`);
    }
}