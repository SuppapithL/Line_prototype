'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import BottomTabs from '@/components/BottomTabs'; // Import the BottomTabs component

export default function HomePage() {
  const [user, setUser] = useState<{ username: string; user_id: string } | null>(null);
  const [loading, setLoading] = useState(true);
  const router = useRouter();

  useEffect(() => {
    async function fetchUser() {
      try {
        const response = await fetch('/api/auth/verify', {
          method: 'GET',
          credentials: 'include', // Ensure cookies are sent
        });
        console.log('Response status:', response.status);
        const data = await response.json();
        console.log('Data from verify endpoint:', data);
        if (data.error) {
          router.push('/login');
        } else {
          setUser(data.user);
        }
      } catch (error) {
        console.error('Error fetching user:', error);
        router.push('/login');
      } finally {
        setLoading(false);
      }
    }
    fetchUser();
  }, [router]);

  if (loading) return <p>Loading...</p>;

  return (
    <div style={{ paddingBottom: '60px' }}> {/* Make room for BottomTabs */}
      <h1>Main File Manager</h1>
      {user ? (
        <>
          <p>Welcome, <strong>{user.username}</strong>!</p>
          <p>Your User ID: {user.user_id}</p>
        </>
      ) : (
        <p>No user information found.</p>
      )}

      {/* Bottom Tabs - will always be displayed at the bottom */}
      <BottomTabs />
    </div>
  );
}
