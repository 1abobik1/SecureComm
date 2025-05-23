import {AxiosResponse} from 'axios';
import {CloudResponse, OneFileResponse} from "@/app/api/models/response/CloudResponse";
import {cloudApi} from '@/app/api/http/cloud';

export default class CloudService {
    static async getAllCloud(type: string): Promise<AxiosResponse<CloudResponse>> {
        return await cloudApi.get<CloudResponse>(`/files/all?type=${type}`);
    }


    static async getOneFile(id:string,type: string): Promise<AxiosResponse<CloudResponse>> {
        return await cloudApi.get<CloudResponse>(`/files/one?id=${id}type=${type}`);
    }
                                                    


    static async uploadFiles(formData: FormData, config = {}) {
        return await cloudApi.post(`/files/many`, formData, {
            headers: {
                'Content-Type': 'multipart/form-data',
            },
            ...config,
        });
    }


    static async deleteFile(type: string, obj_id: string) {
        return await cloudApi.delete(`files/one?id=${obj_id}&type=${type}`);
    }
}