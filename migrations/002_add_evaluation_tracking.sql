-- Add token tracking, status fields, and comprehensive evaluation tracking

ALTER TABLE evaluations ADD COLUMN IF NOT EXISTS model_name VARCHAR(64);
ALTER TABLE evaluations ADD COLUMN IF NOT EXISTS prompt_tokens INT DEFAULT 0;
ALTER TABLE evaluations ADD COLUMN IF NOT EXISTS completion_tokens INT DEFAULT 0;
ALTER TABLE evaluations ADD COLUMN IF NOT EXISTS total_tokens INT DEFAULT 0;
ALTER TABLE evaluations ADD COLUMN IF NOT EXISTS estimated_cost_usd DECIMAL(10,6) DEFAULT 0;
ALTER TABLE evaluations ADD COLUMN IF NOT EXISTS status VARCHAR(32) DEFAULT 'success';
ALTER TABLE evaluations ADD COLUMN IF NOT EXISTS error_message TEXT;
ALTER TABLE evaluations ADD COLUMN IF NOT EXISTS retry_count INT DEFAULT 0;
ALTER TABLE evaluations ADD COLUMN IF NOT EXISTS token_budget_exceeded BOOLEAN DEFAULT false;

CREATE TABLE IF NOT EXISTS human_review_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id VARCHAR(64) NOT NULL,
    evaluation_id UUID,
    reason VARCHAR(256) NOT NULL,
    priority INT DEFAULT 2,
    status VARCHAR(32) DEFAULT 'pending',
    assigned_to VARCHAR(64),
    routing_confidence FLOAT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,
    reviewer_notes TEXT
);

CREATE INDEX IF NOT EXISTS idx_review_queue_status ON human_review_queue(status, priority);
CREATE INDEX IF NOT EXISTS idx_review_queue_conversation ON human_review_queue(conversation_id);

CREATE INDEX IF NOT EXISTS idx_evaluations_status ON evaluations(status);

ALTER TABLE conversations ADD COLUMN IF NOT EXISTS evaluation_status VARCHAR(32);
CREATE INDEX IF NOT EXISTS idx_conversations_eval_status ON conversations(evaluation_status);

