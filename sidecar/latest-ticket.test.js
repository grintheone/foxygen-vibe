"use strict";

const assert = require("node:assert/strict");
const test = require("node:test");
const { ServiceBroker } = require("moleculer");

const {
	LATEST_TICKET_QUERY,
	createLatestTicketService,
	databasePoolOptions,
	findLatestTicket
} = require("./latest-ticket");

test("databasePoolOptions builds split PostgreSQL configuration", () => {
	assert.deepEqual(databasePoolOptions({
		DB_HOST: "postgres",
		DB_PORT: "5432",
		DB_USER: "foxygen",
		DB_PASSWORD: "secret",
		DB_NAME: "foxygendb",
		DB_SSLMODE: "disable"
	}), {
		host: "postgres",
		port: 5432,
		user: "foxygen",
		password: "secret",
		database: "foxygendb",
		ssl: false
	});
});

test("databasePoolOptions prefers DATABASE_URL", () => {
	assert.deepEqual(databasePoolOptions({
		DATABASE_URL: "postgres://foxygen:secret@postgres:5432/foxygendb"
	}), {
		connectionString: "postgres://foxygen:secret@postgres:5432/foxygendb"
	});
});

test("findLatestTicket returns the newest row", async () => {
	const expected = {
		id: "9ab1544d-7bb5-40c1-b30e-d59a842b860f",
		number: 42,
		status: "created"
	};
	const pool = {
		async query(query) {
			assert.equal(query, LATEST_TICKET_QUERY);
			return { rows: [expected] };
		}
	};

	assert.equal(await findLatestTicket(pool), expected);
});

test("findLatestTicket returns null when there are no tickets", async () => {
	const pool = {
		async query() {
			return { rows: [] };
		}
	};

	assert.equal(await findLatestTicket(pool), null);
});

test("foxygen.latestTicket action uses the ticket query", async () => {
	const expected = { id: "ticket-id", number: 7 };
	const pool = {
		async query() {
			return { rows: [expected] };
		}
	};
	const service = createLatestTicketService(pool);

	assert.equal(service.name, "foxygen");
	assert.equal(await service.actions.latestTicket.handler(), expected);
});

test("foxygen.latestTicket is callable through a Moleculer broker", async () => {
	const expected = { id: "ticket-id", number: 8 };
	const pool = {
		async query() {
			return { rows: [expected] };
		}
	};
	const broker = new ServiceBroker({ logger: false, transporter: null });
	broker.createService(createLatestTicketService(pool));

	await broker.start();
	try {
		assert.deepEqual(await broker.call("foxygen.latestTicket"), expected);
	} finally {
		await broker.stop();
	}
});
