-- Merge legacy community tickets into the PixelAPI per-user support thread.
WITH ticket_bounds AS (
    SELECT
        user_id,
        MIN(created_at) AS created_at,
        MAX(updated_at) AS updated_at,
        MAX(updated_at) AS last_message_at
    FROM support_tickets
    GROUP BY user_id
),
latest_ticket AS (
    SELECT DISTINCT ON (user_id)
        user_id,
        subject,
        CASE status
            WHEN 'waiting_user' THEN 'pending_user'
            WHEN 'waiting_admin' THEN 'pending_admin'
            ELSE status
        END AS status,
        priority,
        assigned_admin_id
    FROM support_tickets
    ORDER BY user_id, updated_at DESC, id DESC
)
INSERT INTO support_threads (
    user_id,
    subject,
    status,
    priority,
    type,
    assigned_admin_id,
    last_message_at,
    created_at,
    updated_at
)
SELECT
    b.user_id,
    COALESCE(NULLIF(l.subject, ''), 'Support'),
    l.status,
    l.priority,
    'support',
    l.assigned_admin_id,
    b.last_message_at,
    b.created_at,
    b.updated_at
FROM ticket_bounds b
JOIN latest_ticket l ON l.user_id = b.user_id
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO support_messages (
    thread_id,
    sender_type,
    sender_id,
    message_type,
    content_format,
    title,
    content,
    source,
    source_id,
    metadata,
    created_at
)
SELECT
    st.id,
    stm.author_role,
    stm.author_user_id,
    'text',
    'plain',
    t.subject,
    stm.content,
    'legacy_community_ticket',
    stm.id::text,
    jsonb_build_object(
        'legacy_ticket_id', t.id,
        'legacy_message_id', stm.id,
        'legacy_category', t.category,
        'legacy_attachment_url', stm.attachment_url
    ),
    stm.created_at
FROM support_ticket_messages stm
JOIN support_tickets t ON t.id = stm.ticket_id
JOIN support_threads st ON st.user_id = t.user_id
ORDER BY stm.created_at, stm.id
ON CONFLICT (thread_id, source, source_id)
    WHERE source <> '' AND source_id <> ''
    DO NOTHING;

WITH last_messages AS (
    SELECT DISTINCT ON (thread_id)
        thread_id,
        id,
        sender_type,
        LEFT(REGEXP_REPLACE(TRIM(content), '\s+', ' ', 'g'), 240) AS excerpt,
        created_at
    FROM support_messages
    ORDER BY thread_id, created_at DESC, id DESC
),
legacy_unread AS (
    SELECT
        user_id,
        SUM(user_unread) AS user_unread,
        SUM(admin_unread) AS admin_unread
    FROM support_tickets
    GROUP BY user_id
),
read_markers AS (
    SELECT
        st.id AS thread_id,
        CASE WHEN lu.user_unread = 0 THEN MAX(sm.id) END AS user_last_read_message_id,
        CASE WHEN lu.user_unread = 0 THEN MAX(sm.created_at) END AS user_last_read_at,
        CASE WHEN lu.admin_unread = 0 THEN MAX(sm.id) END AS admin_last_read_message_id,
        CASE WHEN lu.admin_unread = 0 THEN MAX(sm.created_at) END AS admin_last_read_at
    FROM support_threads st
    JOIN legacy_unread lu ON lu.user_id = st.user_id
    LEFT JOIN support_messages sm ON sm.thread_id = st.id
    GROUP BY st.id, lu.user_unread, lu.admin_unread
)
UPDATE support_threads st
SET
    last_message_id = lm.id,
    last_message_sender_type = lm.sender_type,
    last_message_excerpt = lm.excerpt,
    last_message_at = lm.created_at,
    user_last_read_message_id = rm.user_last_read_message_id,
    user_last_read_at = rm.user_last_read_at,
    admin_last_read_message_id = rm.admin_last_read_message_id,
    admin_last_read_at = rm.admin_last_read_at,
    updated_at = GREATEST(st.updated_at, lm.created_at)
FROM last_messages lm
JOIN read_markers rm ON rm.thread_id = lm.thread_id
WHERE st.id = lm.thread_id;
