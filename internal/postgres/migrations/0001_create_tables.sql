CREATE TABLE IF NOT EXISTS tasks (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title text NOT NULL CHECK (char_length(title) BETWEEN 1 AND 200),
    completed boolean NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS signups (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    full_name text NOT NULL,
    email text NOT NULL,
    plan text NOT NULL CHECK (plan IN ('Starter', 'Growth', 'Enterprise')),
    seats integer NOT NULL CHECK (seats >= 1),
    notes text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS inventory_items (
    name text PRIMARY KEY,
    quantity integer NOT NULL CHECK (quantity >= 0),
    status text NOT NULL CHECK (status IN ('In stock', 'Low', 'Out of stock'))
);
