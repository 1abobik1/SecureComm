'use client'
import type {AppProps} from 'next/app'
import React from 'react'
import Store from "@/app/api/store/store"
import {observer} from 'mobx-react-lite'
import {Context} from '@/app/api/store/context';

function MyApp({ Component, pageProps }: AppProps) {
    const store = new Store();

    return (
        <Context.Provider value={ store }>
            <Component {...pageProps} />
        </Context.Provider>
    )
}

export default observer(MyApp);