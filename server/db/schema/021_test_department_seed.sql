DO $$
DECLARE
  legacy_department_title CONSTANT TEXT := 'Test Department';
  test_department_title CONSTANT TEXT := 'Тестовый отдел';
  coordinator_username CONSTANT TEXT := 'ivanova.anna.01';
  regular_username CONSTANT TEXT := 'petrov.ivan.02';
  coordinator_user_id CONSTANT UUID := 'f4307c83-df42-4c4f-9367-39cf2f761b19';
  regular_user_id CONSTANT UUID := '56b02761-a0f3-4871-a2fd-8474fdb463df';
  created_ticket_id CONSTANT UUID := 'ac7cdd11-46c8-4474-8522-5dd140ab77fe';
  inwork_ticket_id CONSTANT UUID := 'ee1f9b03-8b1f-44cb-bce8-c7f2c46f1a91';
  closed_ticket_id CONSTANT UUID := '8c41d793-670e-493a-8ddd-a0eb0e904d74';
  configured_default_password TEXT := NULLIF(current_setting('foxygen.import_default_password', true), '');
  seed_password_hash TEXT;
  test_department_id UUID;
  legacy_department_id UUID;
  coordinator_role_id INT;
  regular_role_id INT;
  ticket_client_id UUID;
  ticket_device_id UUID;
  ticket_contact_id UUID;
  ticket_type_id VARCHAR(128);
  ticket_reason_id VARCHAR(128);
  created_status_id VARCHAR(128);
  inwork_status_id VARCHAR(128);
  closed_status_id VARCHAR(128);
