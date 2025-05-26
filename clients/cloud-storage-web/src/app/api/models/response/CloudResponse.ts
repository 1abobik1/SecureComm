import {FileData} from '../FileData'

export interface CloudResponse {
    file_data: FileData[];
    message: string;
    status: number;
}
export interface OneFileResponse {
    created_at: string,
    mime_type: string,
    name: string,
    obj_id: string,
    url: string
}
