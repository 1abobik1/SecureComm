'use client';

import React, {useContext, useEffect} from 'react';
import {usePathname, useRouter} from 'next/navigation';
import {Context} from '@/app/_app';
import SideBar from '@/app/ui/SideBar';
import Header from '../ui/Header';
import {observer} from 'mobx-react-lite';

function Layout({ children }: { children: React.ReactNode }) {
    const { store } = useContext(Context);
    const router = useRouter();
    const pathname = usePathname();

    useEffect(() => {
        // Сохраняем текущий путь в localStorage при его изменении
        if (pathname) {
            localStorage.setItem('lastPath', pathname);
        }
    }, [pathname]);

    useEffect(() => {
        if (!store.isLoading && !store.isAuth) {
            router.push('/');
        }
    }, [store.isAuth, store.isLoading]);

    if (store.isLoading) {
        return <div className="p-12">Загрузка...</div>;
    }

    if (!store.isAuth) {
        return null;
    }

    return (
        <div className="flex h-screen flex-col md:flex-row md:overflow-hidden">
            <div className="w-full flex-none md:w-64 bg-gray-100">
                <SideBar />
            </div>

            <div className="flex flex-col flex-grow w-100%">
                <Header />
                <div className="flex-grow p-6 md:overflow-y-auto md:p-12">{children}</div>
            </div>
        </div>
    );
}

export default observer(Layout);