BEGIN
  SELECT id
  INTO legacy_department_id
  FROM departments
  WHERE title = legacy_department_title
  ORDER BY id
  LIMIT 1;

  SELECT id
  INTO test_department_id
  FROM departments
  WHERE title = test_department_title
  ORDER BY id
  LIMIT 1;

  IF legacy_department_id IS NOT NULL AND test_department_id IS NULL THEN
    UPDATE departments
    SET title = test_department_title
    WHERE id = legacy_department_id
    RETURNING id INTO test_department_id;
  ELSIF legacy_department_id IS NOT NULL AND test_department_id IS NOT NULL AND legacy_department_id <> test_department_id THEN
    UPDATE users
    SET department = test_department_id
    WHERE department = legacy_department_id;

    UPDATE tickets
    SET department = test_department_id
    WHERE department = legacy_department_id;

    DELETE FROM departments
    WHERE id = legacy_department_id;
  END IF;

  IF test_department_id IS NULL THEN
    INSERT INTO departments (title)
    VALUES (test_department_title)
    ON CONFLICT (title) DO UPDATE
    SET title = EXCLUDED.title
    RETURNING id INTO test_department_id;
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

  IF seed_password_hash IS NOT NULL THEN
    INSERT INTO accounts (user_id, username, disabled, password_hash)
    VALUES (coordinator_user_id, coordinator_username, FALSE, seed_password_hash)
    ON CONFLICT (user_id) DO UPDATE
    SET username = EXCLUDED.username,
        disabled = EXCLUDED.disabled,
        password_hash = EXCLUDED.password_hash;

    INSERT INTO accounts (user_id, username, disabled, password_hash)
    VALUES (regular_user_id, regular_username, FALSE, seed_password_hash)
    ON CONFLICT (user_id) DO UPDATE
    SET username = EXCLUDED.username,
        disabled = EXCLUDED.disabled,
        password_hash = EXCLUDED.password_hash;
  END IF;

  IF EXISTS (SELECT 1 FROM accounts WHERE user_id = coordinator_user_id) THEN
    INSERT INTO users (user_id, first_name, last_name, department, email, phone, logo, latest_ticket)
    VALUES (
      coordinator_user_id,
      'Анна',
      'Иванова',
      test_department_id,
      'anna.ivanova@foxygen.local',
      '+79000000001',
      '',
      created_ticket_id
    )
    ON CONFLICT (user_id) DO UPDATE
    SET first_name = EXCLUDED.first_name,
        last_name = EXCLUDED.last_name,
        department = EXCLUDED.department,
        email = EXCLUDED.email,
        phone = EXCLUDED.phone,
        logo = EXCLUDED.logo,
        latest_ticket = EXCLUDED.latest_ticket;

    INSERT INTO account_roles (user_id, role_id)
    VALUES (coordinator_user_id, coordinator_role_id)
    ON CONFLICT (user_id) DO UPDATE
    SET role_id = EXCLUDED.role_id;
  END IF;

  IF EXISTS (SELECT 1 FROM accounts WHERE user_id = regular_user_id) THEN
    INSERT INTO users (user_id, first_name, last_name, department, email, phone, logo, latest_ticket)
    VALUES (
      regular_user_id,
      'Иван',
      'Петров',
      test_department_id,
      'ivan.petrov@foxygen.local',
      '+79000000002',
      '',
      created_ticket_id
    )
    ON CONFLICT (user_id) DO UPDATE
    SET first_name = EXCLUDED.first_name,
        last_name = EXCLUDED.last_name,
        department = EXCLUDED.department,
        email = EXCLUDED.email,
        phone = EXCLUDED.phone,
        logo = EXCLUDED.logo,
        latest_ticket = EXCLUDED.latest_ticket;

    INSERT INTO account_roles (user_id, role_id)
    VALUES (regular_user_id, regular_role_id)
    ON CONFLICT (user_id) DO UPDATE
    SET role_id = EXCLUDED.role_id;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM accounts WHERE user_id = coordinator_user_id)
     OR NOT EXISTS (SELECT 1 FROM accounts WHERE user_id = regular_user_id) THEN
    RETURN;
  END IF;

  SELECT linked.client_id, linked.device_id, linked.contact_id
  INTO ticket_client_id, ticket_device_id, ticket_contact_id
  FROM (
    SELECT
      a.actual_client AS client_id,
      a.device AS device_id,
      (
        SELECT c.id
        FROM contacts AS c
        WHERE c.client_id = a.actual_client
        ORDER BY c.name, c.id
        LIMIT 1
      ) AS contact_id
    FROM agreements AS a
    WHERE a.actual_client IS NOT NULL
      AND a.device IS NOT NULL
    ORDER BY a.assigned_at DESC NULLS LAST, a.id
    LIMIT 1
  ) AS linked;

  IF ticket_client_id IS NULL THEN
    SELECT id INTO ticket_client_id FROM clients ORDER BY title, id LIMIT 1;
  END IF;

  IF ticket_device_id IS NULL THEN
    SELECT id INTO ticket_device_id FROM devices ORDER BY serial_number, id LIMIT 1;
  END IF;

  IF ticket_contact_id IS NULL AND ticket_client_id IS NOT NULL THEN
    SELECT id
    INTO ticket_contact_id
    FROM contacts
    WHERE client_id = ticket_client_id
    ORDER BY name, id
    LIMIT 1;
  END IF;

  SELECT type INTO ticket_type_id FROM ticket_types WHERE type = 'internal' LIMIT 1;
  IF ticket_type_id IS NULL THEN
    SELECT type INTO ticket_type_id FROM ticket_types ORDER BY type LIMIT 1;
  END IF;

  SELECT id INTO ticket_reason_id FROM ticket_reasons WHERE id = 'maintenance' LIMIT 1;
  IF ticket_reason_id IS NULL THEN
    SELECT id INTO ticket_reason_id FROM ticket_reasons WHERE id = 'maintanence' LIMIT 1;
  END IF;
  IF ticket_reason_id IS NULL THEN
    SELECT id INTO ticket_reason_id FROM ticket_reasons ORDER BY id LIMIT 1;
  END IF;

  SELECT type INTO created_status_id FROM ticket_statuses WHERE type = 'created' LIMIT 1;
  SELECT type INTO inwork_status_id FROM ticket_statuses WHERE type = 'inWork' LIMIT 1;
  SELECT type INTO closed_status_id FROM ticket_statuses WHERE type = 'closed' LIMIT 1;

  INSERT INTO tickets (
    id,
    created_at,
    urgent,
    client,
    device,
    ticket_type,
    author,
    department,
    reason,
    description,
    contact_person,
    status,
    result,
    double_signed
  )
  VALUES (
    created_ticket_id,
    TIMESTAMP '2026-04-01 09:00:00',
    TRUE,
    ticket_client_id,
    ticket_device_id,
    ticket_type_id,
    regular_user_id,
    test_department_id,
    ticket_reason_id,
    'Тестовая заявка: срочное обращение, созданное тестовым пользователем для распределения координатором.',
    ticket_contact_id,
    created_status_id,
    '',
    FALSE
  )
  ON CONFLICT (id) DO UPDATE
  SET created_at = EXCLUDED.created_at,
      urgent = EXCLUDED.urgent,
      client = EXCLUDED.client,
      device = EXCLUDED.device,
      ticket_type = EXCLUDED.ticket_type,
      author = EXCLUDED.author,
      department = EXCLUDED.department,
      reason = EXCLUDED.reason,
      description = EXCLUDED.description,
      contact_person = EXCLUDED.contact_person,
      status = EXCLUDED.status,
      result = EXCLUDED.result,
      double_signed = EXCLUDED.double_signed;

  INSERT INTO tickets (
    id,
    created_at,
    assigned_at,
    workstarted_at,
    planned_start,
    planned_end,
    assigned_start,
    assigned_end,
    urgent,
    client,
    device,
    ticket_type,
    author,
    department,
    assigned_by,
    reason,
    description,
    contact_person,
    executor,
    status,
    result,
    double_signed
  )
  VALUES (
    inwork_ticket_id,
    TIMESTAMP '2026-04-02 08:30:00',
    TIMESTAMP '2026-04-02 09:00:00',
    TIMESTAMP '2026-04-02 09:20:00',
    TIMESTAMP '2026-04-02 09:00:00',
    TIMESTAMP '2026-04-02 14:00:00',
    TIMESTAMP '2026-04-02 09:00:00',
    TIMESTAMP '2026-04-02 14:00:00',
    FALSE,
    ticket_client_id,
    ticket_device_id,
    ticket_type_id,
    coordinator_user_id,
    test_department_id,
    coordinator_user_id,
    ticket_reason_id,
    'Тестовая заявка: координатор назначил работу тестовому пользователю, и сейчас она находится в работе.',
    ticket_contact_id,
    regular_user_id,
    inwork_status_id,
    'Диагностика выполняется, клиент уведомлен.',
    FALSE
  )
  ON CONFLICT (id) DO UPDATE
  SET created_at = EXCLUDED.created_at,
      assigned_at = EXCLUDED.assigned_at,
      workstarted_at = EXCLUDED.workstarted_at,
      planned_start = EXCLUDED.planned_start,
      planned_end = EXCLUDED.planned_end,
      assigned_start = EXCLUDED.assigned_start,
      assigned_end = EXCLUDED.assigned_end,
      urgent = EXCLUDED.urgent,
      client = EXCLUDED.client,
      device = EXCLUDED.device,
      ticket_type = EXCLUDED.ticket_type,
      author = EXCLUDED.author,
      department = EXCLUDED.department,
      assigned_by = EXCLUDED.assigned_by,
      reason = EXCLUDED.reason,
      description = EXCLUDED.description,
      contact_person = EXCLUDED.contact_person,
      executor = EXCLUDED.executor,
      status = EXCLUDED.status,
      result = EXCLUDED.result,
      double_signed = EXCLUDED.double_signed;

  INSERT INTO tickets (
    id,
    created_at,
    assigned_at,
    workstarted_at,
    workfinished_at,
    planned_start,
    planned_end,
    assigned_start,
    assigned_end,
    urgent,
    closed_at,
    client,
    device,
    ticket_type,
    author,
    department,
    assigned_by,
    reason,
    description,
    contact_person,
    executor,
    status,
    result,
    double_signed
  )
  VALUES (
    closed_ticket_id,
    TIMESTAMP '2026-04-03 08:00:00',
    TIMESTAMP '2026-04-03 08:30:00',
    TIMESTAMP '2026-04-03 09:00:00',
    TIMESTAMP '2026-04-03 11:15:00',
    TIMESTAMP '2026-04-03 09:00:00',
    TIMESTAMP '2026-04-03 12:00:00',
    TIMESTAMP '2026-04-03 09:00:00',
    TIMESTAMP '2026-04-03 12:00:00',
    FALSE,
    TIMESTAMP '2026-04-03 11:45:00',
    ticket_client_id,
    ticket_device_id,
    ticket_type_id,
    regular_user_id,
    test_department_id,
    coordinator_user_id,
    ticket_reason_id,
    'Тестовая заявка: завершенный сервисный выезд для проверки сценария работы тестового отдела.',
    ticket_contact_id,
    regular_user_id,
    closed_status_id,
    'Работы успешно завершены. Прибор проверен, обслужен и возвращен в эксплуатацию.',
    FALSE
  )
  ON CONFLICT (id) DO UPDATE
  SET created_at = EXCLUDED.created_at,
      assigned_at = EXCLUDED.assigned_at,
      workstarted_at = EXCLUDED.workstarted_at,
      workfinished_at = EXCLUDED.workfinished_at,
      planned_start = EXCLUDED.planned_start,
      planned_end = EXCLUDED.planned_end,
      assigned_start = EXCLUDED.assigned_start,
      assigned_end = EXCLUDED.assigned_end,
      urgent = EXCLUDED.urgent,
      closed_at = EXCLUDED.closed_at,
      client = EXCLUDED.client,
      device = EXCLUDED.device,
      ticket_type = EXCLUDED.ticket_type,
      author = EXCLUDED.author,
      department = EXCLUDED.department,
      assigned_by = EXCLUDED.assigned_by,
      reason = EXCLUDED.reason,
      description = EXCLUDED.description,
      contact_person = EXCLUDED.contact_person,
      executor = EXCLUDED.executor,
      status = EXCLUDED.status,
      result = EXCLUDED.result,
      double_signed = EXCLUDED.double_signed;

  UPDATE users
  SET latest_ticket = created_ticket_id
  WHERE user_id IN (coordinator_user_id, regular_user_id);
END $$;
