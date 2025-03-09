'use client';

import { useRouter } from 'next/navigation';

export default function Tab3Page() {
  const router = useRouter();

  async function handleLogout() {
    try {
      await fetch('/api/auth/logout', {
        method: 'POST',
        credentials: 'include',
      });
      router.push('/login'); // Redirect to login after logout
    } catch (error) {
      console.error('Logout failed:', error);
    }
  }

  return (
    <div style={{ padding: '20px' }}>
      <h1>Tab 3</h1>
      <p>This is content for Tab 3.</p>

      <hr style={{ margin: '20px 0' }} />

      <p 
        onClick={handleLogout} 
        style={{
          textAlign: 'center',
          color: 'blue',
          textDecoration: 'underline',
          cursor: 'pointer',
          fontWeight: 'bold',
          fontSize: '16px',
          padding: '10px 0',
        }}
      >
        Logout
      </p>

      <hr style={{ margin: '20px 0' }} />
    </div>
  );
}
