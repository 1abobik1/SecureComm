// app/api/store/context.ts
import {createContext} from 'react';
import Store from "@/app/api/store/store";

const store = new Store()

export const Context = createContext<Store>(store);