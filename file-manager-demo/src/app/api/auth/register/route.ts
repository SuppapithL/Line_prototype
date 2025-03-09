import { NextResponse } from 'next/server';
import db from '@/lib/db';
import bcrypt from 'bcryptjs';
import { v4 as uuidv4 } from 'uuid';

export async function POST(request: Request) {
  try {
    // Expect only username and password from the client
    const { username, password } = await request.json();

    // Check if the username already exists
    const checkQuery = 'SELECT * FROM users WHERE username = $1';
    const checkValues = [username];
    const checkResult = await db.query(checkQuery, checkValues);

    if (checkResult.rows.length > 0) {
      return NextResponse.json({ error: 'Username already exists' }, { status: 409 });
    }

    // Generate a unique user id using uuidv4
    const userId = uuidv4();

    // Hash the password (in production, you must always hash passwords)
    const hashedPassword = await bcrypt.hash(password, 10);

    // Insert new user into the database
    const insertQuery =
      'INSERT INTO users (user_id, username, password) VALUES ($1, $2, $3) RETURNING user_id, username';
    const insertValues = [userId, username, hashedPassword];
    const insertResult = await db.query(insertQuery, insertValues);

    return NextResponse.json(
      { message: 'Registration successful', user: insertResult.rows[0] },
      { status: 201 }
    );
  } catch (error) {
    console.error('Registration error:', error);
    return NextResponse.json({ error: 'Internal Server Error' }, { status: 500 });
  }
}
