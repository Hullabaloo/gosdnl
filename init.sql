CREATE TABLE individuals (
  id SERIAL PRIMARY KEY,
  uid INTEGER NOT NULL,
  firstname TEXT NOT NULL,
  lastname TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);