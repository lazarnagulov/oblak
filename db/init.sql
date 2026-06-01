CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS api_keys (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS functions (
    id UUID PRIMARY KEY,
    owner_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    runtime VARCHAR(50) NOT NULL,
    module_name VARCHAR(100) NOT NULL,
    handler_name VARCHAR(100) NOT NULL,
    artifact_hash VARCHAR(64) NOT NULL,
    timeout_seconds INT NOT NULL DEFAULT 5,
    memory_mb INT NOT NULL DEFAULT 128,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(owner_id, name)
);

CREATE TABLE IF NOT EXISTS executions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    function_id UUID NOT NULL REFERENCES functions(id) ON DELETE CASCADE,
    status VARCHAR(30) NOT NULL,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    execution_time_ms INT,
    logs TEXT,
    result_data TEXT,
    error_message TEXT,
    worker_node VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_ExecutionStatus CHECK (status IN (
        'PENDING',
        'RUNNING',
        'SUCCESS',
        'FAILED',
        'TIMEOUT'
    )) 
);

CREATE TABLE IF NOT EXISTS function_access_tokens (
    id SERIAL PRIMARY KEY,
    function_id UUID NOT NULL REFERENCES functions(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_functions_owner_id
    ON functions(owner_id);

CREATE INDEX IF NOT EXISTS idx_executions_status
    ON executions(status);

CREATE INDEX IF NOT EXISTS idx_executions_function_id
    ON executions(function_id);

CREATE INDEX IF NOT EXISTS idx_function_access_tokens_function_id
    ON function_access_tokens(function_id);

INSERT INTO users (username, password_hash) 
VALUES ('lazar', '$2a$10$wFRsfsXjMRaLaDN/wJ6AQuIIanU0v6Kk/QUpo0v5x.mWPy.KYTYgC')
ON CONFLICT (username) DO NOTHING;
