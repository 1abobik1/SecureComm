'use client';
import React from 'react';
import {usePathname} from 'next/navigation';
import ProfileCircle from './ProfileCircle';
import FileUploader from '../components/FileUploader';
import TypeFileIcon from './TypeFileIcon';

const links = [
    {name: 'Home',href: '/cloud/home',icontype: 'home' ,},
    {name: 'Videos',href: '/cloud/videos',icontype: 'video' ,},
    {name: 'Photos',href: '/cloud/photos',icontype: 'photo' ,},
    {name: 'Docs',href: '/cloud/docs',icontype: 'text' ,},
    {name: 'Others',href: '/cloud/unknown',icontype: 'unknown' ,},

  ];
const Header = () => {
    const pathname = usePathname();
    const activeLink = links.find((link) => pathname.startsWith(link.href));
    if (!activeLink) return null;


    return (
        <div className='flex flex-row justify-between px-5 '>
            <div className="flex flex-row  p-4 my-4 ">
                <TypeFileIcon type={activeLink.icontype} size={28}/>

            </div>
            <div className='flex flex-row py-6 mx-5'>
            <FileUploader/>
            <div className='mx-5'><ProfileCircle /></div>
            </div>
        </div>
    );
};

export default Header;
