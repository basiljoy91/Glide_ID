ALTER TABLE users
    ADD COLUMN IF NOT EXISTS manager_id UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS employment_type VARCHAR(50) NOT NULL DEFAULT 'full_time',
    ADD COLUMN IF NOT EXISTS work_location VARCHAR(120),
    ADD COLUMN IF NOT EXISTS cost_center VARCHAR(120),
    ADD COLUMN IF NOT EXISTS invite_status VARCHAR(30) NOT NULL DEFAULT 'not_invited',
    ADD COLUMN IF NOT EXISTS invite_sent_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS offboarded_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS offboarding_reason TEXT;

CREATE INDEX IF NOT EXISTS idx_users_manager_id ON users(manager_id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS employee_documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    document_type VARCHAR(50) NOT NULL,
    name VARCHAR(150) NOT NULL,
    file_url TEXT NOT NULL,
    expires_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}'::jsonb,
    uploaded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_employee_documents_user_id ON employee_documents(user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_employee_documents_tenant_id ON employee_documents(tenant_id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS leave_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    leave_type VARCHAR(50) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    day_count DECIMAL(6,2) NOT NULL DEFAULT 1,
    reason TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    review_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_leave_requests_tenant_status ON leave_requests(tenant_id, status, start_date DESC);
CREATE INDEX IF NOT EXISTS idx_leave_requests_user_id ON leave_requests(user_id, start_date DESC);

CREATE TABLE IF NOT EXISTS attendance_regularization_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    attendance_log_id UUID REFERENCES attendance_logs(id) ON DELETE SET NULL,
    request_date DATE NOT NULL,
    requested_status attendance_status NOT NULL,
    requested_punch_time TIMESTAMPTZ NOT NULL,
    reason TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    review_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_regularization_requests_tenant_status ON attendance_regularization_requests(tenant_id, status, request_date DESC);
CREATE INDEX IF NOT EXISTS idx_regularization_requests_user_id ON attendance_regularization_requests(user_id, request_date DESC);

CREATE TABLE IF NOT EXISTS overtime_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    work_date DATE NOT NULL,
    requested_minutes INTEGER NOT NULL,
    approved_minutes INTEGER NOT NULL DEFAULT 0,
    reason TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    review_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_overtime_requests_tenant_status ON overtime_requests(tenant_id, status, work_date DESC);
CREATE INDEX IF NOT EXISTS idx_overtime_requests_user_id ON overtime_requests(user_id, work_date DESC);

CREATE TABLE IF NOT EXISTS shift_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    shift_name VARCHAR(120) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    work_days TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    is_rota BOOLEAN NOT NULL DEFAULT false,
    notes TEXT,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_shift_assignments_tenant_user ON shift_assignments(tenant_id, user_id, start_date DESC) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS attendance_exception_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    attendance_log_id UUID NOT NULL REFERENCES attendance_logs(id) ON DELETE CASCADE,
    assigned_to UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    sla_due_at TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    note TEXT,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_exception_assignment_per_log UNIQUE (attendance_log_id)
);

CREATE INDEX IF NOT EXISTS idx_exception_assignments_tenant_status ON attendance_exception_assignments(tenant_id, status, sla_due_at);
CREATE INDEX IF NOT EXISTS idx_exception_assignments_assigned_to ON attendance_exception_assignments(assigned_to, status);

CREATE TABLE IF NOT EXISTS bulk_change_batches (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    change_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'previewed',
    summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    applied_at TIMESTAMPTZ,
    rolled_back_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS bulk_change_batch_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    batch_id UUID NOT NULL REFERENCES bulk_change_batches(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    before_state JSONB NOT NULL,
    after_state JSONB NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_bulk_change_batches_tenant_status ON bulk_change_batches(tenant_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_bulk_change_items_batch_id ON bulk_change_batch_items(batch_id);
