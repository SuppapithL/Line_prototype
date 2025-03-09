import { Pool } from 'pg';

const pool = new Pool({
  connectionString: "postgres://postgres:Suppapith2@localhost:5432/file_manager?sslmode=disable",
});

export default pool;
