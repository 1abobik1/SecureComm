'use client'
import type {AppProps} from 'next/app'
import React, {createContext} from 'react'
import Store from "@/app/api/store/store"
import {observer} from 'mobx-react-lite'

interface State {
    store: Store
}

const store = new Store()
export const Context = createContext<State>({ store })

function MyApp({ Component, pageProps }: AppProps) {
    return (
        <Context.Provider value={{ store }}>
            <Component {...pageProps} />
        </Context.Provider>
    )
}

export default observer(MyApp)