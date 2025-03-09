import { NextResponse } from 'next/server';
import cookie from 'cookie';

export async function GET(request: Request) {
  try {
    // Get cookies from the request headers
    const cookies = request.headers.get('cookie');
    console.log('Cookies received in verify:', cookies);

    if (!cookies) {
      return NextResponse.json({ error: 'No cookies found' }, { status: 401 });
    }

    // Parse cookies
    const parsedCookies = cookie.parse(cookies);
    console.log('Parsed cookies:', parsedCookies);

    const userCookie = parsedCookies.user;
    if (!userCookie) {
      return NextResponse.json({ error: 'No user cookie found' }, { status: 401 });
    }

    // Decode the cookie (it is URL-encoded)
    const user = JSON.parse(decodeURIComponent(userCookie));
    console.log('Decoded user:', user);
    return NextResponse.json({ user }, { status: 200 });
  } catch (error) {
    console.error('Error verifying cookie:', error);
    return NextResponse.json({ error: 'Internal Server Error' }, { status: 500 });
  }
}
