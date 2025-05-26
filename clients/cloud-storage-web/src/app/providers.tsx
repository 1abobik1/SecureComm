// app/providers.tsx
'use client';
import {Context} from '@/app/api/store/context';
import Store from "@/app/api/store/store";


export function Providers({ children }: { children: React.ReactNode }) {
    const store = new Store();

    return (
        <Context.Provider value={store}>
            {children}
        </Context.Provider>
    );
}