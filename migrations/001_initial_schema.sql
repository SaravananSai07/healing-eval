-- Initial schema for healing-eval

CREATE TABLE IF NOT EXISTS conversations (
    id VARCHAR(64) PRIMARY KEY,
    agent_version VARCHAR(32) NOT NULL,
    turns JSONB NOT NULL,
    feedback JSONB,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS evaluations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id VARCHAR(64) NOT NULL,
    evaluator_type VARCHAR(32) NOT NULL,
    scores JSONB NOT NULL,
    issues JSONB DEFAULT '[]',
    confidence FLOAT,
    raw_output JSONB,
    latency_ms INT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS annotators (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(256),
    reliability_score FLOAT DEFAULT 1.0,
    total_annotations INT DEFAULT 0,
    agreement_rate FLOAT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS annotations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id VARCHAR(64) NOT NULL,
    turn_id INT,
    annotator_id VARCHAR(64) NOT NULL,
    annotation_type VARCHAR(32) NOT NULL,
    label VARCHAR(64) NOT NULL,
    confidence FLOAT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS improvement_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_id VARCHAR(64),
    suggestion_type VARCHAR(32) NOT NULL,
    target VARCHAR(256),
    suggestion TEXT NOT NULL,
    rationale TEXT,
    confidence FLOAT,
    status VARCHAR(32) DEFAULT 'pending',
    impact_measured JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS evaluator_accuracy (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    evaluator_type VARCHAR(32) NOT NULL,
    metric_date DATE NOT NULL,
    precision_score FLOAT,
    recall_score FLOAT,
    f1_score FLOAT,
    sample_count INT,
    human_correlation FLOAT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_conversations_created_at ON conversations(created_at);
CREATE INDEX IF NOT EXISTS idx_conversations_agent_version ON conversations(agent_version);
CREATE INDEX IF NOT EXISTS idx_evaluations_conversation_id ON evaluations(conversation_id);
CREATE INDEX IF NOT EXISTS idx_evaluations_evaluator_type ON evaluations(evaluator_type);
CREATE INDEX IF NOT EXISTS idx_evaluations_created_at ON evaluations(created_at);
CREATE INDEX IF NOT EXISTS idx_annotations_conversation_id ON annotations(conversation_id);
CREATE INDEX IF NOT EXISTS idx_annotations_annotator_id ON annotations(annotator_id);
CREATE INDEX IF NOT EXISTS idx_suggestions_status ON improvement_suggestions(status);
CREATE INDEX IF NOT EXISTS idx_evaluator_accuracy_type_date ON evaluator_accuracy(evaluator_type, metric_date);

