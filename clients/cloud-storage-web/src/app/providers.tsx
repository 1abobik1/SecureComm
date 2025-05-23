// app/providers.tsx
'use client'

import React, {createContext, ReactNode, useContext} from 'react'
import Store from "@/app/api/store/store"

const store = new Store()
const StoreContext = createContext<Store>(store)

export function useStore() {
    return useContext(StoreContext)
}

export function Providers({ children }: { children: ReactNode }) {
    return (
        <StoreContext.Provider value={store}>
            {children}
        </StoreContext.Provider>
    )
}