"use strict";

const { ServiceBroker } = require("moleculer");
const SidecarService = require("moleculer-sidecar");
const { Pool } = require("pg");
const { createLatestTicketService, databasePoolOptions } = require("./latest-ticket");

const broker = new ServiceBroker({
	namespace: process.env.NAMESPACE || "",
	nodeID: process.env.NODEID || undefined,
	transporter: process.env.TRANSPORTER || null,
	serializer: process.env.SERIALIZER || "JSON",
	logger: true,
	logLevel: process.env.LOGLEVEL || "info"
});

const databasePool = new Pool(databasePoolOptions(process.env));

broker.createService(SidecarService);
broker.createService(createLatestTicketService(databasePool));

async function stop(signal) {
	broker.logger.info(`Received ${signal}; stopping Moleculer Sidecar...`);
	try {
		await broker.stop();
	} finally {
		await databasePool.end();
	}
}

process.once("SIGINT", () => void stop("SIGINT"));
process.once("SIGTERM", () => void stop("SIGTERM"));

broker.start().catch(err => {
	console.error("Unable to start Moleculer Sidecar", err);
	return databasePool.end().finally(() => {
		process.exitCode = 1;
	});
});
