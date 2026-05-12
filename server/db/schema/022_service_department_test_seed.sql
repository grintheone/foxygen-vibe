DO $$
DECLARE
  service_vbb_department_title CONSTANT TEXT := 'Сервис ВББ';
  service_vbb_department_legacy_id CONSTANT UUID := '1f62a255-ef3a-11e5-8d88-001a64d22812';
  service_kpd_department_title CONSTANT TEXT := 'Сервис КПД';
  service_kpd_department_id_constant CONSTANT UUID := '5d32c97a-2a37-4f4a-90bb-b7e2f44d2ec1';

  old_markin_first_name CONSTANT TEXT := 'Сергей';
  old_markin_last_name CONSTANT TEXT := 'Маркин';
  old_markin_legacy_user_id CONSTANT UUID := '997f472f-d2b6-11e8-9619-001a64d22812';
  old_markin_disabled_username CONSTANT TEXT := 'markin.sergey_vbb';

  new_markin_username CONSTANT TEXT := 'markin.sergey';
  new_markin_user_id_constant CONSTANT UUID := 'd6a06212-cd14-4d90-8b81-d64d7217d1f3';

  bogoduhov_first_name CONSTANT TEXT := 'Михаил';
  bogoduhov_last_name CONSTANT TEXT := 'Богодухов';
  bogoduhov_username CONSTANT TEXT := 'bogoduhov.mikhail';
  bogoduhov_user_id_constant CONSTANT UUID := '68a93f39-6d04-4c2c-8d0f-d8a9b37e7a83';

  gusev_first_name CONSTANT TEXT := 'Артем';
  gusev_last_name CONSTANT TEXT := 'Гусев';
  gusev_username CONSTANT TEXT := 'gusev.artem';
  gusev_legacy_user_id CONSTANT UUID := 'a9cb1f49-d74a-11ec-80f2-40b0765b1e01';

  trifonova_first_name CONSTANT TEXT := 'Наталья';
  trifonova_last_name CONSTANT TEXT := 'Трифонова';
  trifonova_username CONSTANT TEXT := 'trifonova.natalya';
  trifonova_user_id_constant CONSTANT UUID := '4dce8bf3-052a-4a3a-985b-c237d44bd7b7';

  beloshnikova_first_name CONSTANT TEXT := 'Оксана';
  beloshnikova_last_name CONSTANT TEXT := 'Белошникова';
  beloshnikova_username CONSTANT TEXT := 'beloshnikova.oksana';
  beloshnikova_user_id_constant CONSTANT UUID := 'd9cf9f2d-42a9-4d53-985d-6e635a0b1f68';

  configured_default_password TEXT := NULLIF(current_setting('foxygen.import_default_password', true), '');
  seed_password_hash TEXT;
  service_vbb_department_id UUID;
  service_kpd_department_id UUID;
  coordinator_role_id INT;
  regular_role_id INT;
  old_markin_user_id UUID;
  old_markin_email TEXT := '';
  old_markin_phone TEXT := '';
  old_markin_logo TEXT := '';
  new_markin_user_id UUID;
  bogoduhov_user_id UUID;
  gusev_user_id UUID;
  trifonova_user_id UUID;
  beloshnikova_user_id UUID;
  skip_test_seed BOOLEAN := COALESCE(current_setting('foxygen.skip_test_data_seed', true), 'false')::BOOLEAN;
