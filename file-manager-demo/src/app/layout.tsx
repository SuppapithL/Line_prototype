'use client';

import BottomTabs from '@/components/BottomTabs';

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html>
      <head>
        <title>File Manager Demo</title>
      </head>
      <body style={{ paddingBottom: '60px' }}>
        {children}
        <BottomTabs />
      </body>
    </html>
  );
}
