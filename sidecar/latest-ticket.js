"use strict";

const LATEST_TICKET_QUERY = `
	SELECT
		t.id::text AS id,
		t.number,
		CASE
			WHEN t.created_at IS NULL THEN NULL
			ELSE to_char(t.created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
		END AS "createdAt",
		COALESCE(t.status, '') AS status,
		t.description,
		t.urgent,
		t.ticket_type AS "ticketType",
		t.reason,
		t.client::text AS "clientId",
		COALESCE(client.title, '') AS "clientName",
		t.device::text AS "deviceId",
		COALESCE(classificator.title, '') AS "deviceName",
		COALESCE(device.serial_number, '') AS "deviceSerialNumber",
		t.author::text AS "authorId",
		TRIM(CONCAT(COALESCE(author.first_name, ''), ' ', COALESCE(author.last_name, ''))) AS "authorName",
		t.executor::text AS "executorId",
		TRIM(CONCAT(COALESCE(executor.first_name, ''), ' ', COALESCE(executor.last_name, ''))) AS "executorName"
	FROM tickets AS t
	LEFT JOIN clients AS client ON client.id = t.client
	LEFT JOIN devices AS device ON device.id = t.device
	LEFT JOIN classificators AS classificator ON classificator.id = device.classificator
	LEFT JOIN users AS author ON author.user_id = t.author
	LEFT JOIN users AS executor ON executor.user_id = t.executor
	ORDER BY t.created_at DESC NULLS LAST, t.number DESC
	LIMIT 1
`;

function databasePoolOptions(environment) {
	if (environment.DATABASE_URL) {
		return {
			connectionString: environment.DATABASE_URL
		};
	}

	const requiredKeys = ["DB_HOST", "DB_PORT", "DB_USER", "DB_NAME"];
	const missingKeys = requiredKeys.filter(key => !environment[key]);
	if (missingKeys.length > 0) {
		throw new Error(`Missing sidecar database configuration: ${missingKeys.join(", ")}`);
	}

	const port = Number.parseInt(environment.DB_PORT, 10);
	if (!Number.isInteger(port) || port <= 0 || port > 65535) {
		throw new Error(`Invalid sidecar database port: ${environment.DB_PORT}`);
	}

	return {
		host: environment.DB_HOST,
		port,
		user: environment.DB_USER,
		password: environment.DB_PASSWORD || "",
		database: environment.DB_NAME,
		ssl: environment.DB_SSLMODE && environment.DB_SSLMODE !== "disable"
			? { rejectUnauthorized: environment.DB_SSLMODE === "verify-full" }
			: false
	};
}

async function findLatestTicket(pool) {
	const result = await pool.query(LATEST_TICKET_QUERY);
	return result.rows[0] || null;
}

function createLatestTicketService(pool) {
	return {
		name: "foxygen",
		actions: {
			latestTicket: {
				async handler() {
					return findLatestTicket(pool);
				}
			}
		}
	};
}

module.exports = {
	LATEST_TICKET_QUERY,
	createLatestTicketService,
	databasePoolOptions,
	findLatestTicket
};
