import * as util from './util.js';
import * as client from './client.js';

export class T {
	client: client.Client;

	private baseURL: URL;
	private name: string;

	constructor(name: string) {
		this.client = new client.Client(util.getBaseURL());
		this.baseURL = new URL(util.getBaseURL(), globalThis?.location?.href);
		this.name = name;

		this.log(`getBaseURL()=${util.getBaseURL()}`);
	}

	async log(msg: string) {
		await this.logEvent('log', msg);
	}

	async logEvent(event: string, details?: string) {
		const params = new URLSearchParams();

		params.set('event', event);
		params.set('name', this.name);

		if (details) {
			params.set('details', details);
		}

		const url = new URL('_logEvent', this.baseURL);
		url.search = `?${params}`;

		const resp = await fetch(url);
		this.true(resp.ok, resp.statusText);
	}

	true(a: any, msg?: string): asserts a {
		if (!a) {
			this.fail(`not true(): ${msg ?? ''}`);
		}
	}

	fail(msg?: string) {
		throw new Error(`${msg ?? ''}`);
	}

	rejects(prom: Promise<any>, msg?: string) {
		prom.then(
			() => this.fail(`promise did not reject: ${msg ?? ''}`),
			() => {},
		);
	}

	equal(a: any, b: any, msg?: string) {
		const jsA = JSON.stringify(a);
		const jsB = JSON.stringify(b);

		if (jsA != jsB) {
			this.fail(`${jsA} != '${jsB}': ${msg ?? ''}`);
		}
	}
}

export async function def(name: string, cb: (t: T) => Promise<void>) {
	const t = new T(name);

	await t.logEvent('begin');

	try {
		await cb(t);
	} catch (e) {
		await t.logEvent('error', `${e}`);
		throw e;
	} finally {
		await t.logEvent('end');
	}
}
