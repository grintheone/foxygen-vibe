"use strict";

const { ServiceBroker } = require("moleculer");
const SidecarService = require("moleculer-sidecar");

const broker = new ServiceBroker({
	namespace: process.env.NAMESPACE || "",
	nodeID: process.env.NODEID || undefined,
	transporter: process.env.TRANSPORTER || null,
	serializer: process.env.SERIALIZER || "JSON",
	logger: true,
	logLevel: process.env.LOGLEVEL || "info"
});

broker.createService(SidecarService);

async function stop(signal) {
	broker.logger.info(`Received ${signal}; stopping Moleculer Sidecar...`);
	await broker.stop();
	process.exit(0);
}

process.once("SIGINT", () => void stop("SIGINT"));
process.once("SIGTERM", () => void stop("SIGTERM"));

broker.start().catch(err => {
	console.error("Unable to start Moleculer Sidecar", err);
	process.exit(1);
});
