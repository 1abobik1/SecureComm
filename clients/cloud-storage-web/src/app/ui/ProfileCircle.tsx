'use client';


import React, { useState, useContext } from 'react';
import { Context } from '@/app/_app'; 
import { useRouter } from 'next/navigation';
import Image from 'next/image'

const ProfileCircle = () => {
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const { store } = useContext(Context);
  const router = useRouter();

  const handleMenuToggle = () => {
    setIsMenuOpen(!isMenuOpen);
  };

  const handleMenuClose = () => {
    setIsMenuOpen(false);
  };

  const handleLogout = async () => {
    await store.logout();
    router.push('/');
  };



  return (
    <div className="relative">
      <div
        onClick={handleMenuToggle}
        className="w-10 h-10 rounded-full overflow-hidden cursor-pointer border-2 border-gray-300"
      >
        <Image
          src="/Anonymous.svg.png"
          alt="Profile"
          width={40}
          height={40}
          className="w-full h-full object-cover"
        />
      </div>

      {isMenuOpen && (
        <div
          className="absolute right-0 mt-2 w-48 bg-white shadow-lg rounded-lg border border-gray-200"
          onClick={handleMenuClose}
        >
          <ul>



            <li
              className="px-4 py-2 text-gray-700 hover:bg-gray-100 cursor-pointer"
              onClick={handleLogout}
            >
              Logout
            </li>
          </ul>
        </div>
      )}
    </div>
  );
};

export default ProfileCircle;
