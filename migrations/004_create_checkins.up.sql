CREATE TABLE check_ins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL DEFAULT '',
    media JSONB NOT NULL DEFAULT '[]',
    checked_date DATE NOT NULL DEFAULT CURRENT_DATE,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(plan_id, user_id, checked_date)
);

CREATE INDEX idx_checkins_plan_user ON check_ins(plan_id, user_id, checked_date DESC);
CREATE INDEX idx_checkins_user_date ON check_ins(user_id, checked_date DESC);
