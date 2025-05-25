-- таблица plans с уникальным name
CREATE TABLE IF NOT EXISTS plans (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    storage_limit BIGINT NOT NULL,  -- 10 GiB = 10737418240
    price_cents INT NOT NULL,       -- в копейках
    period_days INT NOT NULL        -- длительность в днях
);

-- тарифы
INSERT INTO plans (name, storage_limit, price_cents, period_days)
VALUES
  ('free', 10737418240, 0, 0),           -- 10 * 1024^3
  ('pro', 1099511627776, 25000, 30)      -- 1 * 1024^4
ON CONFLICT (name) DO NOTHING;

CREATE TABLE IF NOT EXISTS user_plans (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    plan_id INT NOT NULL REFERENCES plans(id),
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    current_used BIGINT NOT NULL DEFAULT 0
);


-- 10 * 1024 * 1024 * 1024 = 10737418240 это 10 гб
-- 1 * 1024 * 1024 * 1024 * 1024 = 1099511627776 это 1тб