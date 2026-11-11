-- Replayed on every startup after the numbered schema files and account seeds.
-- Disable the selected A78 accounts without affecting namesakes elsewhere.
UPDATE accounts AS a
SET disabled = TRUE
FROM users AS u
JOIN departments AS d ON d.id = u.department
WHERE a.user_id = u.user_id
  AND a.disabled = FALSE
  AND d.title = 'Сервис А78'
  AND LOWER(TRIM(u.last_name)) IN (
    'корнилова',
    'заяц',
    'абушаев',
    'терентьев',
    'симонов',
    'носов',
    'шигаев',
    'комаров'
  );

-- Match the importer's transliterated lastname.firstname username format.
UPDATE accounts AS a
SET username = 'chizhikov.artem'
FROM users AS u
JOIN departments AS d ON d.id = u.department
WHERE a.user_id = u.user_id
  AND d.title = 'Сервис А78'
  AND LOWER(TRIM(u.last_name)) = 'чижиков'
  AND LOWER(TRIM(u.first_name)) IN ('артем', 'артём')
  AND a.username::TEXT IS DISTINCT FROM 'chizhikov.artem';

UPDATE accounts AS a
SET username = 'kozhadey.nikolay'
FROM users AS u
JOIN departments AS d ON d.id = u.department
WHERE a.user_id = u.user_id
  AND d.title = 'Сервис А78'
  AND LOWER(TRIM(u.last_name)) = 'кожадей'
  AND LOWER(TRIM(u.first_name)) = 'николай'
  AND a.username::TEXT IS DISTINCT FROM 'kozhadey.nikolay';

UPDATE accounts AS a
SET username = 'sheremeev.andrey'
FROM users AS u
JOIN departments AS d ON d.id = u.department
WHERE a.user_id = u.user_id
  AND d.title = 'Сервис А78'
  AND LOWER(TRIM(u.last_name)) = 'шеремеев'
  AND LOWER(TRIM(u.first_name)) = 'андрей'
  AND a.username::TEXT IS DISTINCT FROM 'sheremeev.andrey';
