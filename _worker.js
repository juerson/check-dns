import { connect } from 'cloudflare:sockets';

const ENCODER = new TextEncoder();
const DECODER = new TextDecoder();

// PSK 配置
const PSK = typeof atob === 'function' ? atob('YUIzIzlka2Y4ITJqUXBMNHM4eFp5Vzd2MVVlUjBtTjI=') : 'aB3#9dkf8!2jQpL4s8xZyW7v1UeR0mN2';

// Cloudflare fallback IPs（base64 解码）
let CF_FALLBACK_IPS = [atob("UHJveHlJUC5DTUxpdXNzc3MubmV0")];

// v2 frame 定义
const MAGIC = 0xA5;
const VERSION = 2;
const CMD = { AUTH: 0x01, OPEN: 0x02, DATA: 0x03, CLOSE: 0x04 };

export default {
	async fetch(req, env, ctx) {
		try {
			// 从 env 动态覆盖 fallback
			if (Array.isArray(env.CF_FALLBACK_IPS)) CF_FALLBACK_IPS = env.CF_FALLBACK_IPS;
			else if (typeof env.CF_FALLBACK_IPS === 'string')
				CF_FALLBACK_IPS = env.CF_FALLBACK_IPS.split(',');

			const upgradeHeader = req.headers.get('Upgrade');
			if (!upgradeHeader || upgradeHeader.toLowerCase() !== 'websocket') {
				return new Response('Use WebSocket', { status: 400 });
			}

			const [client, server] = Object.values(new WebSocketPair());
			server.accept();

			handleSession(server);

			return new Response(null, { status: 101, webSocket: client });
		} catch (err) {
			return new Response(err.toString(), { status: 500 });
		}
	},
};

// --------------------- session handler ---------------------
async function handleSession(ws) {
	let remoteSocket, remoteWriter, remoteReader;
	let isClosed = false;

	const cleanup = () => {
		if (isClosed) return;
		isClosed = true;
		try { remoteReader?.cancel(); } catch { }
		try { remoteWriter?.releaseLock(); } catch { }
		try { remoteReader?.releaseLock(); } catch { }
		try { remoteSocket?.close(); } catch { }
		safeCloseWebSocket(ws);
	};

	const parseFrame = (buffer) => {
		if (buffer.byteLength < 8) return null;
		const view = new DataView(buffer);
		if (view.getUint8(0) !== MAGIC || view.getUint8(1) !== VERSION) return null;
		const cmd = view.getUint8(2);
		const flags = view.getUint8(3);
		const payload_len = view.getUint32(4, false);
		if (payload_len > 64 * 1024 || buffer.byteLength < 8 + payload_len) return null;
		const payload = buffer.slice(8, 8 + payload_len);
		return { cmd, flags, payload };
	};

	const pumpRemoteToWS = async () => {
		try {
			while (!isClosed && remoteReader) {
				const { done, value } = await remoteReader.read();
				if (done) break;
				if (ws.readyState !== WebSocket.OPEN) break;
				if (value?.byteLength > 0) ws.send(value);
			}
		} catch { }
		cleanup();
	};

	// ------------------- TCP 连接，支持 CF_FALLBACK_IPS -------------------
	const connectRemote = async (host, port, firstPayload) => {
		const attempts = [null, ...CF_FALLBACK_IPS]; // null 表示原 host
		for (let i = 0; i < attempts.length; i++) {
			try {
				const targetHost = attempts[i] || host;
				remoteSocket = connect({ hostname: targetHost, port });
				if (remoteSocket.opened) await remoteSocket.opened;

				remoteWriter = remoteSocket.writable.getWriter();
				remoteReader = remoteSocket.readable.getReader();

				if (firstPayload && firstPayload.byteLength > 0) {
					await remoteWriter.write(new Uint8Array(firstPayload));
				}

				pumpRemoteToWS();
				return; // 成功就返回
			} catch (err) {
				// 清理
				try { remoteReader?.cancel(); } catch { }
				try { remoteWriter?.releaseLock(); } catch { }
				try { remoteReader?.releaseLock(); } catch { }
				try { remoteSocket?.close(); } catch { }
				remoteWriter = remoteReader = remoteSocket = null;

				if (i === attempts.length - 1) throw err; // 最后一次失败才抛出
			}
		}
	};

	// ---------------- WebSocket 事件 ----------------
	ws.addEventListener('message', async (evt) => {
		if (isClosed) return;
		try {
			const buffer = evt.data instanceof ArrayBuffer ? evt.data : ENCODER.encode(evt.data);
			const frame = parseFrame(buffer);
			if (!frame) {
				cleanup();
				return;
			}

			const { cmd, payload } = frame;

			if (cmd === CMD.AUTH) {
				const key = DECODER.decode(payload);
				if (key !== PSK) {
					ws.send(makeFrame(CMD.CLOSE, 0, ENCODER.encode('PSK fail')));
					cleanup();
				}
			} else if (cmd === CMD.OPEN) {
				const text = DECODER.decode(payload);
				const sep = text.indexOf('|');
				const addr = sep !== -1 ? text.substring(0, sep) : text;
				const firstPayload = sep !== -1 ? ENCODER.encode(text.substring(sep + 1)) : null;
				const portSep = addr.lastIndexOf(':');
				const host = addr.substring(0, portSep);
				const port = parseInt(addr.substring(portSep + 1), 10);

				await connectRemote(host, port, firstPayload);
			} else if (cmd === CMD.DATA) {
				if (remoteWriter) await remoteWriter.write(new Uint8Array(payload));
			} else if (cmd === CMD.CLOSE) {
				cleanup();
			}
		} catch {
			cleanup();
		}
	});

	ws.addEventListener('close', cleanup);
	ws.addEventListener('error', cleanup);
}

// --------------------- utils ---------------------
function safeCloseWebSocket(ws) {
	try {
		if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CLOSING) {
			ws.close(1000, 'Server closed');
		}
	} catch { }
}

function makeFrame(cmd, flags = 0, payload = new Uint8Array(0)) {
	const buffer = new ArrayBuffer(8 + payload.byteLength);
	const view = new DataView(buffer);
	view.setUint8(0, MAGIC);
	view.setUint8(1, VERSION);
	view.setUint8(2, cmd);
	view.setUint8(3, flags);
	view.setUint32(4, payload.byteLength, false);
	new Uint8Array(buffer, 8).set(payload);
	return buffer;
}
