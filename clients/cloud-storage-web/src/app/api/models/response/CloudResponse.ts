import {FileData} from '../FileData'

export interface CloudResponse {
    file_data: FileData[];
    message: string;
    status: number;
}
export interface OneFileResponse {
    file_data: FileData;
    message: string;
    status: number;
}
