import { NextResponse } from 'next/server';
import db from '@/lib/db';
import bcrypt from 'bcryptjs';
import cookie from 'cookie';

export async function POST(request: Request) {
  try {
    const { userIdOrUsername, password } = await request.json();
    console.log('Login API received:', { userIdOrUsername, password });

    // Query user by user_id OR username
    const query = 'SELECT * FROM users WHERE user_id = $1 OR username = $2';
    const values = [userIdOrUsername, userIdOrUsername];
    const result = await db.query(query, values);
    console.log('Query result:', result.rows);

    if (result.rowCount === 0) {
      return NextResponse.json({ error: 'Invalid credentials' }, { status: 401 });
    }

    const user = result.rows[0];
    const isValidPassword = await bcrypt.compare(password, user.password);
    if (!isValidPassword) {
      return NextResponse.json({ error: 'Invalid credentials' }, { status: 401 });
    }

    console.log('Login successful for:', userIdOrUsername);

    // Use cookie.serialize to set an HttpOnly cookie (for security)
    const userCookie = cookie.serialize('user', JSON.stringify({
      user_id: user.user_id,
      username: user.username,
    }), {
      path: '/',
      httpOnly: true, // secure: cannot be read by client-side JavaScript
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'strict',
      maxAge: 60 * 60 * 24 * 7, // 7 days (in seconds)
    });

    const response = NextResponse.json({
      message: 'Login successful',
      user: { user_id: user.user_id, username: user.username },
    });
    response.headers.set('Set-Cookie', userCookie);
    return response;
  } catch (error) {
    console.error('Login API error:', error);
    return NextResponse.json({ error: 'Internal Server Error' }, { status: 500 });
  }
}
