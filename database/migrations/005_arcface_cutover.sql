-- ============================================================================
-- Migration: ArcFace Cutover for Face Recognition
-- Description:
--   1) Tag vectors by embedding model to prevent mixed-model comparisons.
--   2) Mark pre-existing vectors as legacy (require re-enrollment for ArcFace-only matching).
-- ============================================================================

ALTER TABLE face_vectors
    ADD COLUMN IF NOT EXISTS model_name TEXT;

-- Existing vectors were produced by mixed/unknown models. Mark as legacy.
UPDATE face_vectors
SET model_name = 'legacy'
WHERE model_name IS NULL OR btrim(model_name) = '';

ALTER TABLE face_vectors
    ALTER COLUMN model_name SET NOT NULL;

ALTER TABLE face_vectors
    ALTER COLUMN model_name SET DEFAULT 'legacy';

CREATE INDEX IF NOT EXISTS idx_face_vectors_tenant_model
    ON face_vectors (tenant_id, model_name);

-- Helpful comment for ops.
COMMENT ON COLUMN face_vectors.model_name IS
    'Embedding model identifier (ArcFace, legacy, etc.). ArcFace-only matching uses model_name=''ArcFace''.';
