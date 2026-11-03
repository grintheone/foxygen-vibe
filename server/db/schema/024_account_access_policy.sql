-- Applied after account seeds on every startup, including after a rebuild.
-- Preserve existing disabled flags for exempt accounts; never re-enable them.
UPDATE accounts AS a
SET disabled = TRUE
WHERE a.disabled = FALSE
  AND a.username NOT IN ('ivanova.anna.01', 'petrov.ivan.02')
  AND NOT EXISTS (
    SELECT 1
    FROM users AS u
    JOIN departments AS d ON d.id = u.department
    WHERE u.user_id = a.user_id
      AND d.title IN ('Сервис ВББ', 'Сервис А78', 'Сервис КПД')
  );
