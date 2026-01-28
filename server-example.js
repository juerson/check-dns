// Example JavaScript WebSocket Server implementing v2 protocol
// This is a reference implementation showing how the Go client should interact with a JavaScript server

// Protocol v2 constants
const MAGIC = 0xA5;
const VERSION = 2;

const CMD_AUTH = 0x01;
const CMD_OPEN = 0x02;
const CMD_DATA = 0x03;
const CMD_CLOSE = 0x04;

const FLAG_NONE = 0x00;

// Configuration
const AUTH_TOKEN = process.env.AUTH_TOKEN || 'your-secret-token';

/**
 * Make a v2 protocol frame
 * @param {number} cmd - Command byte
 * @param {number} flags - Flags byte
 * @param {Buffer} payload - Payload data
 * @returns {Buffer} Complete frame
 */
function makeFrame(cmd, flags, payload) {
	const payloadLen = payload ? payload.length : 0;
	const frame = Buffer.allocUnsafe(8 + payloadLen);

	frame[0] = MAGIC;
	frame[1] = VERSION;
	frame[2] = cmd;
	frame[3] = flags;
	frame.writeUInt32BE(payloadLen, 4);

	if (payload && payloadLen > 0) {
		payload.copy(frame, 8);
	}

	return frame;
}

/**
 * Parse a v2 protocol frame
 * @param {Buffer} buffer - Frame data
 * @returns {Object|null} Parsed frame or null if invalid
 */
function parseFrame(buffer) {
	if (buffer.length < 8) {
		console.error('Frame too short');
		return null;
	}

	const magic = buffer[0];
	if (magic !== MAGIC) {
		console.error(`Invalid MAGIC: expected 0x${MAGIC.toString(16)}, got 0x${magic.toString(16)}`);
		return null;
	}

	const version = buffer[1];
	if (version !== VERSION) {
		console.error(`Unsupported VERSION: expected ${VERSION}, got ${version}`);
		return null;
	}

	const cmd = buffer[2];
	const flags = buffer[3];
	const payloadLen = buffer.readUInt32BE(4);

	if (buffer.length < 8 + payloadLen) {
		console.error(`Incomplete frame: expected ${8 + payloadLen} bytes, got ${buffer.length}`);
		return null;
	}

	const payload = payloadLen > 0 ? buffer.slice(8, 8 + payloadLen) : null;

	return { cmd, flags, payload };
}

/**
 * Handle WebSocket connection
 * @param {WebSocket} ws - WebSocket connection
 */
function handleConnection(ws) {
	console.log('New connection established');
	let authenticated = false;
	let targetConn = null;

	ws.on('message', async (data) => {
		const frame = parseFrame(data);
		if (!frame) {
			console.error('Invalid frame received, closing connection');
			ws.close();
			return;
		}

		const { cmd, flags, payload } = frame;

		switch (cmd) {
			case CMD_AUTH:
				// Handle authentication
				const token = payload ? payload.toString('utf8') : '';
				console.log(`AUTH frame received with token: ${token.substring(0, 10)}...`);

				if (token === AUTH_TOKEN) {
					authenticated = true;
					console.log('Authentication successful');
					// Send AUTH success (could be any non-CLOSE frame, we'll send back AUTH)
					ws.send(makeFrame(CMD_AUTH, FLAG_NONE, Buffer.from('OK')));
				} else {
					console.log('Authentication failed');
					ws.send(makeFrame(CMD_CLOSE, FLAG_NONE, Buffer.from('AUTH_FAILED')));
					ws.close();
				}
				break;

			case CMD_OPEN:
				// Handle connection open request
				if (!authenticated) {
					console.error('OPEN received before authentication');
					ws.send(makeFrame(CMD_CLOSE, FLAG_NONE, Buffer.from('NOT_AUTHENTICATED')));
					ws.close();
					return;
				}

				const openData = payload.toString('utf8');
				console.log(`OPEN frame received: ${openData}`);

				// Parse "host:port|firstPayload" format
				let target, firstPayload;
				const pipeIndex = openData.indexOf('|');
				if (pipeIndex !== -1) {
					target = openData.substring(0, pipeIndex);
					firstPayload = Buffer.from(openData.substring(pipeIndex + 1));
				} else {
					target = openData;
					firstPayload = null;
				}

				console.log(`Connecting to target: ${target}`);

				// In a real implementation, you would establish connection to target
				// For this example, we'll just acknowledge
				try {
					// Simulated connection to target
					// const net = require('net');
					// targetConn = net.connect(...)

					console.log(`Connected to ${target}`);
					ws.send(makeFrame(CMD_OPEN, FLAG_NONE, Buffer.from('OK')));

					// If there was firstPayload, send it to target
					if (firstPayload && firstPayload.length > 0) {
						console.log(`Forwarding first payload (${firstPayload.length} bytes) to target`);
						// targetConn.write(firstPayload);
					}
				} catch (error) {
					console.error(`Failed to connect to ${target}:`, error);
					ws.send(makeFrame(CMD_CLOSE, FLAG_NONE, Buffer.from('CONNECT_FAILED')));
					ws.close();
				}
				break;

			case CMD_DATA:
				// Handle data transfer
				if (!authenticated) {
					console.error('DATA received before authentication');
					ws.close();
					return;
				}

				console.log(`DATA frame received: ${payload ? payload.length : 0} bytes`);

				// In a real implementation, forward data to target connection
				// targetConn.write(payload);

				// For demonstration, echo data back
				// ws.send(makeFrame(CMD_DATA, FLAG_NONE, payload));
				break;

			case CMD_CLOSE:
				// Handle close request
				console.log('CLOSE frame received');
				if (targetConn) {
					// targetConn.end();
				}
				ws.close();
				break;

			default:
				console.error(`Unknown command: 0x${cmd.toString(16)}`);
				ws.send(makeFrame(CMD_CLOSE, FLAG_NONE, Buffer.from('UNKNOWN_CMD')));
				ws.close();
		}
	});

	ws.on('close', () => {
		console.log('Connection closed');
		if (targetConn) {
			// targetConn.end();
		}
	});

	ws.on('error', (error) => {
		console.error('WebSocket error:', error);
		if (targetConn) {
			// targetConn.end();
		}
	});
}

// Example usage with Cloudflare Workers
export default {
	async fetch(request, env, ctx) {
		// Check if this is a WebSocket upgrade request
		const upgradeHeader = request.headers.get('Upgrade');
		if (upgradeHeader !== 'websocket') {
			return new Response('Expected WebSocket connection', { status: 426 });
		}

		// Accept WebSocket connection
		const pair = new WebSocketPair();
		const [client, server] = Object.values(pair);

		// Handle the connection
		server.accept();
		handleConnection(server);

		// Return the client side to the browser
		return new Response(null, {
			status: 101,
			webSocket: client,
		});
	},
};

// For Node.js testing
if (typeof module !== 'undefined' && module.exports) {
	module.exports = { makeFrame, parseFrame, handleConnection };
}
