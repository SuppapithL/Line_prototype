'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';

export default function BottomTabs() {
  const pathname = usePathname();

  const tabs = [
    { label: 'Tab 1', href: '/tab1' },
    { label: 'Tab 2', href: '/tab2' },
    { label: 'Tab 3', href: '/tab3' },
  ];

  return (
    <nav
      style={{
        position: 'fixed',
        bottom: 0,
        left: 0,
        right: 0,
        height: '60px',
        backgroundColor: '#fff',
        borderTop: '1px solid #ccc',
        display: 'flex',
        justifyContent: 'space-around',
        alignItems: 'center',
        zIndex: 1000,
      }}
    >
      {tabs.map((tab) => (
        <Link key={tab.href} href={tab.href} style={{ textDecoration: 'none' }}>
          <span
            style={{
              color: pathname === tab.href ? 'blue' : 'black',
              fontWeight: pathname === tab.href ? 'bold' : 'normal',
            }}
          >
            {tab.label}
          </span>
        </Link>
      ))}
    </nav>
  );
}