BEGIN
  IF skip_test_seed THEN
    RETURN;
  END IF;

  SELECT id
  INTO service_vbb_department_id
  FROM departments
  WHERE title = service_vbb_department_title
  ORDER BY CASE WHEN id = service_vbb_department_legacy_id THEN 0 ELSE 1 END, id
  LIMIT 1;

  IF service_vbb_department_id IS NULL THEN
    INSERT INTO departments (id, title)
    VALUES (service_vbb_department_legacy_id, service_vbb_department_title)
    ON CONFLICT (id) DO UPDATE
    SET title = EXCLUDED.title;

    service_vbb_department_id := service_vbb_department_legacy_id;
  END IF;

  SELECT id
  INTO service_kpd_department_id
  FROM departments
  WHERE title = service_kpd_department_title
  ORDER BY CASE WHEN id = service_kpd_department_id_constant THEN 0 ELSE 1 END, id
  LIMIT 1;

  IF service_kpd_department_id IS NULL THEN
    INSERT INTO departments (id, title)
    VALUES (service_kpd_department_id_constant, service_kpd_department_title)
    ON CONFLICT (id) DO UPDATE
    SET title = EXCLUDED.title;

    service_kpd_department_id := service_kpd_department_id_constant;
  END IF;

  INSERT INTO roles (id, name, description)
  SELECT 2, 'coordinator', ''
  WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'coordinator')
    AND NOT EXISTS (SELECT 1 FROM roles WHERE id = 2);

  INSERT INTO roles (id, name, description)
  SELECT COALESCE((SELECT MAX(id) FROM roles), 0) + 1, 'coordinator', ''
  WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'coordinator');

  INSERT INTO roles (id, name, description)
  SELECT 3, 'user', ''
  WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'user')
    AND NOT EXISTS (SELECT 1 FROM roles WHERE id = 3);

  INSERT INTO roles (id, name, description)
  SELECT COALESCE((SELECT MAX(id) FROM roles), 0) + 1, 'user', ''
  WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'user');

  SELECT id
  INTO coordinator_role_id
  FROM roles
  WHERE name = 'coordinator'
  ORDER BY id
  LIMIT 1;

  SELECT id
  INTO regular_role_id
  FROM roles
  WHERE name = 'user'
  ORDER BY id
  LIMIT 1;

  IF configured_default_password IS NOT NULL THEN
    seed_password_hash := crypt(configured_default_password, gen_salt('bf'));
  ELSE
    SELECT password_hash
    INTO seed_password_hash
    FROM accounts
    GROUP BY password_hash
    HAVING COUNT(*) > 1
    ORDER BY COUNT(*) DESC, password_hash
    LIMIT 1;
  END IF;

  SELECT
    a.user_id,
    COALESCE(u.email, ''),
    COALESCE(u.phone, ''),
    COALESCE(u.logo, '')
  INTO
    old_markin_user_id,
    old_markin_email,
    old_markin_phone,
    old_markin_logo
  FROM accounts AS a
  JOIN users AS u ON u.user_id = a.user_id
  WHERE a.user_id = old_markin_legacy_user_id
     OR (
       u.first_name = old_markin_first_name
       AND u.last_name = old_markin_last_name
       AND u.department = service_vbb_department_id
       AND a.username::text IN (new_markin_username, old_markin_disabled_username)
     )
  ORDER BY
    CASE
      WHEN a.user_id = old_markin_legacy_user_id THEN 0
      WHEN a.username::text = old_markin_disabled_username THEN 1
      ELSE 2
    END,
    a.user_id
  LIMIT 1;

  IF old_markin_user_id IS NOT NULL THEN
    IF seed_password_hash IS NOT NULL THEN
      UPDATE accounts
      SET username = old_markin_disabled_username,
          disabled = TRUE,
          password_hash = seed_password_hash
      WHERE user_id = old_markin_user_id;
    ELSE
      UPDATE accounts
      SET username = old_markin_disabled_username,
          disabled = TRUE
      WHERE user_id = old_markin_user_id;
    END IF;

    UPDATE users
    SET first_name = old_markin_first_name,
        last_name = old_markin_last_name,
        department = service_vbb_department_id
    WHERE user_id = old_markin_user_id;
  END IF;

  SELECT user_id
  INTO new_markin_user_id
  FROM accounts
  WHERE user_id = new_markin_user_id_constant
     OR username = new_markin_username
  ORDER BY CASE WHEN user_id = new_markin_user_id_constant THEN 0 ELSE 1 END, user_id
  LIMIT 1;

  IF new_markin_user_id IS NULL THEN
    new_markin_user_id := new_markin_user_id_constant;
  END IF;

  IF EXISTS (SELECT 1 FROM accounts WHERE user_id = new_markin_user_id) THEN
    IF seed_password_hash IS NOT NULL THEN
      UPDATE accounts
      SET username = new_markin_username,
          disabled = FALSE,
          password_hash = seed_password_hash
      WHERE user_id = new_markin_user_id;
    ELSE
      UPDATE accounts
      SET username = new_markin_username,
          disabled = FALSE
      WHERE user_id = new_markin_user_id;
    END IF;
  ELSIF seed_password_hash IS NOT NULL THEN
    INSERT INTO accounts (user_id, username, disabled, password_hash)
    VALUES (new_markin_user_id, new_markin_username, FALSE, seed_password_hash);
  END IF;

  IF EXISTS (SELECT 1 FROM accounts WHERE user_id = new_markin_user_id) THEN
    INSERT INTO users (user_id, first_name, last_name, department, email, phone, logo)
    VALUES (
      new_markin_user_id,
      old_markin_first_name,
      old_markin_last_name,
      service_kpd_department_id,
      old_markin_email,
      old_markin_phone,
      old_markin_logo
    )
    ON CONFLICT (user_id) DO UPDATE
    SET first_name = EXCLUDED.first_name,
        last_name = EXCLUDED.last_name,
        department = EXCLUDED.department,
        email = EXCLUDED.email,
        phone = EXCLUDED.phone,
        logo = EXCLUDED.logo;

    INSERT INTO account_roles (user_id, role_id)
    VALUES (new_markin_user_id, coordinator_role_id)
    ON CONFLICT (user_id) DO UPDATE
    SET role_id = EXCLUDED.role_id;
  END IF;

  SELECT a.user_id
  INTO bogoduhov_user_id
  FROM accounts AS a
  JOIN users AS u ON u.user_id = a.user_id
  WHERE a.user_id = bogoduhov_user_id_constant
     OR (
       a.username::text = bogoduhov_username
       AND u.first_name = bogoduhov_first_name
       AND u.last_name = bogoduhov_last_name
       AND u.department = service_kpd_department_id
     )
  ORDER BY CASE WHEN a.user_id = bogoduhov_user_id_constant THEN 0 ELSE 1 END, a.user_id
  LIMIT 1;

  IF bogoduhov_user_id IS NOT NULL THEN
    DELETE FROM accounts
    WHERE user_id = bogoduhov_user_id;
  END IF;

  SELECT a.user_id
  INTO gusev_user_id
  FROM accounts AS a
  JOIN users AS u ON u.user_id = a.user_id
  WHERE a.user_id = gusev_legacy_user_id
     OR (
       u.first_name = gusev_first_name
       AND u.last_name = gusev_last_name
       AND u.department = service_vbb_department_id
       AND a.username::text = gusev_username
     )
  ORDER BY CASE WHEN a.user_id = gusev_legacy_user_id THEN 0 ELSE 1 END, a.user_id
  LIMIT 1;

  IF gusev_user_id IS NOT NULL THEN
    UPDATE users
    SET first_name = gusev_first_name,
        last_name = gusev_last_name,
        department = service_vbb_department_id
    WHERE user_id = gusev_user_id;

    INSERT INTO account_roles (user_id, role_id)
    VALUES (gusev_user_id, coordinator_role_id)
    ON CONFLICT (user_id) DO UPDATE
    SET role_id = EXCLUDED.role_id;
  END IF;

  SELECT user_id
  INTO trifonova_user_id
  FROM accounts
  WHERE user_id = trifonova_user_id_constant
     OR username = trifonova_username
  ORDER BY CASE WHEN user_id = trifonova_user_id_constant THEN 0 ELSE 1 END, user_id
  LIMIT 1;

  IF trifonova_user_id IS NULL THEN
    trifonova_user_id := trifonova_user_id_constant;
  END IF;

  IF EXISTS (SELECT 1 FROM accounts WHERE user_id = trifonova_user_id) THEN
    IF seed_password_hash IS NOT NULL THEN
      UPDATE accounts
      SET username = trifonova_username,
          disabled = FALSE,
          password_hash = seed_password_hash
      WHERE user_id = trifonova_user_id;
    ELSE
      UPDATE accounts
      SET username = trifonova_username,
          disabled = FALSE
      WHERE user_id = trifonova_user_id;
    END IF;
  ELSIF seed_password_hash IS NOT NULL THEN
    INSERT INTO accounts (user_id, username, disabled, password_hash)
    VALUES (trifonova_user_id, trifonova_username, FALSE, seed_password_hash);
  END IF;

  IF EXISTS (SELECT 1 FROM accounts WHERE user_id = trifonova_user_id) THEN
    INSERT INTO users (user_id, first_name, last_name, department, email, phone, logo)
    VALUES (
      trifonova_user_id,
      trifonova_first_name,
      trifonova_last_name,
      service_vbb_department_id,
      '',
      '',
      ''
    )
    ON CONFLICT (user_id) DO UPDATE
    SET first_name = EXCLUDED.first_name,
        last_name = EXCLUDED.last_name,
        department = EXCLUDED.department,
        email = EXCLUDED.email,
        phone = EXCLUDED.phone,
        logo = EXCLUDED.logo;

    INSERT INTO account_roles (user_id, role_id)
    VALUES (trifonova_user_id, coordinator_role_id)
    ON CONFLICT (user_id) DO UPDATE
    SET role_id = EXCLUDED.role_id;
  END IF;

  SELECT user_id
  INTO beloshnikova_user_id
  FROM accounts
  WHERE user_id = beloshnikova_user_id_constant
     OR username = beloshnikova_username
  ORDER BY CASE WHEN user_id = beloshnikova_user_id_constant THEN 0 ELSE 1 END, user_id
  LIMIT 1;

  IF beloshnikova_user_id IS NULL THEN
    beloshnikova_user_id := beloshnikova_user_id_constant;
  END IF;

  IF EXISTS (SELECT 1 FROM accounts WHERE user_id = beloshnikova_user_id) THEN
    IF seed_password_hash IS NOT NULL THEN
      UPDATE accounts
      SET username = beloshnikova_username,
          disabled = FALSE,
          password_hash = seed_password_hash
      WHERE user_id = beloshnikova_user_id;
    ELSE
      UPDATE accounts
      SET username = beloshnikova_username,
          disabled = FALSE
      WHERE user_id = beloshnikova_user_id;
    END IF;
  ELSIF seed_password_hash IS NOT NULL THEN
    INSERT INTO accounts (user_id, username, disabled, password_hash)
    VALUES (beloshnikova_user_id, beloshnikova_username, FALSE, seed_password_hash);
  END IF;

  IF EXISTS (SELECT 1 FROM accounts WHERE user_id = beloshnikova_user_id) THEN
    INSERT INTO users (user_id, first_name, last_name, department, email, phone, logo)
    VALUES (
      beloshnikova_user_id,
      beloshnikova_first_name,
      beloshnikova_last_name,
      service_vbb_department_id,
      '',
      '',
      ''
    )
    ON CONFLICT (user_id) DO UPDATE
    SET first_name = EXCLUDED.first_name,
        last_name = EXCLUDED.last_name,
        department = EXCLUDED.department,
        email = EXCLUDED.email,
        phone = EXCLUDED.phone,
        logo = EXCLUDED.logo;

    INSERT INTO account_roles (user_id, role_id)
    VALUES (beloshnikova_user_id, coordinator_role_id)
    ON CONFLICT (user_id) DO UPDATE
    SET role_id = EXCLUDED.role_id;
  END IF;
END $$;
