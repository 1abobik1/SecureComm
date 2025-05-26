

import clsx from 'clsx';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import TypeFileIcon from './TypeFileIcon';

const links = [
  {name: 'Home',href: '/cloud/home',icontype: 'home' ,},
  {name: 'Videos',href: '/cloud/videos',icontype: 'video' ,},
  {name: 'Photos',href: '/cloud/photos',icontype: 'photo' ,},
  {name: 'Docs',href: '/cloud/docs',icontype: 'text' ,},
  {name: 'Others',href: '/cloud/unknown',icontype: 'unknown' ,},
];

export default function NavLinks() {
  const pathname = usePathname();


  return (
    <>
      {links.map((link) => {

        return (

          <Link
          key={link.name}
          href={link.href}
          className={clsx(
            'flex h-[36px] w-full items-center gap-2 rounded-md text-sm font-medium text-gray-500 hover:text-blue-500 md:p-2 md:px-3',
            {
              'bg-blue-300 text-blue-600': pathname === link.href,
            }
          )}
        >
         <div className="ml-2  flex items-center">
  <TypeFileIcon type={link.icontype} size={20} />
  <p className="text-left min-w-[80px] truncate ml-2">{link.name}</p>
</div>

        </Link>

        );
      })}
    </>
  );
}
